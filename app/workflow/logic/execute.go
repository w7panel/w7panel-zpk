package logic

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"helm.sh/helm/v3/pkg/release"
	"k8s.io/client-go/kubernetes"
)

type Executor struct {
	Workflow
	once  sync.Once
	slots chan struct{}
}

var defaultExecutor = &Executor{}

func InitExecutor() { defaultExecutor.Start() }

func (l *Executor) Start() {
	l.once.Do(func() {
		if !facade.GetConfig().GetBool("workflow.enabled") {
			return
		}
		l.slots = make(chan struct{}, 2)
		go func() {
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				if err := l.writable(); err != nil {
					continue
				}
				if err := l.Dispatch(); err != nil {
					slog.Error("workflow dispatch", "error", err)
				}
			}
		}()
	})
}

func (l *Executor) Dispatch() error {
	select {
	case l.slots <- struct{}{}:
	default:
		return nil
	}
	launched := false
	defer func() {
		if !launched {
			<-l.slots
		}
	}()
	if err := l.ExpireInterrupted(); err != nil {
		return err
	}
	if err := l.cleanupRunCharts(); err != nil {
		slog.Error("cleanup workflow charts", "error", err)
	}
	run, err := l.claimRun()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if run.WorkflowRunID == "" {
		return nil
	}
	launched = true
	go func() {
		defer func() { <-l.slots }()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
		defer cancel()
		err := l.Execute(ctx, run)
		status, message := "succeeded", ""
		if err != nil {
			status = "failed"
			message = err.Error()
			if len(message) > 2048 {
				message = message[:2048]
			}
			slog.Error("workflow execution", "run", run.WorkflowRunID, "error", err)
		}
		if e := l.writable(); e != nil {
			return
		}
		query := dao.Q.RegistryRepositoryDeployRuleMatchLog
		if _, e := query.Where(query.WorkflowRunID.Eq(run.WorkflowRunID), query.WorkflowTaskKey.Neq(""), query.ExecutionStatus.Eq("running")).UpdateSimple(query.UpdatedAt.Value(time.Now()), query.ExecutionStatus.Value("unknown"), query.FailureReason.Value("流程已结束，未能确认本任务的执行结果，请查看任务日志")); e != nil {
			slog.Error("save unresolved workflow tasks", "run", run.WorkflowRunID, "error", e)
		}
		if _, e := query.Where(query.WorkflowRunID.Eq(run.WorkflowRunID), query.WorkflowTaskKey.Eq(""), query.ExecutionStatus.Eq("running")).UpdateSimple(query.UpdatedAt.Value(time.Now()), query.ExecutionStatus.Value(status), query.FailureReason.Value(message), query.K8sLog.Value("Helm 执行实例 wf-"+run.WorkflowRunID+": "+status+" "+message)); e != nil {
			slog.Error("save workflow result", "run", run.WorkflowRunID, "error", e)
		}
	}()
	return nil
}

func (l *Executor) Execute(ctx context.Context, run entity.RegistryRepositoryDeployRuleMatchLog) error {
	defer func() {
		if err := (Helm{}).RemoveRunChart(run.RepositoryID, run.WorkflowRunID); err != nil {
			slog.Error("cleanup workflow run chart", "run", run.WorkflowRunID, "error", err)
		}
	}()
	snapshot, err := l.decodeExecutionSnapshot(run.EncryptedConfigSnapshot)
	if err != nil {
		return err
	}
	config, image := snapshot.Config, snapshot.Image
	if image.Reference == "" || image.Repository == "" || image.Host == "" {
		return errors.New("执行记录缺少源镜像信息，请重新触发")
	}
	selected, err := l.selectedTasks(ctx, run)
	if err != nil {
		return err
	}
	if len(selected) == 0 {
		return errors.New("执行记录没有匹配任务")
	}
	remote := map[string]bool{}
	kubeconfigs := map[string]string{}
	for _, task := range config.Tasks {
		if !selected[task.ID] {
			continue
		}
		if task.Action == ActionDeploy && task.K8sConfig != "" {
			remote[task.ID] = true
			kubeconfigs[task.ID] = task.K8sConfig
		}
		if task.Action == ActionBuild {
			token := snapshot.RegistryToken
			if token == "" {
				return errors.New("执行记录缺少镜像仓库令牌，请重新触发")
			}
		}
	}
	values := map[string]interface{}{
		"selected": selected, "remote": remote, "kubeconfigs": kubeconfigs, "registryToken": snapshot.RegistryToken,
		"run": map[string]interface{}{"original": image.Reference, "repository": image.Repository, "host": image.Host, "tag": run.TriggerTag, "plainHTTP": image.PlainHTTP},
	}
	for _, pair := range []struct{ key, config string }{{"buildImage", "workflow.build_image"}, {"deployImage", "workflow.deploy_image"}} {
		if image := facade.GetConfig().GetString(pair.config); image != "" {
			values[pair.key] = image
		}
	}
	return (Helm{}).Install(ctx, run, config, values, func(client kubernetes.Interface, installed *release.Release) error {
		return l.persistTaskResults(client, run, installed)
	})
}

func (l *Executor) claimRun() (entity.RegistryRepositoryDeployRuleMatchLog, error) {
	var run entity.RegistryRepositoryDeployRuleMatchLog
	// Claim at most one active run per repository using workflow records only.
	err := dao.Q.Transaction(func(tx *dao.Query) error {
		query := tx.RegistryRepositoryDeployRuleMatchLog
		pending, err := query.Where(query.ExecutionStatus.Eq("pending"), query.WorkflowTaskKey.Eq(""), query.Columns(query.RepositoryID).NotIn(query.Select(query.RepositoryID).Where(query.ExecutionStatus.Eq("running")))).Order(query.CreatedAt, query.ID).Limit(1).Find()
		if err != nil {
			return err
		}
		if len(pending) == 0 {
			return gorm.ErrRecordNotFound
		}
		run = *pending[0]

		// All dispatchers for this repository lock the same workflow coordinator row.
		// Locking reads below see committed state even under MySQL repeatable read.
		if _, err := query.Clauses(clause.Locking{Strength: "UPDATE"}).Where(query.RepositoryID.Eq(run.RepositoryID), query.WorkflowTaskKey.Eq("")).Order(query.ID).First(); err != nil {
			return err
		}
		active, err := query.Clauses(clause.Locking{Strength: "UPDATE"}).Where(query.RepositoryID.Eq(run.RepositoryID), query.WorkflowTaskKey.Eq(""), query.ExecutionStatus.Eq("running")).Limit(1).Find()
		if err != nil {
			return err
		}
		if len(active) > 0 {
			return gorm.ErrRecordNotFound
		}
		result, err := query.Where(query.WorkflowRunID.Eq(run.WorkflowRunID), query.ExecutionStatus.Eq("pending")).UpdateSimple(query.UpdatedAt.Value(time.Now()), query.ExecutionStatus.Value("running"))
		if err != nil {
			return err
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
	return run, err
}
