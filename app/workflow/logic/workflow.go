package logic

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/distribution/reference"
	"github.com/google/uuid"
	systemlogic "github.com/w7panel/w7panel-zpk/app/system/logic"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	"github.com/w7panel/w7panel-zpk/common/function"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"gorm.io/gen"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"io"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"net/url"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
	"strings"
	"time"
)

type ImageContext struct {
	// Reference is the pinned input; the other fields describe the output repository.
	Reference  string `json:"reference"`
	Repository string `json:"repository"`
	Host       string `json:"host"`
	PlainHTTP  bool   `json:"plain_http"`
}

type Workflow struct{}

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

func (l Workflow) readTaskConfig(q *dao.Query, imageRepository string) (Config, []entity.RegistryRepositoryDeployRule, error) {
	var rules []entity.RegistryRepositoryDeployRule
	query := q.RegistryRepositoryDeployRule
	if err := query.Where(query.RepositoryName.Eq(repositoryPath(imageRepository))).Order(query.ID).Scan(&rules); err != nil {
		return Config{}, nil, err
	}
	config := Config{Tasks: []Task{}}
	for _, rule := range rules {
		kubeConfig, err := (Deploy{}).DecryptK8sConfig(rule.K8sConfig)
		if err != nil {
			return Config{}, nil, err
		}

		config.Tasks = append(config.Tasks, Task{ID: rule.WorkflowTaskKey, Name: rule.TaskName, Stage: int(rule.ExecutionOrder), Action: rule.TaskAction, DeployType: int(rule.DeployType), MatchType: int(rule.MatchType), TagName: rule.TagName, Source: rule.ImageSource, SourceTask: rule.SourceBuildTaskKey, Dockerfile: rule.DockerfileInstructions, TagSuffix: rule.OutputTagSuffix, K8sConfig: kubeConfig, K8sNamespace: rule.K8sNamespace, K8sControllerType: rule.K8sControllerType, K8sAppName: rule.K8sAppName, K8sContainerName: rule.K8sContainerName})
	}
	if len(rules) > 0 {
		config.Repository = repositoryConfig(rules[0])
		if rules[0].RegistryPassword != "" {
			password, err := l.decryptConfigSnapshot(rules[0].RegistryPassword)
			if err != nil {
				return Config{}, nil, err
			}
			config.Repository.Password = password
		}
	}
	return config, rules, nil
}
func (l Workflow) Get(imageRepository string) (Config, error) {
	config, _, err := l.readTaskConfig(dao.Q, imageRepository)
	return config, err
}

// Repository paths identify workflows; the local registry address is resolved at use time.
func repositoryPath(imageRepository string) string {
	host, name, found := strings.Cut(imageRepository, "/")
	if found && (strings.ContainsAny(host, ".:") || host == "localhost") {
		return name
	}
	return imageRepository
}

func currentRepositoryImage(name string) ImageContext {
	host := strings.TrimRight(facade.GetConfig().GetString("registry_cli.default.external_domain"), "/")
	registryURL, _ := url.Parse(facade.GetConfig().GetString("registry_cli.default.url"))
	return ImageContext{Repository: host + "/" + name, Host: host, PlainHTTP: registryURL != nil && registryURL.Scheme == "http"}
}

func repositoryConfig(rule entity.RegistryRepositoryDeployRule) RepositoryConfig {
	return RepositoryConfig{Username: rule.RegistryUsername, UserID: rule.UserID, Name: rule.RepositoryName, Image: currentRepositoryImage(rule.RepositoryName)}
}

func (l Workflow) Rule(imageRepository string) (*entity.RegistryRepositoryDeployRule, error) {
	query := dao.Q.RegistryRepositoryDeployRule
	return query.Where(query.RepositoryName.Eq(repositoryPath(imageRepository)), query.WorkflowTaskKey.Neq("")).Order(query.ID).First()
}

// SavedRepository retains history ownership even after all tasks have been removed.
func (l Workflow) SavedRepository(imageRepository string) (RepositoryConfig, error) {
	rule, err := l.Rule(imageRepository)
	if err == nil {
		return repositoryConfig(*rule), nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return RepositoryConfig{}, err
	}
	query := dao.Q.RegistryRepositoryDeployRuleMatchLog
	run, err := query.Where(query.TriggerRepository.Eq(repositoryPath(imageRepository)), query.WorkflowTaskKey.Eq("")).Order(query.ID.Desc()).First()
	if err != nil {
		return RepositoryConfig{}, err
	}
	config, err := l.decodeSnapshotConfig(run.EncryptedConfigSnapshot)
	config.Repository.Image = currentRepositoryImage(config.Repository.Name)
	return config.Repository, err
}

