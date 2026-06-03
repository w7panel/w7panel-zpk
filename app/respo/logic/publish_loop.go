package logic

import (
	"fmt"
	"hash/fnv"
	"log/slog"
	"slices"
	"sync"

	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
)

const (
	FormulaPublishStatusPending = int32(1)
	FormulaPublishStatusSuccess = int32(2)
	FormulaPublishStatusFail    = int32(3)
	formulaPublishWorkerCount   = 5
)

type formulaPublishTask struct {
	formulaIdentify string
	VersionName     string
	VersionId       int32
}

type FormulaPublishLoop struct {
	once    sync.Once
	queues  []chan formulaPublishTask
	pending sync.Map
}

var defaultFormulaPublishLoop = &FormulaPublishLoop{
	queues: makeFormulaPublishQueues(),
}

func InitFormulaPublishLoop() {
	defaultFormulaPublishLoop.Start()
}

func AddFormulaPublishTask(formulaIdentify, versionName string, versionId int32) error {
	return defaultFormulaPublishLoop.Add(formulaIdentify, versionName, versionId)
}

func RecoverFormulaPublishTasks() {
	go func() {
		err := defaultFormulaPublishLoop.Recover()
		if err != nil {
			panic(err)
		}
	}()
}

func (l *FormulaPublishLoop) Start() {
	l.once.Do(func() {
		for i, queue := range l.queues {
			go l.run(i+1, queue)
		}
	})
}

func (l *FormulaPublishLoop) Add(formulaIdentify, versionName string, versionId int32) error {
	task := formulaPublishTask{
		formulaIdentify: formulaIdentify,
		VersionName:     versionName,
		VersionId:       versionId,
	}
	if _, loaded := l.pending.LoadOrStore(l.taskKey(task), struct{}{}); loaded {
		return nil
	}

	err := updateVersionPublishState(nil, versionId, FormulaPublishStatusPending, "")
	if err != nil {
		l.pending.Delete(l.taskKey(task))
		return err
	}

	l.queueFor(task) <- task
	return nil
}

func (l *FormulaPublishLoop) Recover() error {
	versions, err := dao.Q.Version.Where(dao.Q.Version.PublishStatus.Eq(FormulaPublishStatusPending)).Find()
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
			slog.Warn("skip recover formula publish task because formula not found", "formula_id", version.FormulaID, "version_id", version.ID)
			continue
		}

		if err := l.Add(formulaName, version.Name, version.ID); err != nil {
			return err
		}
	}

	if len(versions) > 0 {
		slog.Info("recover formula publish tasks", "count", len(versions))
	}
	return nil
}

func (l *FormulaPublishLoop) run(workerID int, queue <-chan formulaPublishTask) {
	for task := range queue {
		slog.Info("run formula publish task begin", "worker_id", workerID, "task", task)
		err := l.handle(task)
		slog.Info("run formula publish task complete", "worker_id", workerID, "task", task, "err", err)
		if err != nil {
			err1 := updateVersionPublishState(nil, task.VersionId, FormulaPublishStatusFail, err.Error())
			slog.Error("formula publish task failed", "worker_id", workerID, "task", task, "err", err, "err1", err1)
		}

		l.pending.Delete(l.taskKey(task))
	}
}

func (l *FormulaPublishLoop) handle(task formulaPublishTask) error {
	depot, _ := NewDepot()
	formula, err := depot.GetFormula(task.formulaIdentify, task.VersionName, nil)
	if err != nil {
		return err
	}

	if _, err = PackFormulaToHelmAndPack(*formula, true); err != nil {
		return err
	}

	err = updateVersionPublishState(formula, task.VersionId, FormulaPublishStatusSuccess, "")
	if err != nil {
		return err
	}

	return nil
}

func (l *FormulaPublishLoop) taskKey(task formulaPublishTask) string {
	return fmt.Sprintf("%s:%s", task.formulaIdentify, task.VersionName)
}

func (l *FormulaPublishLoop) queueFor(task formulaPublishTask) chan formulaPublishTask {
	if len(l.queues) == 0 {
		panic("formula publish queues not initialized")
	}

	index := hashFormulaIdentify(task.formulaIdentify) % uint32(len(l.queues))
	return l.queues[index]
}

func makeFormulaPublishQueues() []chan formulaPublishTask {
	queues := make([]chan formulaPublishTask, formulaPublishWorkerCount)
	for i := range queues {
		queues[i] = make(chan formulaPublishTask, 128)
	}
	return queues
}

func hashFormulaIdentify(formulaIdentify string) uint32 {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(formulaIdentify))
	return hasher.Sum32()
}

func updateVersionPublishState(formula *Formula, versionID int32, status int32, failReason string) error {
	return dao.Q.Transaction(func(tx *dao.Query) error {
		if formula != nil && status == FormulaPublishStatusSuccess {
			updateFormula := entity.Formula{
				VersionLatestID: versionID,
			}
			if formula.AuditStatus == FORMULA_AUDIT_FAIL {
				updateFormula.AuditStatus = FOEMULA_AUDIT_ING
			}
			_, err := tx.Formula.Where(tx.Formula.ID.Eq(formula.ID)).Updates(updateFormula)
			if err != nil {
				return err
			}
		}

		_, err := tx.Version.Where(tx.Version.ID.Eq(versionID)).Updates(entity.Version{
			PublishStatus:     status,
			PublishFailReason: failReason,
		})

		return err
	})
}
