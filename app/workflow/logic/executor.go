package logic

import (
	"context"
	"errors"
	"fmt"
	"io"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log/slog"
	"sigs.k8s.io/yaml"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	"github.com/w7panel/w7panel-zpk/common/service/taskqueue"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"helm.sh/helm/v3/pkg/release"
	"k8s.io/client-go/kubernetes"
)

type Executor struct {
	Workflow
	once sync.Once
}

var defaultExecutor = &Executor{}

func InitExecutor() { defaultExecutor.Start() }

func (l *Executor) Start() {
	l.once.Do(func() {
		if !facade.GetConfig().GetBool("workflow.enabled") {
			return
		}
		go func() {
			if err := l.Run(context.Background()); err != nil {
				slog.Error("workflow worker", "error", err)
			}
		}()
	})
}

func (l *Executor) Run(ctx context.Context) error {
	concurrency := facade.GetConfig().GetInt("workflow.concurrency")
	queue, err := taskqueue.New(ctx, concurrency, facade.GetConfig().GetInt("workflow.queue_size"), l.processRun)
	if err != nil {
		return err
	}
	defer queue.Close()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return nil
		}
		if err := l.checkWritesAllowed(); err == nil {
			if err := l.expireInterrupted(); err != nil {
				slog.Error("expire workflow runs", "error", err)
			}
			if err := l.dispatchRuns(ctx, queue, concurrency); err != nil && ctx.Err() == nil {
				slog.Error("workflow dispatch", "error", err)
			}
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

// Only IDs enter the process queue; a worker claims current database state when it starts.
func (l *Executor) dispatchRuns(ctx context.Context, queue *taskqueue.Queue[int32], pageSize int) error {
	var afterID int32
	for {
		query := dao.Q.RegistryRepositoryDeployRuleMatchLog
		rows, err := query.WithContext(ctx).Select(query.ID).Where(query.ID.Gt(afterID), query.ExecutionStatus.Eq("pending"), query.WorkflowTaskKey.Eq("")).Order(query.ID).Limit(pageSize).Find()
		if err != nil {
			return err
		}
		for _, row := range rows {
			if !queue.Push(row.ID) {
				return ctx.Err()
			}
			afterID = row.ID
		}
		if len(rows) < pageSize {
			return nil
		}
	}
}

func (l *Executor) processRun(taskCtx context.Context, id int32) {
	if err := l.checkWritesAllowed(); err != nil {
		return
	}
	run, err := l.claimRun(taskCtx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return
	}
	if err != nil {
		slog.Error("claim workflow run", "id", id, "error", err)
		return
	}
	ctx, cancel := context.WithTimeout(taskCtx, 2*time.Hour)
	defer cancel()
	err = l.Execute(ctx, run)
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
}

// Resolve local endpoints when execution begins, leaving external manual inputs pinned.
func (snapshot executionSnapshot) runtimeImage() ImageContext {
	image := snapshot.Image
	target := currentRepositoryImage(snapshot.Repository.Name)
	if strings.HasPrefix(image.Reference, image.Repository+"@") {
		image.Reference = target.Repository + strings.TrimPrefix(image.Reference, image.Repository)
	}
	image.Repository, image.Host, image.PlainHTTP = target.Repository, target.Host, target.PlainHTTP
	return image
}

func (l *Executor) Execute(ctx context.Context, run entity.RegistryRepositoryDeployRuleMatchLog) error {
	executionSnapshot, err := l.decodeExecutionSnapshot(run.EncryptedConfigSnapshot)
	if err != nil {
		return err
	}
	config, sourceImage := executionSnapshot.Config, executionSnapshot.runtimeImage()
	config.Repository.Image = currentRepositoryImage(config.Repository.Name)
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
			if executionSnapshot.RegistryUsername == "" || executionSnapshot.RegistryPassword == "" {
				return errors.New("执行记录缺少镜像仓库账号密码，请重新触发")
			}
		}
	}
	executionValues := map[string]interface{}{
		"selected": selectedTaskIDs, "remote": remoteDeployTaskIDs, "kubeconfigs": targetKubeconfigsByTaskID, "registryUsername": executionSnapshot.RegistryUsername, "registryPassword": executionSnapshot.RegistryPassword,
		"run": map[string]interface{}{"original": sourceImage.Reference, "repository": sourceImage.Repository, "host": sourceImage.Host, "tag": run.TriggerTag, "plainHTTP": sourceImage.PlainHTTP},
	}
	for _, pair := range []struct{ key, config string }{{"buildImage", "workflow.build_image"}, {"deployImage", "workflow.deploy_image"}} {
		if executorImage := facade.GetConfig().GetString(pair.config); executorImage != "" {
			executionValues[pair.key] = executorImage
		}
	}
	return (Helm{}).Install(ctx, run, config, executionValues, func(client kubernetes.Interface, helmRelease *release.Release, final bool) error {
		return l.persistTaskResults(client, run, helmRelease, final)
	})
}