type executionSnapshot struct {
	Config
	Image            ImageContext `json:"image"`
	RegistryUsername string       `json:"registry_username"`
	RegistryPassword string       `json:"registry_password"`
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

func (l Workflow) Save(imageRepository string, config Config) error {
	if err := l.checkWritesAllowed(); err != nil {
		return err
	}
	config.Repository.Name = repositoryPath(imageRepository)
	config.Repository.Image = currentRepositoryImage(config.Repository.Name)
	if err := config.Validate(); err != nil {
		return err
	}
	err := dao.Q.Transaction(func(tx *dao.Query) error {
		ruleQuery := tx.RegistryRepositoryDeployRule
		rules, err := ruleQuery.Select(ruleQuery.ID, ruleQuery.RepositoryID, ruleQuery.WorkflowTaskKey, ruleQuery.UserID, ruleQuery.RepositoryName, ruleQuery.CreatedAt, ruleQuery.LatestTriggerAt).Where(ruleQuery.RepositoryName.Eq(repositoryPath(imageRepository))).Order(ruleQuery.ID).Find()
		if err != nil {
			return err
		}
		if len(rules) > 0 {
			config.Repository = repositoryConfig(*rules[0])
		}
		// Credentials are captured only when configuration is saved.
		config.Repository.Username = facade.GetConfig().GetString("registry_cli.default.username")
		config.Repository.Password = facade.GetConfig().GetString("registry_cli.default.password")
		encryptedPassword := ""
		for _, task := range config.Tasks {
			if task.Action != ActionBuild {
				continue
			}
			if config.Repository.Username == "" || config.Repository.Password == "" {
				return errors.New("未配置默认镜像仓库管理员账号密码")
			}
			encryptedPassword, err = l.encryptConfigSnapshot(config.Repository.Password)
			if err != nil {
				return err
			}
			break
		}
		existingRulesByTaskID := map[string]*entity.RegistryRepositoryDeployRule{}
		for _, rule := range rules {
			existingRulesByTaskID[rule.WorkflowTaskKey] = rule
		}
		savedRuleIDs := []int32{}
		for _, task := range config.Tasks {
			kubeConfig, err := (Deploy{}).EncryptK8sConfig(task.K8sConfig)
			if err != nil {
				return err
			}

			existingRule := existingRulesByTaskID[task.ID]
			if existingRule == nil {
				existingRule = &entity.RegistryRepositoryDeployRule{}
			}
			savedRule := entity.RegistryRepositoryDeployRule{
				ID:                     existingRule.ID,
				RepositoryID:           existingRule.RepositoryID,
				UserID:                 config.Repository.UserID,
				RegistryUsername:       config.Repository.Username,
				RegistryPassword:       encryptedPassword,
				RepositoryName:         config.Repository.Name,
				DeployType:             int32(task.DeployType),
				MatchType:              int32(task.MatchType),
				TagName:                task.TagName,
				K8sConfig:              kubeConfig,
				K8sNamespace:           task.K8sNamespace,
				K8sControllerType:      task.K8sControllerType,
				K8sAppName:             task.K8sAppName,
				K8sContainerName:       task.K8sContainerName,
				CreatedAt:              existingRule.CreatedAt,
				LatestTriggerAt:        existingRule.LatestTriggerAt,
				WorkflowTaskKey:        task.ID,
				TaskName:               task.Name,
				ExecutionOrder:         int32(task.Stage),
				TaskAction:             task.Action,
				ImageSource:            task.Source,
				SourceBuildTaskKey:     task.SourceTask,
				DockerfileInstructions: task.Dockerfile,
				OutputTagSuffix:        task.TagSuffix,
			}
			if err := tx.RegistryRepositoryDeployRule.Save(&savedRule); err != nil {
				return err
			}

			savedRuleIDs = append(savedRuleIDs, savedRule.ID)
		}
		query := ruleQuery.Where(ruleQuery.RepositoryName.Eq(repositoryPath(imageRepository)))
		if len(savedRuleIDs) > 0 {
			query = query.Where(ruleQuery.ID.NotIn(savedRuleIDs...))
		}
		if _, err = query.Delete(); err != nil {
			return err
		}
		return nil
	})
	return err
}
func (l Workflow) Runs(imageRepository string, taskID string, page, pageSize int) ([]entity.RegistryRepositoryDeployRuleMatchLog, int64, error) {
	runs := []entity.RegistryRepositoryDeployRuleMatchLog{}
	query := dao.Q.RegistryRepositoryDeployRuleMatchLog
	filtered := query.Where(query.TriggerRepository.Eq(repositoryPath(imageRepository)), query.WorkflowTaskKey.Eq(taskID), query.WorkflowRunID.IsNotNull())
	total, err := filtered.Count()
	if err != nil {
		return nil, 0, err
	}
	err = filtered.Order(query.CreatedAt.Desc(), query.ID.Desc()).Offset((page - 1) * pageSize).Limit(pageSize).Scan(&runs)
	return runs, total, err
}

// selectedTaskIDs derives the trigger's task selection from its immutable task records.
func (l Workflow) selectedTaskIDs(ctx context.Context, run entity.RegistryRepositoryDeployRuleMatchLog) (map[string]bool, error) {
	var taskIDs []string
	query := dao.Q.RegistryRepositoryDeployRuleMatchLog
	if err := query.WithContext(ctx).Where(query.WorkflowRunID.Eq(run.WorkflowRunID), query.WorkflowTaskKey.Neq("")).Pluck(query.WorkflowTaskKey, &taskIDs); err != nil {
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
func (l Workflow) identifyWorkflowBuildPush(imageRepository, tag, digest, pushActor string) (bool, error) {
	if err := l.checkWritesAllowed(); err != nil {
		return false, err
	}
	if pushActor == "" {
		return false, nil
	}
	query := dao.Q.RegistryRepositoryDeployRuleMatchLog
	// Completed output records also cover webhook delivery after the task has finished.
	outputs, err := query.Where(query.TriggerRepository.Eq(repositoryPath(imageRepository)), query.BuildOutputTag.Eq(tag), query.BuildOutputDigest.Eq(digest), query.WorkflowTaskKey.Neq("")).Order(query.ID.Desc()).Limit(1).Find()
	if err != nil {
		return false, err
	}
	if len(outputs) > 0 {
		run, err := query.Where(query.WorkflowRunID.Eq(outputs[0].WorkflowRunID), query.WorkflowTaskKey.Eq("")).First()
		if err != nil {
			return false, err
		}
		snapshot, err := l.decodeExecutionSnapshot(run.EncryptedConfigSnapshot)
		if err != nil {
			return false, err
		}
		if snapshot.RegistryUsername == pushActor {
			return true, nil
		}
	}
	runs, err := query.Where(query.TriggerRepository.Eq(repositoryPath(imageRepository)), query.WorkflowTaskKey.Eq(""), query.ExecutionStatus.Eq("running")).Find()
	if err != nil {
		return false, err
	}
	for _, run := range runs {
		snapshot, err := l.decodeExecutionSnapshot(run.EncryptedConfigSnapshot)
		if err != nil {
			return false, err
		}
		if pushActor != snapshot.RegistryUsername {
			continue
		}
		selected, err := l.selectedTaskIDs(context.Background(), *run)
		if err != nil {
			return false, err
		}
		for _, task := range snapshot.Tasks {
			if !selected[task.ID] || task.Action != ActionBuild || tag != run.TriggerTag+task.TagSuffix {
				continue
			}
			taskLog, err := query.Where(query.WorkflowRunID.Eq(run.WorkflowRunID), query.WorkflowTaskKey.Eq(task.ID)).First()
			if err != nil {
				return false, err
			}
			if taskLog.ExecutionStatus != "running" {
				continue
			}
			_, err = query.Where(query.ID.Eq(taskLog.ID)).UpdateSimple(query.UpdatedAt.Value(time.Now()), query.BuildOutputTag.Value(tag), query.BuildOutputDigest.Value(digest))
			return true, err
		}
	}
	return false, nil
}

// Trigger stores task configuration and input in an immutable execution snapshot.
func (l Workflow) Trigger(imageRepository string, repository, tag, digest, eventID string, eventType int, image ImageContext) error {
	if err := l.checkWritesAllowed(); err != nil {
		return err
	}
	return dao.Q.Transaction(func(tx *dao.Query) error {
		return l.enqueueImageWorkflow(tx, imageRepository, repository, tag, digest, eventID, eventType, image)
	})
}

func (l Workflow) enqueueImageWorkflow(tx *dao.Query, imageRepository string, repository, tag, digest, eventID string, eventType int, image ImageContext) error {
	config, rules, err := l.readTaskConfig(tx, imageRepository)
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
	event := ImagePushed{Repository: repository, Tag: tag, Digest: digest, EventID: eventID, Image: image}
	return l.enqueueTasks(tx, config, rules, tasks, taskMatchErr, event)
}

// TriggerManual uses the same saved Chart, queue and execution snapshots as push triggers.
func (l Workflow) TriggerManual(event ImagePushed, taskID string, admin bool) (string, error) {
	if err := l.checkWritesAllowed(); err != nil {
		return "", err
	}
	if !facade.GetConfig().GetBool("workflow.enabled") {
		return "", errors.New("镜像自动任务执行器未启用")
	}
	runID, err := event.RunID()
	if err != nil {
		return "", err
	}
	err = dao.Q.Transaction(func(tx *dao.Query) error {
		config, rules, err := l.readTaskConfig(tx, event.Image.Repository)
		if err != nil {
			return err
		}
		tasks, err := config.SelectManualTasks(event.Tag, taskID)
		if err != nil {
			return err
		}
		for _, task := range tasks {
			if task.Action == ActionDeploy && task.K8sConfig == "" && !admin {
				return errors.New("当前集群部署仅管理员可执行")
			}
		}
		return l.enqueueTasks(tx, config, rules, tasks, nil, event)
	})
	if err != nil {
		return "", err
	}
	return runID, nil
}

func (l Workflow) enqueueTasks(tx *dao.Query, config Config, rules []entity.RegistryRepositoryDeployRule, tasks []Task, taskMatchErr error, event ImagePushed) error {
	if config.Repository.Name != "" {
		event.Image.Repository = config.Repository.Image.Repository
		event.Image.Host = config.Repository.Image.Host
		event.Image.PlainHTTP = config.Repository.Image.PlainHTTP
	}
	if event.EventID == "" {
		return errors.New("镜像推送事件缺少唯一 ID")
	}
	if !validDigest(event.Digest) {
		return errors.New("镜像 digest 无效")
	}
	runID, err := event.RunID()
	if err != nil {
		return err
	}
	matchedTaskIDs := map[string]bool{}
	for _, task := range tasks {
		matchedTaskIDs[task.ID] = true
	}
	snapshot := executionSnapshot{Config: config, Image: event.Image}
	for _, task := range tasks {
		if task.Action != ActionBuild {
			continue
		}
		snapshot.RegistryUsername = config.Repository.Username
		snapshot.RegistryPassword = config.Repository.Password
		if snapshot.RegistryUsername == "" || snapshot.RegistryPassword == "" {
			return errors.New("未配置默认镜像仓库管理员账号密码")
		}
		break
	}
	snapshotJSON, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	encryptedSnapshot, err := l.encryptConfigSnapshot(string(snapshotJSON))
	if err != nil {
		return err
	}
	run := entity.RegistryRepositoryDeployRuleMatchLog{WorkflowRunID: runID, TriggerRepository: config.Repository.Name, TriggerTag: event.Tag, TriggerDigest: event.Digest, ImageName: event.Image.Reference, ExecutionStatus: "pending", EncryptedConfigSnapshot: encryptedSnapshot}
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

	for _, rule := range rules {
		if !matchedTaskIDs[rule.WorkflowTaskKey] {
			continue
		}
		taskLog := entity.RegistryRepositoryDeployRuleMatchLog{WorkflowRunID: runID, WorkflowTaskKey: rule.WorkflowTaskKey, RuleID: rule.ID, TriggerRepository: config.Repository.Name, ImageName: run.ImageName, ExecutionStatus: run.ExecutionStatus, K8sLog: "Helm 执行实例 wf-" + runID, TriggerTag: event.Tag, TriggerDigest: event.Digest}
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
func (l Workflow) Status(imageRepository string, runID string) (entity.RegistryRepositoryDeployRuleMatchLog, error) {
	query := dao.Q.RegistryRepositoryDeployRuleMatchLog
	run, err := query.Where(query.WorkflowRunID.Eq(runID), query.WorkflowTaskKey.Eq(""), query.TriggerRepository.Eq(repositoryPath(imageRepository))).First()
	if err != nil {
		return entity.RegistryRepositoryDeployRuleMatchLog{}, err
	}
	return *run, nil
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
	return l.readLogs(ctx, nil, run, tailLines, taskID)
}

func (l Workflow) readLogs(ctx context.Context, client kubernetes.Interface, run entity.RegistryRepositoryDeployRuleMatchLog, tailLines int64, taskID string) (LogResult, error) {
	query := dao.Q.RegistryRepositoryDeployRuleMatchLog
	taskLog, err := query.WithContext(ctx).Where(query.WorkflowRunID.Eq(run.WorkflowRunID), query.WorkflowTaskKey.Eq(taskID)).First()
	if err != nil {
		return LogResult{}, err
	}
	archived := taskLog.ExecutionStatus == "succeeded" || taskLog.ExecutionStatus == "failed" || (run.ExecutionStatus != "pending" && run.ExecutionStatus != "running")
	if archived && (taskLog.K8sLog != "" || taskLog.FailureReason != "") {
		return archivedTaskLogs(*taskLog, tailLines), nil
	}
	if client == nil {
		config, err := rest.InClusterConfig()
		if err != nil {
			return LogResult{}, err
		}
		client, err = kubernetes.NewForConfig(config)
		if err != nil {
			return LogResult{}, err
		}
	}
	pods, err := client.CoreV1().Pods(executionNamespace).List(ctx, metav1.ListOptions{LabelSelector: "w7.cc/workflow-run=wf-" + run.WorkflowRunID})
	if err != nil {
		return LogResult{}, err
	}
	logResult := LogResult{Status: taskLog.ExecutionStatus, Archived: archived}
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
		result := archivedTaskLogs(*taskLog, tailLines)
		result.Archived = archived
		return result, nil
	}
	logResult.Log = logText.String()
	return logResult, nil
}

func archivedTaskLogs(taskLog entity.RegistryRepositoryDeployRuleMatchLog, tailLines int64) LogResult {
	lines := strings.Split(strings.TrimRight(taskLog.K8sLog, "\n"), "\n")
	if int64(len(lines)) > tailLines {
		lines = lines[len(lines)-int(tailLines):]
	}
	text := strings.Join(lines, "\n")
	if text == "" {
		text = taskLog.FailureReason
	}
	if text == "" {
		text = "暂无任务日志"
	}
	return LogResult{Log: text, Status: taskLog.ExecutionStatus, Archived: true}
}

// OnImagePushed receives the source image and task credentials, then queues execution.
// The caller owns repository coordination and applies Rebuilt to its tag metadata.
type ImagePushed struct {
	Repository string
	Tag        string
	Digest     string
	Actor      string
	EventID    string
	EventType  int
	Image      ImageContext
}

type ImagePushResult struct{ Rebuilt bool }

func (event ImagePushed) RunID() (string, error) {
	if event.Image.Repository == "" || event.EventID == "" {
		return "", errors.New("镜像推送缺少仓库或事件标识")
	}
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%s", repositoryPath(event.Image.Repository), event.EventID)))
	return hex.EncodeToString(sum[:16]), nil
}

func (l Workflow) OnImagePushed(event ImagePushed) (ImagePushResult, error) {
	result := ImagePushResult{}
	if event.Image.Repository == "" {
		query := dao.Q.RegistryRepositoryDeployRule
		rule, err := query.Where(query.RepositoryName.Eq(event.Repository), query.WorkflowTaskKey.Neq("")).Order(query.ID).First()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return result, nil
		}
		if err != nil {
			return result, err
		}
		event.Image = currentRepositoryImage(rule.RepositoryName)
		event.Image.Reference = event.Image.Repository + "@" + event.Digest
	}
	if event.Image.Host == "" || event.Image.Repository != event.Image.Host+"/"+event.Repository || event.Image.Reference != event.Image.Repository+"@"+event.Digest {
		return result, errors.New("源镜像信息与推送事件不一致")
	}
	named, err := reference.WithName(event.Image.Repository)
	if err != nil || reference.Domain(named) != event.Image.Host {
		return result, errors.New("源镜像仓库地址无效")
	}
	isWorkflowBuildPush, err := l.identifyWorkflowBuildPush(event.Image.Repository, event.Tag, event.Digest, event.Actor)
	if err != nil {
		return result, err
	}
	result.Rebuilt = isWorkflowBuildPush
	if isWorkflowBuildPush {
		return result, nil
	}
	err = l.Trigger(event.Image.Repository, event.Repository, event.Tag, event.Digest, event.EventID, event.EventType, event.Image)
	return result, err
}

// ResolveManualImage resolves public inputs anonymously and pins the source digest.
func (Workflow) ResolveManualImage(ctx context.Context, source reference.NamedTagged) (ImagePushed, error) {
	repository, err := remote.NewRepository(source.Name())
	if err != nil {
		return ImagePushed{}, err
	}
	if repository.Reference.Registry == "docker.io" {
		repository.Reference.Registry = "registry-1.docker.io"
	}
	repository.Client = &auth.Client{Client: retry.DefaultClient, Cache: auth.NewCache()}
	resolveCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	descriptor, err := repository.Resolve(resolveCtx, source.Tag())
	if err != nil {
		return ImagePushed{}, fmt.Errorf("读取公共镜像失败：%w", err)
	}
	digest := descriptor.Digest.String()
	return ImagePushed{
		Tag: source.Tag(), Digest: digest,
		EventID: "manual-" + uuid.NewString(),
		Image:   ImageContext{Reference: source.Name() + "@" + digest},
	}, nil
}
