package logic

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/distribution/reference"
	systemlogic "github.com/w7panel/w7panel-zpk/app/system/logic"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	"github.com/w7panel/w7panel-zpk/common/function"
	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"gorm.io/gen"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"helm.sh/helm/v3/pkg/release"
	"io"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"log/slog"
	"sigs.k8s.io/yaml"
	"strings"
	"time"
)

type ImageContext struct {
	Reference  string `json:"reference"`
	Repository string `json:"repository"`
	Host       string `json:"host"`
	PlainHTTP  bool   `json:"plain_http"`
}

type Workflow struct {
	commonlogic.Logic
}

func (l Workflow) checkWritesAllowed() error {
	if systemlogic.ShouldBlockWrites(facade.GetConfig()) {
		return errors.New("数据库切换期间禁止执行工作流")
	}
	return nil
}
func (l Workflow) encryptConfigSnapshot(snapshotJSON string) (string, error) {
	key := function.GetMd5(facade.GetConfig().GetString("setting.secret"))
	return function.AesEncrypt(snapshotJSON, key)
}

func (l Workflow) decryptConfigSnapshot(encryptedSnapshot string) (string, error) {
	key := function.GetMd5(facade.GetConfig().GetString("setting.secret"))
	return function.AesDecrypt(encryptedSnapshot, key)
}

func (l Workflow) readTaskConfig(q *dao.Query, repositoryID int32) (Config, []entity.RegistryRepositoryDeployRule, error) {
	var rules []entity.RegistryRepositoryDeployRule
	query := q.RegistryRepositoryDeployRule
	if err := query.Where(query.RepositoryID.Eq(repositoryID)).Order(query.ID).Scan(&rules); err != nil {
		return Config{}, nil, err
	}
	config := Config{Namespace: executionNamespace, Tasks: []Task{}}
	for _, rule := range rules {
		key := rule.WorkflowTaskKey
		if key == "" {
			key = fmt.Sprintf("rule-%d", rule.ID)
		}
		kubeConfig, err := (Deploy{}).DecryptK8sConfig(rule.K8sConfig)
		if err != nil {
			return Config{}, nil, err
		}

		stage := int(rule.ExecutionOrder)
		if stage < 1 {
			stage = 1
		}
		action := rule.TaskAction
		if action == "" {
			action = ActionDeploy
		}
		source := rule.ImageSource
		if source == "" {
			source = SourceOriginal
		}
		name := rule.TaskName
		if name == "" {
			name = rule.K8sAppName
		}
		config.Tasks = append(config.Tasks, Task{ID: key, Name: name, Stage: stage, Action: action, DeployType: int(rule.DeployType), MatchType: int(rule.MatchType), TagName: rule.TagName, Source: source, SourceTask: rule.SourceBuildTaskKey, Dockerfile: rule.DockerfileInstructions, TagSuffix: rule.OutputTagSuffix, K8sConfig: kubeConfig, K8sNamespace: rule.K8sNamespace, K8sControllerType: rule.K8sControllerType, K8sAppName: rule.K8sAppName, K8sContainerName: rule.K8sContainerName})
	}
	return config, rules, nil
}
func (l Workflow) Get(repositoryID int32) (Config, error) {
	config, _, err := l.readTaskConfig(dao.Q, repositoryID)
	return config, err
}

type executionSnapshot struct {
	Config
	Image         ImageContext `json:"image"`
	RegistryToken string       `json:"registry_token"`
}

func (l Workflow) decodeExecutionSnapshot(content string) (executionSnapshot, error) {
	plain, err := l.decryptConfigSnapshot(content)
	if err != nil {
		return executionSnapshot{}, errors.New("解密工作流快照失败，请检查 setting.secret")
	}
	var snapshot executionSnapshot
	err = json.Unmarshal([]byte(plain), &snapshot)
	return snapshot, err
}