func (l *Executor) claimRun(ctx context.Context, id int32) (entity.RegistryRepositoryDeployRuleMatchLog, error) {
	var run entity.RegistryRepositoryDeployRuleMatchLog
	// Claim at most one active run per repository using workflow records only.
	err := dao.Q.Transaction(func(tx *dao.Query) error {
		query := tx.RegistryRepositoryDeployRuleMatchLog
		candidate, err := query.WithContext(ctx).Where(query.ID.Eq(id), query.ExecutionStatus.Eq("pending"), query.WorkflowTaskKey.Eq("")).First()
		if err != nil {
			return err
		}
		run = *candidate

		// All dispatchers for this repository lock the same workflow coordinator row.
		// Locking reads below see committed state even under MySQL repeatable read.
		if _, err := query.Clauses(clause.Locking{Strength: "UPDATE"}).Where(query.TriggerRepository.Eq(run.TriggerRepository), query.WorkflowTaskKey.Eq("")).Order(query.ID).First(); err != nil {
			return err
		}
		activeRuns, err := query.Clauses(clause.Locking{Strength: "UPDATE"}).Where(query.TriggerRepository.Eq(run.TriggerRepository), query.WorkflowTaskKey.Eq(""), query.ExecutionStatus.Eq("running")).Limit(1).Find()
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

func (l *Executor) expireInterrupted() error {
	query := dao.Q.RegistryRepositoryDeployRuleMatchLog
	_, err := query.Where(query.ExecutionStatus.Eq("running"), query.UpdatedAt.Lt(time.Now().Add(-3*time.Hour))).UpdateSimple(query.UpdatedAt.Value(time.Now()), query.ExecutionStatus.Value("failed"), query.FailureReason.Value("执行中断或超过最大执行时间，请检查集群资源后重新触发"))
	return err
}

// persistTaskResults archives completed containers after each stage. Only the
// final collection classifies missing tasks as unknown or never started.
func (l *Executor) persistTaskResults(client kubernetes.Interface, run entity.RegistryRepositoryDeployRuleMatchLog, helmRelease *release.Release, final bool) error {
	if err := l.checkWritesAllowed(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pods, podListErr := client.CoreV1().Pods(executionNamespace).List(ctx, metav1.ListOptions{LabelSelector: "w7.cc/workflow-run=wf-" + run.WorkflowRunID})
	config, err := l.decodeSnapshotConfig(run.EncryptedConfigSnapshot)
	if err != nil {
		slog.Error("read workflow result configuration", "error", err)
		return err
	}
	tasksByID := map[string]Task{}
	for _, task := range config.Tasks {
		tasksByID[task.ID] = task
	}
	type taskExecutionResult struct{ status, logText, outputDigest, failureReason string }
	resultsByTaskID := map[string]taskExecutionResult{}
	// Helm records completed hooks before moving on to the next execution group.
	// Keep those results even when completed Pods have disappeared or cannot be listed.
	if helmRelease != nil {
		for _, hook := range helmRelease.Hooks {
			if hook.Kind != "Job" || hook.LastRun.Phase == release.HookPhaseUnknown || hook.LastRun.Phase == "" {
				continue
			}
			var job batchv1.Job
			if err := yaml.Unmarshal([]byte(hook.Manifest), &job); err != nil {
				return err
			}
			for _, container := range job.Spec.Template.Spec.Containers {
				taskResult := taskExecutionResult{status: "unknown", failureReason: "执行组已启动，但未获取到该任务的容器结果"}
				if hook.LastRun.Phase == release.HookPhaseSucceeded {
					taskResult.status = "succeeded"
					taskResult.failureReason = ""
				}
				if previousResult, ok := resultsByTaskID[container.Name]; !ok || previousResult.status != "succeeded" {
					resultsByTaskID[container.Name] = taskResult
				}
			}
		}
	}
	tailLines, logByteLimit := int64(100), int64(16384)
	if pods == nil {
		pods = &corev1.PodList{}
	}
	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			if !final {
				completed := slices.ContainsFunc(pod.Status.ContainerStatuses, func(state corev1.ContainerStatus) bool {
					return state.Name == container.Name && state.State.Terminated != nil
				})
				if !completed {
					continue
				}
			}
			taskResult := taskExecutionResult{status: "failed", logText: fmt.Sprintf("%s / %s\n", pod.Name, container.Name)}
			for _, state := range pod.Status.ContainerStatuses {
				if state.Name == container.Name && state.State.Terminated != nil {
					if state.State.Terminated.ExitCode == 0 {
						taskResult.status = "succeeded"
					}
					taskResult.logText += fmt.Sprintf("退出码: %d\n", state.State.Terminated.ExitCode)
					if taskResult.status == "failed" {
						taskResult.failureReason = fmt.Sprintf("%s：%s（退出码 %d）", state.State.Terminated.Reason, state.State.Terminated.Message, state.State.Terminated.ExitCode)
					}
					if tasksByID[container.Name].Action == ActionBuild {
						_, digest, hasDigestSeparator := strings.Cut(strings.TrimSpace(state.State.Terminated.Message), "@")
						if hasDigestSeparator && validDigest(digest) {
							taskResult.outputDigest = digest
						}
					}
				}
				if state.Name == container.Name && state.State.Waiting != nil {
					taskResult.failureReason = state.State.Waiting.Reason + ": " + state.State.Waiting.Message
					taskResult.logText += taskResult.failureReason + "\n"
				}
			}
			stream, err := client.CoreV1().Pods(executionNamespace).GetLogs(pod.Name, &corev1.PodLogOptions{Container: container.Name, TailLines: &tailLines, LimitBytes: &logByteLimit}).Stream(ctx)
			if err == nil {
				content, readErr := io.ReadAll(io.LimitReader(stream, logByteLimit))
				stream.Close()
				if readErr == nil {
					taskResult.logText += string(content)
				}
			}
			if previousResult, ok := resultsByTaskID[container.Name]; !ok || previousResult.status != "succeeded" || taskResult.status == "succeeded" {
				resultsByTaskID[container.Name] = taskResult
			}
		}
	}
	var taskLogs []entity.RegistryRepositoryDeployRuleMatchLog
	query := dao.Q.RegistryRepositoryDeployRuleMatchLog
	if err := query.Where(query.WorkflowRunID.Eq(run.WorkflowRunID), query.WorkflowTaskKey.Neq("")).Scan(&taskLogs); err != nil {
		slog.Error("read workflow task logs", "error", err)
		return err
	}
	for _, taskLog := range taskLogs {
		taskResult, ok := resultsByTaskID[taskLog.WorkflowTaskKey]
		if (!ok || taskResult.logText == "") && (taskLog.ExecutionStatus == "succeeded" || taskLog.ExecutionStatus == "failed") {
			continue
		}
		if !ok {
			if !final {
				continue
			}
			if podListErr != nil {
				taskResult = taskExecutionResult{status: "unknown", failureReason: "未能读取容器状态：" + podListErr.Error()}
			} else {
				taskResult = taskExecutionResult{status: "skipped", failureReason: "前置执行组失败或流程未启动，本任务未执行"}
			}
		}
		outputTag, outputDigest := taskLog.BuildOutputTag, taskLog.BuildOutputDigest
		if taskResult.outputDigest != "" {
			outputTag, outputDigest = run.TriggerTag+tasksByID[taskLog.WorkflowTaskKey].TagSuffix, taskResult.outputDigest
		}
		if _, err := query.Where(query.ID.Eq(taskLog.ID)).UpdateSimple(query.UpdatedAt.Value(time.Now()), query.ExecutionStatus.Value(taskResult.status), query.FailureReason.Value(taskResult.failureReason), query.K8sLog.Value(strings.TrimSpace(taskResult.logText)), query.BuildOutputTag.Value(outputTag), query.BuildOutputDigest.Value(outputDigest)); err != nil {
			return err
		}
	}
	return podListErr
}
