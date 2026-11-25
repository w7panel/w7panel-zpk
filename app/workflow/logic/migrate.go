package logic

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	"github.com/w7panel/w7panel-zpk/common/function"
	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
)

// MigrateWorkflowRules runs once for each repository-linked rule group at startup.
// Registry metadata is read only here to migrate historical rules.
func (l Workflow) MigrateWorkflowRules() error {
	query := dao.Q.RegistryRepositoryDeployRule
	rules, err := query.Where(query.RepositoryID.Gt(0)).Order(query.ID).Find()
	if err != nil || len(rules) == 0 {
		return err
	}
	host := strings.TrimRight(facade.GetConfig().GetString("registry_cli.default.external_domain"), "/")
	registryURL, err := url.Parse(facade.GetConfig().GetString("registry_cli.default.url"))
	if err != nil {
		return err
	}
	if host == "" {
		return fmt.Errorf("迁移工作流需要配置镜像仓库地址")
	}
	grouped := map[int32][]*entity.RegistryRepositoryDeployRule{}
	for _, rule := range rules {
		grouped[rule.RepositoryID] = append(grouped[rule.RepositoryID], rule)
	}
	for repositoryID, group := range grouped {
		repository, err := dao.Q.RegistryRepository.Where(dao.Q.RegistryRepository.ID.Eq(repositoryID)).First()
		if err != nil {
			return fmt.Errorf("迁移仓库 %d 的工作流：%w", repositoryID, err)
		}
		name := commonlogic.BuildRepositoryName(repository.Name, repository.Namespace)
		image := host + "/" + name
		ids := make([]int32, 0, len(group))
		for _, rule := range group {
			ids = append(ids, rule.ID)
			if rule.WorkflowTaskKey == "" {
				rule.WorkflowTaskKey = fmt.Sprintf("rule-%d", rule.ID)
			}
			if rule.ExecutionOrder < 1 {
				rule.ExecutionOrder = 1
			}
			if rule.TaskAction == "" {
				rule.TaskAction = ActionDeploy
			}
			if rule.ImageSource == "" {
				rule.ImageSource = SourceOriginal
			}
			if rule.TaskName == "" {
				rule.TaskName = rule.K8sAppName
			}

			if _, err := query.Where(query.ID.Eq(rule.ID)).Select(query.UserID, query.RepositoryName, query.ImageRepository, query.ImageHost, query.PlainHTTP, query.WorkflowTaskKey, query.ExecutionOrder, query.TaskAction, query.ImageSource, query.TaskName).Updates(entity.RegistryRepositoryDeployRule{WorkflowTaskKey: rule.WorkflowTaskKey, ExecutionOrder: rule.ExecutionOrder, TaskAction: rule.TaskAction, ImageSource: rule.ImageSource, TaskName: rule.TaskName, UserID: repository.UserID, RepositoryName: name, ImageRepository: image, ImageHost: host, PlainHTTP: registryURL.Scheme == "http"}); err != nil {
				return err
			}
		}
		config, err := l.Get(image)
		if err != nil {
			return err
		}
		if err := l.Save(image, config); err != nil {
			return fmt.Errorf("迁移镜像 %s 的工作流：%w", image, err)
		}
		// Keep the old linkage until logs are migrated so failures retry on next startup.
		if err := migrateWorkflowLogs(image, config, ids); err != nil {
			return err
		}
		if _, err := query.Where(query.ID.In(ids...)).Update(query.RepositoryID, 0); err != nil {
			return err
		}
	}
	return nil
}

func migrateWorkflowLogs(image string, config Config, ruleIDs []int32) error {
	logs := dao.Q.RegistryRepositoryDeployRuleMatchLog
	rules := dao.Q.RegistryRepositoryDeployRule
	migrated, err := rules.Where(rules.ID.In(ruleIDs...)).Find()
	if err != nil {
		return err
	}
	keys := map[int32]string{}
	for _, rule := range migrated {
		keys[rule.ID] = rule.WorkflowTaskKey
	}
	rows, err := logs.Where(logs.RuleID.In(ruleIDs...)).Find()
	if err != nil {
		return err
	}
	content, err := json.Marshal(config)
	if err != nil {
		return err
	}
	snapshot, err := function.AesEncrypt(string(content), function.GetMd5(facade.GetConfig().GetString("setting.secret")))
	if err != nil {
		return err
	}
	for _, row := range rows {
		if row.WorkflowRunID != "" {
			if _, err := logs.Where(logs.ID.Eq(row.ID)).Update(logs.TriggerRepository, image); err != nil {
				return err
			}
			continue
		}
		runID := function.GetMd5(fmt.Sprintf("legacy-deploy:%d", row.ID))
		if err := dao.Q.Transaction(func(tx *dao.Query) error {
			query := tx.RegistryRepositoryDeployRuleMatchLog
			coordinator := entity.RegistryRepositoryDeployRuleMatchLog{WorkflowRunID: runID, TriggerRepository: image, ImageName: row.ImageName, ExecutionStatus: "unknown", CreatedAt: row.CreatedAt, UpdatedAt: time.Now(), EncryptedConfigSnapshot: snapshot}
			if err := query.Create(&coordinator); err != nil {
				return err
			}
			_, err := query.Where(query.ID.Eq(row.ID)).Select(query.WorkflowRunID, query.WorkflowTaskKey, query.TriggerRepository, query.ExecutionStatus).Updates(entity.RegistryRepositoryDeployRuleMatchLog{WorkflowRunID: runID, WorkflowTaskKey: keys[row.RuleID], TriggerRepository: image, ExecutionStatus: "unknown"})
			return err
		}); err != nil {
			return err
		}
	}
	return nil
}