func (l Workflow) decodeSnapshotConfig(content string) (Config, error) {
	plain, err := l.decryptConfigSnapshot(content)
	if err != nil {
		return Config{}, errors.New("解密工作流快照失败，请检查 setting.secret")
	}
	var config Config
	err = json.Unmarshal([]byte(plain), &config)
	return config, err
}

func (l Workflow) Save(repositoryID int32, config Config) error {
	if err := l.checkWritesAllowed(); err != nil {
		return err
	}
	config.Namespace = executionNamespace
	if err := config.Validate(); err != nil {
		return err
	}
	return dao.Q.Transaction(func(tx *dao.Query) error {
		_, rules, err := l.readTaskConfig(tx, repositoryID)
		if err != nil {
			return err
		}
		existingRulesByTaskID := map[string]entity.RegistryRepositoryDeployRule{}
		for _, rule := range rules {
			taskID := rule.WorkflowTaskKey
			if taskID == "" {
				taskID = fmt.Sprintf("rule-%d", rule.ID)
			}
			existingRulesByTaskID[taskID] = rule
		}
		savedRuleIDs := []int32{}
		for _, task := range config.Tasks {
			kubeConfig, err := (Deploy{}).EncryptK8sConfig(task.K8sConfig)
			if err != nil {
				return err
			}

			existingRule := existingRulesByTaskID[task.ID]
			savedRule := entity.RegistryRepositoryDeployRule{ID: existingRule.ID, RepositoryID: repositoryID, DeployType: int32(task.DeployType), MatchType: int32(task.MatchType), TagName: task.TagName, K8sConfig: kubeConfig, K8sNamespace: task.K8sNamespace, K8sControllerType: task.K8sControllerType, K8sAppName: task.K8sAppName, K8sContainerName: task.K8sContainerName, CreatedAt: existingRule.CreatedAt, LatestTriggerAt: existingRule.LatestTriggerAt, WorkflowTaskKey: task.ID, TaskName: task.Name, ExecutionOrder: int32(task.Stage), TaskAction: task.Action, ImageSource: task.Source, SourceBuildTaskKey: task.SourceTask, DockerfileInstructions: task.Dockerfile, OutputTagSuffix: task.TagSuffix}
			if err := tx.RegistryRepositoryDeployRule.Save(&savedRule); err != nil {
				return err
			}
			savedRuleIDs = append(savedRuleIDs, savedRule.ID)
		}
		ruleQuery := tx.RegistryRepositoryDeployRule
		query := ruleQuery.Where(ruleQuery.RepositoryID.Eq(repositoryID))
		if len(savedRuleIDs) > 0 {
			query = query.Where(ruleQuery.ID.NotIn(savedRuleIDs...))
		}
		if _, err = query.Delete(); err != nil {
			return err
		}
		_, err = (Helm{}).SaveChart(repositoryID, config)
		return err
	})
}
func (l Workflow) Runs(repositoryID int32, taskID string) ([]entity.RegistryRepositoryDeployRuleMatchLog, error) {
	runs := []entity.RegistryRepositoryDeployRuleMatchLog{}
	query := dao.Q.RegistryRepositoryDeployRuleMatchLog
	err := query.Where(query.RepositoryID.Eq(repositoryID), query.WorkflowTaskKey.Eq(taskID), query.WorkflowRunID.IsNotNull()).Order(query.CreatedAt.Desc(), query.ID.Desc()).Limit(50).Scan(&runs)
	return runs, err
}

// selectedTaskIDs derives the trigger's task selection from its immutable task records.
func (l Workflow) selectedTaskIDs(ctx context.Context, run entity.RegistryRepositoryDeployRuleMatchLog) (map[string]bool, error) {
	var taskIDs []string
	query := dao.Q.RegistryRepositoryDeployRuleMatchLog
	if err := query.WithContext(ctx).Where(query.RepositoryID.Eq(run.RepositoryID), query.WorkflowRunID.Eq(run.WorkflowRunID), query.WorkflowTaskKey.Neq("")).Pluck(query.WorkflowTaskKey, &taskIDs); err != nil {
		return nil, err
	}
	selectedTaskIDs := make(map[string]bool, len(taskIDs))
	for _, id := range taskIDs {
		selectedTaskIDs[id] = true
	}
	return selectedTaskIDs, nil
}

