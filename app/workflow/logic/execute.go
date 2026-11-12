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
				if err := l.checkWritesAllowed(); err != nil {
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
	executionStarted := false
	defer func() {
		if !executionStarted {
			<-l.slots
		}
	}()
	if err := l.ExpireInterrupted(); err != nil {
		return err
	}
	if err := l.cleanupRunCharts(); err != nil {
		slog.Error("cleanup workflow charts", "error", err)
	}
	run, err := l.claimNextPendingRun()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if run.WorkflowRunID == "" {
		return nil
	}
	executionStarted = true
	go func() {
		defer func() { <-l.slots }()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
		defer cancel()
		err := l.Execute(ctx, run)
		runStatus, failureReason := "succeeded", ""
		if err != nil {
			runStatus = "failed"
			failureReason = err.Error()
			if len(failureReason) > 2048 {
				failureReason = failureReason[:2048]
			}
			slog.Error("workflow execution", "run", run.WorkflowRunID, "error", err)
		}
		if e := l.checkWritesAllowed(); e != nil {
			return
		}
		query := dao.Q.RegistryRepositoryDeployRuleMatchLog
		if _, e := query.Where(query.WorkflowRunID.Eq(run.WorkflowRunID), query.WorkflowTaskKey.Neq(""), query.ExecutionStatus.Eq("running")).UpdateSimple(query.UpdatedAt.Value(time.Now()), query.ExecutionStatus.Value("unknown"), query.FailureReason.Value("流程已结束，未能确认本任务的执行结果，请查看任务日志")); e != nil {
			slog.Error("save unresolved workflow tasks", "run", run.WorkflowRunID, "error", e)
		}
		if _, e := query.Where(query.WorkflowRunID.Eq(run.WorkflowRunID), query.WorkflowTaskKey.Eq(""), query.ExecutionStatus.Eq("running")).UpdateSimple(query.UpdatedAt.Value(time.Now()), query.ExecutionStatus.Value(runStatus), query.FailureReason.Value(failureReason), query.K8sLog.Value("Helm 执行实例 wf-"+run.WorkflowRunID+": "+runStatus+" "+failureReason)); e != nil {
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
	executionSnapshot, err := l.decodeExecutionSnapshot(run.EncryptedConfigSnapshot)
	if err != nil {
		return err
	}
	config, sourceImage := executionSnapshot.Config, executionSnapshot.Image
	if sourceImage.Reference == "" || sourceImage.Repository == "" || sourceImage.Host == "" {
		return errors.New("执行记录缺少源镜像信息，请重新触发")
	}
	selectedTaskIDs, err := l.selectedTaskIDs(ctx, run)
	if err != nil {
		return err
	}
	if len(selectedTaskIDs) == 0 {
		return errors.New("执行记录没有匹配任务")
	}
	remoteDeployTaskIDs := map[string]bool{}
	targetKubeconfigsByTaskID := map[string]string{}
	for _, task := range config.Tasks {
		if !selectedTaskIDs[task.ID] {
			continue
		}
		if task.Action == ActionDeploy && task.K8sConfig != "" {
			remoteDeployTaskIDs[task.ID] = true
			targetKubeconfigsByTaskID[task.ID] = task.K8sConfig
		}
		if task.Action == ActionBuild {
			token := executionSnapshot.RegistryToken
			if token == "" {
				return errors.New("执行记录缺少镜像仓库令牌，请重新触发")
			}
		}
	}
	executionValues := map[string]interface{}{
		"selected": selectedTaskIDs, "remote": remoteDeployTaskIDs, "kubeconfigs": targetKubeconfigsByTaskID, "registryToken": executionSnapshot.RegistryToken,
		"run": map[string]interface{}{"original": sourceImage.Reference, "repository": sourceImage.Repository, "host": sourceImage.Host, "tag": run.TriggerTag, "plainHTTP": sourceImage.PlainHTTP},
	}
	for _, pair := range []struct{ key, config string }{{"buildImage", "workflow.build_image"}, {"deployImage", "workflow.deploy_image"}} {
		if executorImage := facade.GetConfig().GetString(pair.config); executorImage != "" {
			executionValues[pair.key] = executorImage
		}
	}
	return (Helm{}).Install(ctx, run, config, executionValues, func(client kubernetes.Interface, helmRelease *release.Release) error {
		return l.persistTaskResults(client, run, helmRelease)
	})
}

func (l *Executor) claimNextPendingRun() (entity.RegistryRepositoryDeployRuleMatchLog, error) {
	var run entity.RegistryRepositoryDeployRuleMatchLog
	// Claim at most one active run per repository using workflow records only.
	err := dao.Q.Transaction(func(tx *dao.Query) error {
		query := tx.RegistryRepositoryDeployRuleMatchLog
		pendingRuns, err := query.Where(query.ExecutionStatus.Eq("pending"), query.WorkflowTaskKey.Eq(""), query.Columns(query.RepositoryID).NotIn(query.Select(query.RepositoryID).Where(query.ExecutionStatus.Eq("running")))).Order(query.CreatedAt, query.ID).Limit(1).Find()
		if err != nil {
			return err
		}
		if len(pendingRuns) == 0 {
			return gorm.ErrRecordNotFound
		}
		run = *pendingRuns[0]

		// All dispatchers for this repository lock the same workflow coordinator row.
		// Locking reads below see committed state even under MySQL repeatable read.
		if _, err := query.Clauses(clause.Locking{Strength: "UPDATE"}).Where(query.RepositoryID.Eq(run.RepositoryID), query.WorkflowTaskKey.Eq("")).Order(query.ID).First(); err != nil {
			return err
		}
		activeRuns, err := query.Clauses(clause.Locking{Strength: "UPDATE"}).Where(query.RepositoryID.Eq(run.RepositoryID), query.WorkflowTaskKey.Eq(""), query.ExecutionStatus.Eq("running")).Limit(1).Find()
		if err != nil {
			return err
		}
		if len(activeRuns) > 0 {
			return gorm.ErrRecordNotFound
		}
		claimResult, err := query.Where(query.WorkflowRunID.Eq(run.WorkflowRunID), query.ExecutionStatus.Eq("pending")).UpdateSimple(query.UpdatedAt.Value(time.Now()), query.ExecutionStatus.Value("running"))
		if err != nil {
			return err
		}
		if claimResult.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
	return run, err
}
