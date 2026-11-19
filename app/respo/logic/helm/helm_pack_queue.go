package helm

import (
	"fmt"
	"hash/fnv"
	"log/slog"
	"slices"
	"sync"

	formulalogic "github.com/w7panel/w7panel-zpk/app/respo/logic/formula"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
)

const (
	formulaHelmPackWorkerCount = 5
)

type formulaHelmPackTask struct {
	formulaIdentify string
	VersionName     string
	VersionId       int32
}

type FormulaHelmPackQueue struct {
	once    sync.Once
	queues  []chan formulaHelmPackTask
	pending sync.Map
}

var defaultFormulaHelmPackQueue = &FormulaHelmPackQueue{
	queues: makeFormulaHelmPackQueues(),
}

func InitFormulaHelmPackQueue() {
	defaultFormulaHelmPackQueue.Start()
}

func AddFormulaHelmPackTask(formulaIdentify, versionName string, versionId int32) error {
	return defaultFormulaHelmPackQueue.Add(formulaIdentify, versionName, versionId)
}

func IsFormulaHelmPackPending(formulaIdentify, versionName string) bool {
	task := formulaHelmPackTask{formulaIdentify: formulaIdentify, VersionName: versionName}
	_, pending := defaultFormulaHelmPackQueue.pending.Load(defaultFormulaHelmPackQueue.taskKey(task))
	return pending
}

func RecoverFormulaHelmPackTasks() {
	go func() {
		err := defaultFormulaHelmPackQueue.Recover()
		if err != nil {
			panic(err)
		}
	}()
}

func (l *FormulaHelmPackQueue) Start() {
	l.once.Do(func() {
		for i, queue := range l.queues {
			go l.run(i+1, queue)
		}
	})
}

func (l *FormulaHelmPackQueue) Add(formulaIdentify, versionName string, versionId int32) error {
	task := formulaHelmPackTask{
		formulaIdentify: formulaIdentify,
		VersionName:     versionName,
		VersionId:       versionId,
	}
	if _, loaded := l.pending.LoadOrStore(l.taskKey(task), struct{}{}); loaded {
		return nil
	}

	err := updateVersionPublishState(nil, versionId, formulalogic.FormulaPublishStatusPending, "")
	if err != nil {
		l.pending.Delete(l.taskKey(task))
		return err
	}

	l.queueFor(task) <- task
	return nil
}

func (l *FormulaHelmPackQueue) Recover() error {
	versions, err := dao.Q.Version.Where(dao.Q.Version.PublishStatus.Eq(formulalogic.FormulaPublishStatusPending)).Find()
	if err != nil {
		return err
	}

	formulaIDs := make([]int32, 0, len(versions))
	for _, version := range versions {
		if slices.Contains(formulaIDs, version.FormulaID) {
			continue
		}
		formulaIDs = append(formulaIDs, version.FormulaID)
	}

	formulaNameByID := make(map[int32]string, len(formulaIDs))
	if len(formulaIDs) > 0 {
		formulas, err := dao.Q.Formula.Select(dao.Q.Formula.ID, dao.Q.Formula.Name).Where(dao.Q.Formula.ID.In(formulaIDs...)).Find()
		if err != nil {
			return err
		}
		for _, formula := range formulas {
			formulaNameByID[formula.ID] = formula.Name
		}
	}

	for _, version := range versions {
		formulaName, ok := formulaNameByID[version.FormulaID]
		if !ok {
			slog.Warn("skip recover formula Helm pack task because formula not found", "formula_id", version.FormulaID, "version_id", version.ID)
			continue
		}

		if err := l.Add(formulaName, version.Name, version.ID); err != nil {
			return err
		}
	}

	if len(versions) > 0 {
		slog.Info("recover formula Helm pack tasks", "count", len(versions))
	}
	return nil
}

func (l *FormulaHelmPackQueue) run(workerID int, queue <-chan formulaHelmPackTask) {
	for task := range queue {
		slog.Info("run formula Helm pack task begin", "worker_id", workerID, "task", task)
		err := l.handle(task)
		slog.Info("run formula Helm pack task complete", "worker_id", workerID, "task", task, "err", err)
		if err != nil {
			err1 := updateVersionPublishState(nil, task.VersionId, formulalogic.FormulaPublishStatusFail, err.Error())
			slog.Error("formula Helm pack task failed", "worker_id", workerID, "task", task, "err", err, "err1", err1)
		}

		l.pending.Delete(l.taskKey(task))
	}
}

func (l *FormulaHelmPackQueue) handle(task formulaHelmPackTask) error {
	depot, _ := formulalogic.NewDepot()
	formula, err := depot.GetFormula(task.formulaIdentify, task.VersionName, nil)
	if err != nil {
		return err
	}

	if _, err = PackFormulaToHelmAndPack(*formula, true); err != nil {
		return err
	}

	err = updateVersionPublishState(formula, task.VersionId, formulalogic.FormulaPublishStatusSuccess, "")
	if err != nil {
		return err
	}

	return nil
}

func (l *FormulaHelmPackQueue) taskKey(task formulaHelmPackTask) string {
	return fmt.Sprintf("%s:%s", task.formulaIdentify, task.VersionName)
}

func (l *FormulaHelmPackQueue) queueFor(task formulaHelmPackTask) chan formulaHelmPackTask {
	if len(l.queues) == 0 {
		panic("formula Helm pack queues not initialized")
	}

	index := hashFormulaIdentify(task.formulaIdentify) % uint32(len(l.queues))
	return l.queues[index]
}

func makeFormulaHelmPackQueues() []chan formulaHelmPackTask {
	queues := make([]chan formulaHelmPackTask, formulaHelmPackWorkerCount)
	for i := range queues {
		queues[i] = make(chan formulaHelmPackTask, 128)
	}
	return queues
}

func hashFormulaIdentify(formulaIdentify string) uint32 {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(formulaIdentify))
	return hasher.Sum32()
}

func updateVersionPublishState(formula *formulalogic.Formula, versionID int32, status int32, failReason string) error {
	return dao.Q.Transaction(func(tx *dao.Query) error {
		if formula != nil && status == formulalogic.FormulaPublishStatusSuccess {
			updateFormula := entity.Formula{
				VersionLatestID: versionID,
			}
			_, err := tx.Formula.Where(tx.Formula.ID.Eq(formula.ID)).Updates(updateFormula)
			if err != nil {
				return err
			}
		}

		_, err := tx.Version.Where(tx.Version.ID.Eq(versionID)).
			Select(tx.Version.PublishStatus, tx.Version.PublishFailReason).Updates(entity.Version{
			PublishStatus:     status,
			PublishFailReason: failReason,
		})

		return err
	})
}