// identifyWorkflowBuildPush verifies the signed run subject and output tag against selected build tasks.
// Current rebuild state belongs to the tag; logs retain immutable output information.
func (l Workflow) identifyWorkflowBuildPush(repositoryID int32, repository, tag, digest, pushActor string) (bool, error) {
	if err := l.checkWritesAllowed(); err != nil {
		return false, err
	}
	runID, taskID, _ := strings.Cut(strings.TrimPrefix(pushActor, WorkflowBuildActorPrefix), "--")
	isWorkflowBuildPush := false
	if strings.HasPrefix(pushActor, WorkflowBuildActorPrefix) && workflowRunIDPattern.MatchString(runID) {
		query := dao.Q.RegistryRepositoryDeployRuleMatchLog
		run, err := query.Where(query.WorkflowRunID.Eq(runID), query.WorkflowTaskKey.Eq(""), query.RepositoryID.Eq(repositoryID), query.TriggerRepository.Eq(repository)).First()
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return false, err
		}
		if err == nil {
			config, err := l.decodeSnapshotConfig(run.EncryptedConfigSnapshot)
			if err != nil {
				return false, err
			}
			selectedTaskIDs, err := l.selectedTaskIDs(context.Background(), *run)
			if err != nil {
				return false, err
			}
			matchedBuildTaskIDs := []string{}
			for _, task := range config.Tasks {
				if (taskID == "" || task.ID == taskID) && selectedTaskIDs[task.ID] && task.Action == ActionBuild && tag == run.TriggerTag+task.TagSuffix {
					matchedBuildTaskIDs = append(matchedBuildTaskIDs, task.ID)
				}
			}
			isWorkflowBuildPush = len(matchedBuildTaskIDs) > 0
			if len(matchedBuildTaskIDs) == 1 {
				taskID = matchedBuildTaskIDs[0]
			} else {
				taskID = ""
			}
		}
	}
	return isWorkflowBuildPush, l.recordBuildOutputFromPush(repositoryID, tag, digest, runID, taskID, isWorkflowBuildPush)
}

func (l Workflow) recordBuildOutputFromPush(repositoryID int32, tag, digest, runID, taskID string, isWorkflowBuildPush bool) error {
	if !isWorkflowBuildPush || taskID == "" {
		return nil
	}
	query := dao.Q.RegistryRepositoryDeployRuleMatchLog
	_, err := query.Where(query.WorkflowRunID.Eq(runID), query.WorkflowTaskKey.Eq(taskID), query.RepositoryID.Eq(repositoryID)).UpdateSimple(query.UpdatedAt.Value(time.Now()), query.BuildOutputTag.Value(tag), query.BuildOutputDigest.Value(digest))
	return err
}

// Trigger snapshots the saved Chart. Unconverted legacy rules continue on their old path.
func (l Workflow) Trigger(repositoryID int32, repository, tag, digest, eventID string, eventType int, image ImageContext, registryToken string) error {
	if err := l.checkWritesAllowed(); err != nil {
		return err
	}
	return dao.Q.Transaction(func(tx *dao.Query) error {
		return l.enqueueImageWorkflow(tx, repositoryID, repository, tag, digest, eventID, eventType, image, registryToken)
	})
}

func (l Workflow) enqueueImageWorkflow(tx *dao.Query, repositoryID int32, repository, tag, digest, eventID string, eventType int, image ImageContext, registryToken string) error {
	config, rules, err := l.readTaskConfig(tx, repositoryID)
	if err != nil {
		return err
	}
	hasWorkflowTasks := false
	for _, rule := range rules {
		if rule.WorkflowTaskKey != "" {
			hasWorkflowTasks = true
			break
		}
	}
	if !hasWorkflowTasks {
		return nil
	}

	tasks, taskMatchErr := config.SelectMatchingTasks(tag, eventType)
	if len(tasks) == 0 && taskMatchErr == nil {
		return nil
	}
	if eventID == "" {
		return errors.New("镜像推送事件缺少唯一 ID")
	}
	if !validDigest(digest) {
		return errors.New("镜像 digest 无效")
	}
	runID, err := (ImagePushed{RepositoryID: repositoryID, EventID: eventID}).RunID()
	if err != nil {
		return err
	}
	matchedTaskIDs := map[string]bool{}
	for _, task := range tasks {
		if task.Action == ActionBuild && registryToken == "" {
			return fmt.Errorf("任务 %s 缺少调用方提供的镜像仓库令牌", task.Name)
		}
		matchedTaskIDs[task.ID] = true
	}
	snapshotJSON, err := json.Marshal(executionSnapshot{Config: config, Image: image, RegistryToken: registryToken})
	if err != nil {
		return err
	}
	encryptedSnapshot, err := l.encryptConfigSnapshot(string(snapshotJSON))
	if err != nil {
		return err
	}
	run := entity.RegistryRepositoryDeployRuleMatchLog{WorkflowRunID: runID, RepositoryID: repositoryID, TriggerRepository: repository, TriggerTag: tag, TriggerDigest: digest, ImageName: repository + "@" + digest, ExecutionStatus: "pending", EncryptedConfigSnapshot: encryptedSnapshot}
	if taskMatchErr != nil {
		run.ExecutionStatus = "failed"
		run.FailureReason = taskMatchErr.Error()
	}
	insertResult := tx.RegistryRepositoryDeployRuleMatchLog.Clauses(clause.OnConflict{DoNothing: true}).WithResult(func(q gen.Dao) { err = q.Create(&run) })
	if err != nil {
		return err
	}
	if insertResult.RowsAffected == 0 {
		return nil
	}
	if taskMatchErr == nil {
		if err := (Helm{}).PinChart(repositoryID, runID, config); err != nil {
			return err
		}
	}
	for _, rule := range rules {
		if !matchedTaskIDs[rule.WorkflowTaskKey] {
			continue
		}
		taskLog := entity.RegistryRepositoryDeployRuleMatchLog{WorkflowRunID: runID, WorkflowTaskKey: rule.WorkflowTaskKey, RuleID: rule.ID, RepositoryID: repositoryID, ImageName: run.ImageName, ExecutionStatus: run.ExecutionStatus, K8sLog: "Helm 执行实例 wf-" + runID, TriggerTag: tag, TriggerDigest: digest}
		if err := tx.RegistryRepositoryDeployRuleMatchLog.Create(&taskLog); err != nil {
			return err
		}
		query := tx.RegistryRepositoryDeployRule
		if _, err := query.Where(query.ID.Eq(rule.ID)).Update(query.LatestTriggerAt, time.Now()); err != nil {
			return err
		}
	}
	return nil
}
func validDigest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != 71 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}
func (l Workflow) Status(repositoryID int32, runID string) (entity.RegistryRepositoryDeployRuleMatchLog, error) {
	query := dao.Q.RegistryRepositoryDeployRuleMatchLog
	run, err := query.Where(query.WorkflowRunID.Eq(runID), query.WorkflowTaskKey.Eq(""), query.RepositoryID.Eq(repositoryID)).First()
	if err != nil {
		return entity.RegistryRepositoryDeployRuleMatchLog{}, err
	}
	return *run, nil
}
func (l Workflow) ExpireInterrupted() error {
	query := dao.Q.RegistryRepositoryDeployRuleMatchLog
	_, err := query.Where(query.ExecutionStatus.Eq("running"), query.UpdatedAt.Lt(time.Now().Add(-3*time.Hour))).UpdateSimple(query.UpdatedAt.Value(time.Now()), query.ExecutionStatus.Value("failed"), query.FailureReason.Value("执行中断或超过最大执行时间，请检查集群资源后重新触发"))
	return err
}

// cleanupRunCharts retries terminal cleanup and removes abandoned files only after a grace period.
// Pending/running records are never removed, including while another replica executes them.
func (l Workflow) cleanupRunCharts() error {
	files, err := (Helm{}).runCharts()
	if err != nil {
		return err
	}
	query := dao.Q.RegistryRepositoryDeployRuleMatchLog
	for _, file := range files {
		run, err := query.Select(query.ExecutionStatus).Where(query.RepositoryID.Eq(file.RepositoryID), query.WorkflowRunID.Eq(file.RunID), query.WorkflowTaskKey.Eq("")).First()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if time.Since(file.ModifiedAt) < 3*time.Hour {
				continue
			}
		} else if err != nil {
			return err
		} else if run.ExecutionStatus == "pending" || run.ExecutionStatus == "running" {
			continue
		}
		if err := (Helm{}).RemoveRunChart(file.RepositoryID, file.RunID); err != nil {
			return err
		}
	}
	return nil
}

// CheckLegacyRuleWritable prevents old mutation endpoints from invalidating a cached workflow.
func (l Workflow) CheckLegacyRuleWritable(repositoryID int32) error {
	query := dao.Q.RegistryRepositoryDeployRule
	count, err := query.Where(query.RepositoryID.Eq(repositoryID), query.WorkflowTaskKey.Neq("")).Count()
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("该仓库已启用镜像自动任务，请通过工作流配置编辑")
	}
	return nil
}

type LogResult struct {
	Log       string
	Status    string
	Archived  bool
	Truncated bool
}

func (l Workflow) Logs(ctx context.Context, run entity.RegistryRepositoryDeployRuleMatchLog, tailLines int64, taskID string) (LogResult, error) {
	if tailLines < 1 || tailLines > 2000 {
		return LogResult{}, errors.New("日志行数必须在 1 到 2000 之间")
	}
	config, err := rest.InClusterConfig()
	if err != nil {
		return LogResult{}, err
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return LogResult{}, err
	}
	return l.readLogs(ctx, client, run, tailLines, taskID)
}

func (l Workflow) readLogs(ctx context.Context, client kubernetes.Interface, run entity.RegistryRepositoryDeployRuleMatchLog, tailLines int64, taskID string) (LogResult, error) {
	query := dao.Q.RegistryRepositoryDeployRuleMatchLog
	taskLog, err := query.WithContext(ctx).Where(query.RepositoryID.Eq(run.RepositoryID), query.WorkflowRunID.Eq(run.WorkflowRunID), query.WorkflowTaskKey.Eq(taskID)).First()
	if err != nil {
		return LogResult{}, err
	}
	pods, err := client.CoreV1().Pods(executionNamespace).List(ctx, metav1.ListOptions{LabelSelector: "w7.cc/workflow-run=wf-" + run.WorkflowRunID})
	if err != nil {
		return LogResult{}, err
	}
	logResult := LogResult{Status: taskLog.ExecutionStatus, Archived: run.ExecutionStatus != "pending" && run.ExecutionStatus != "running"}
	var logText strings.Builder
	logByteLimit := int64(1024 * 1024)
	hasLivePod := false
	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			if container.Name != taskID {
				continue
			}
			hasLivePod = true
			logResult.Archived = false
			fmt.Fprintf(&logText, "Pod: %s / %s\n", pod.Name, container.Name)
			for _, state := range pod.Status.ContainerStatuses {
				if state.Name != container.Name {
					continue
				}
				switch {
				case state.State.Terminated != nil:
					terminated := state.State.Terminated
					logResult.Status = "failed"
					if terminated.ExitCode == 0 {
						logResult.Status = "succeeded"
					}
					fmt.Fprintf(&logText, "退出码: %d · %s\n%s\n", terminated.ExitCode, terminated.Reason, terminated.Message)
				case state.State.Waiting != nil:
					fmt.Fprintf(&logText, "%s: %s\n", state.State.Waiting.Reason, state.State.Waiting.Message)
				}
			}
			stream, err := client.CoreV1().Pods(executionNamespace).GetLogs(pod.Name, &corev1.PodLogOptions{Container: container.Name, TailLines: &tailLines, LimitBytes: &logByteLimit}).Stream(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return LogResult{}, ctx.Err()
				}
				fmt.Fprintf(&logText, "日志暂不可用：%s\n", err)
				continue
			}
			bytesRead, copyErr := io.Copy(&logText, io.LimitReader(stream, logByteLimit))
			stream.Close()
			if copyErr != nil {
				return LogResult{}, copyErr
			}
			if bytesRead >= logByteLimit {
				logResult.Truncated = true
				logText.WriteString("\n[该容器日志达到 1 MiB 显示上限]\n")
			}
			logText.WriteString("\n")
		}
	}
	if !hasLivePod {
		lines := strings.Split(strings.TrimRight(taskLog.K8sLog, "\n"), "\n")
		if int64(len(lines)) > tailLines {
			lines = lines[len(lines)-int(tailLines):]
		}
		logText.WriteString(strings.Join(lines, "\n"))
		if logText.Len() == 0 {
			logText.WriteString(taskLog.FailureReason)
		}
		if logText.Len() == 0 {
			logText.WriteString("暂无任务日志")
		}
	}
	logResult.Log = logText.String()
	return logResult, nil
}

// persistTaskResults runs before ordinary release resources are removed. It keeps
// bounded logs and distinguishes completed, failed and never-started tasks.
func (l *Executor) persistTaskResults(client kubernetes.Interface, run entity.RegistryRepositoryDeployRuleMatchLog, helmRelease *release.Release) error {
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
		if !ok && (taskLog.ExecutionStatus == "succeeded" || taskLog.ExecutionStatus == "failed") {
			continue
		}
		if !ok {
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

// OnImagePushed receives the source image and task credentials, then queues execution.
// The caller owns repository coordination and applies Rebuilt to its tag metadata.
type ImagePushed struct {
	RepositoryID  int32
	Repository    string
	Tag           string
	Digest        string
	Actor         string
	EventID       string
	EventType     int
	Image         ImageContext
	RegistryToken string
}

type ImagePushResult struct{ Rebuilt bool }

func (event ImagePushed) RunID() (string, error) {
	if event.RepositoryID < 1 || event.EventID == "" {
		return "", errors.New("镜像推送缺少仓库或事件标识")
	}
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d:%s", event.RepositoryID, event.EventID)))
	return hex.EncodeToString(sum[:16]), nil
}

func (l Workflow) OnImagePushed(event ImagePushed) (ImagePushResult, error) {
	result := ImagePushResult{}
	if event.Image.Host == "" || event.Image.Repository != event.Image.Host+"/"+event.Repository || event.Image.Reference != event.Image.Repository+"@"+event.Digest {
		return result, errors.New("源镜像信息与推送事件不一致")
	}
	named, err := reference.WithName(event.Image.Repository)
	if err != nil || reference.Domain(named) != event.Image.Host {
		return result, errors.New("源镜像仓库地址无效")
	}
	isWorkflowBuildPush, err := l.identifyWorkflowBuildPush(event.RepositoryID, event.Repository, event.Tag, event.Digest, event.Actor)
	if err != nil {
		return result, err
	}
	result.Rebuilt = isWorkflowBuildPush
	if isWorkflowBuildPush {
		return result, nil
	}
	err = l.Trigger(event.RepositoryID, event.Repository, event.Tag, event.Digest, event.EventID, event.EventType, event.Image, event.RegistryToken)
	return result, err
}
