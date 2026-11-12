package logic

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	"github.com/w7panel/w7panel-zpk/common/function"
	"github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/w7panel/w7panel-zpk/common/service"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"gorm.io/gorm"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	K8sDeployWhenTagUpdate = int32(1)
	K8sDeployWhenTagAdd    = int32(2)
)
const (
	K8sDeployRuleTagMatchPrefix  = int32(1)
	K8sDeployRuleTagMatchRegular = int32(2)
)

const (
	K8sControllerTypeDeployments = "deployments"
	K8sControllerTypeDaemonsets  = "daemonsets"
	K8sControllerTypeStatefulSet = "statefulsets"
)

type Deploy struct {
	logic.Logic
}

// DeployLegacy preserves direct deployment for rules not yet converted to workflows.
func (l Deploy) DeployLegacy(repositoryID int32, tagName, imageReference string, eventType int) {
	// 兼容尚未转换为工作流的旧部署规则，沿用原有 tag 匹配和直接部署逻辑。
	// 已转换的规则（WorkflowTaskKey 非空）由 Workflow.Trigger 通过 Helm 执行，此处跳过以避免重复部署。
	deployType := int32(eventType)

	deployRules, _ := dao.Q.RegistryRepositoryDeployRule.
		Where(dao.RegistryRepositoryDeployRule.RepositoryID.Eq(repositoryID)).
		Where(dao.Q.RegistryRepositoryDeployRule.DeployType.Eq(deployType)).Find()
	if len(deployRules) == 0 {
		return
	}

	for _, rule := range deployRules {
		if rule.WorkflowTaskKey != "" {
			continue
		}
		matchesTagRule := false
		if deployType == K8sDeployWhenTagAdd && rule.MatchType == K8sDeployRuleTagMatchRegular {
			tagPattern, err := regexp.Compile(rule.TagName)
			if err != nil {
				slog.Error("invalid deploy rule regex", "rule", rule.ID, "error", err)
				continue
			}
			tagMatches := tagPattern.FindAllString(tagName, 1)
			matchesTagRule = len(tagMatches) > 0
		} else if deployType == K8sDeployWhenTagAdd && rule.MatchType == K8sDeployRuleTagMatchPrefix {
			for _, tagPrefix := range strings.Split(rule.TagName, ",") {
				if strings.HasPrefix(tagName, tagPrefix) {
					matchesTagRule = true
					break
				}
			}
		} else {
			matchesTagRule = slices.Contains(strings.Split(rule.TagName, ","), tagName)
		}
		if matchesTagRule {
			l.deployK8sApp(rule, imageReference)
		}
	}
}

func (l Deploy) deployK8sApp(rule *entity.RegistryRepositoryDeployRule, imageReference string) {
	log := "prepare patch image: " + imageReference + "\n"

	clientSet, err := l.GetK8sClient(rule.K8sConfig)
	if err != nil {
		log += fmt.Sprintf("get k8s client error: %v\n", err)
	}

	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		cluster := service.Kubernetes{Client: clientSet}
		for _, container := range strings.Split(rule.K8sContainerName, ",") {
			status, err := cluster.PatchImage(ctx, rule.K8sNamespace, rule.K8sControllerType, rule.K8sAppName, container, imageReference)
			if err != nil {
				log += fmt.Sprintf("patch %s container %s image err: %v \n", rule.K8sControllerType, container, err)
			} else {
				log += fmt.Sprintf("patch %s container %s image success, status: %s \n", rule.K8sControllerType, container, status)
			}
		}
	}

	err = dao.Q.Transaction(func(tx *dao.Query) error {
		curTime := time.Now()
		_, err = tx.RegistryRepositoryDeployRule.Where(tx.RegistryRepositoryDeployRule.ID.Eq(rule.ID)).Updates(entity.RegistryRepositoryDeployRule{
			LatestTriggerAt: &curTime,
		})
		if err != nil {
			return err
		}

		err = tx.RegistryRepositoryDeployRuleMatchLog.Create(&entity.RegistryRepositoryDeployRuleMatchLog{
			RuleID:    rule.ID,
			ImageName: imageReference,
			K8sLog:    log,
		})
		return err
	})
	if err != nil {
		slog.Error("add registry repository deploy rule log err", "rule_id", rule.ID, "image", imageReference, "err", err)
	}
}

func (l Deploy) MakeK8sConfig(k8sConfig string) (*rest.Config, error) {
	k8sConfig, err := l.DecryptK8sConfig(k8sConfig)
	if err != nil {
		return nil, err
	}
	var config *rest.Config
	if k8sConfig != "" {
		if err := l.ValidateK8sConfig(k8sConfig); err != nil {
			return nil, err
		}
		config, err = clientcmd.RESTConfigFromKubeConfig([]byte(k8sConfig))
		if err != nil {
			return nil, err
		}
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	}

	return config, nil
}

func (l Deploy) GetK8sClient(k8sConfig string) (*kubernetes.Clientset, error) {
	config, err := l.MakeK8sConfig(k8sConfig)
	if err != nil {
		return nil, err
	}

	return (service.Kubernetes{}).NewClient(config)
}

const encryptedK8sConfigPrefix = "aes:"

// EncryptK8sConfig leaves the empty in-cluster configuration unchanged.
func (l Deploy) EncryptK8sConfig(content string) (string, error) {
	if content == "" {
		return "", nil
	}
	// Normalize an encrypted input before saving to avoid double encryption.
	plain, err := l.DecryptK8sConfig(content)
	if err != nil {
		return "", err
	}
	key := function.GetMd5(facade.GetConfig().GetString("setting.secret"))
	encrypted, err := function.AesEncrypt(plain, key)
	if err != nil {
		return "", errors.New("加密部署规则 KUBECONFIG 失败")
	}
	return encryptedK8sConfigPrefix + encrypted, nil
}

// DecryptK8sConfig accepts historical plaintext; marked ciphertext must decrypt successfully.
func (l Deploy) DecryptK8sConfig(content string) (string, error) {
	if !strings.HasPrefix(content, encryptedK8sConfigPrefix) {
		return content, nil
	}
	key := function.GetMd5(facade.GetConfig().GetString("setting.secret"))
	plain, err := function.AesDecrypt(strings.TrimPrefix(content, encryptedK8sConfigPrefix), key)
	if err != nil || plain == "" {
		return "", errors.New("解密部署规则 KUBECONFIG 失败，请检查 setting.secret 或重新保存配置")
	}
	return plain, nil
}

// ValidateK8sConfig keeps credential loading self-contained inside task Secrets.
func (l Deploy) ValidateK8sConfig(content string) error {
	cfg, err := clientcmd.Load([]byte(content))
	if err != nil || clientcmd.Validate(*cfg) != nil {
		return errors.New("KUBECONFIG 无效")
	}
	for _, cluster := range cfg.Clusters {
		if cluster.CertificateAuthority != "" {
			return errors.New("请使用内嵌 CA 数据的 KUBECONFIG")
		}
	}
	for _, auth := range cfg.AuthInfos {
		if auth.Exec != nil || auth.AuthProvider != nil || auth.TokenFile != "" || auth.ClientCertificate != "" || auth.ClientKey != "" {
			return errors.New("KUBECONFIG 不支持外部认证程序或本地凭证文件")
		}
	}
	return nil
}

func (l Deploy) Rule(ruleID int32) (*entity.RegistryRepositoryDeployRule, error) {
	query := dao.Q.RegistryRepositoryDeployRule
	rule, err := query.Where(query.ID.Eq(ruleID)).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("该规则不存在")
	}
	return rule, err
}

func (l Deploy) SaveRule(rule entity.RegistryRepositoryDeployRule) error {
	workflow := Workflow{}
	if err := workflow.checkWritesAllowed(); err != nil {
		return err
	}
	if rule.ID != 0 {
		existing, err := l.Rule(rule.ID)
		if err != nil {
			return err
		}
		if err := workflow.CheckLegacyRuleWritable(existing.RepositoryID); err != nil {
			return err
		}
	}
	if err := workflow.CheckLegacyRuleWritable(rule.RepositoryID); err != nil {
		return err
	}
	encrypted, err := l.EncryptK8sConfig(rule.K8sConfig)
	if err != nil {
		return err
	}
	rule.K8sConfig = encrypted
	query := dao.Q.RegistryRepositoryDeployRule
	if rule.ID == 0 {
		return query.Create(&rule)
	}
	_, err = query.Where(query.ID.Eq(rule.ID)).Select(query.RepositoryID, query.DeployType, query.MatchType, query.K8sNamespace, query.K8sControllerType, query.K8sAppName, query.K8sContainerName, query.TagName, query.K8sConfig).Updates(rule)
	return err
}

func (l Deploy) RuleInfo(rule *entity.RegistryRepositoryDeployRule) (*entity.RegistryRepositoryDeployRule, error) {
	decryptedRule := *rule
	kubeconfig, err := l.DecryptK8sConfig(decryptedRule.K8sConfig)
	if err != nil {
		return nil, err
	}
	decryptedRule.K8sConfig = kubeconfig
	return &decryptedRule, nil
}

func (l Deploy) DeleteRule(rule *entity.RegistryRepositoryDeployRule) error {
	workflow := Workflow{}
	if err := workflow.checkWritesAllowed(); err != nil {
		return err
	}
	if err := workflow.CheckLegacyRuleWritable(rule.RepositoryID); err != nil {
		return err
	}
	return dao.Q.Transaction(func(tx *dao.Query) error {
		query := tx.RegistryRepositoryDeployRuleMatchLog
		if _, err := query.Where(query.RuleID.Eq(rule.ID)).Delete(); err != nil {
			return err
		}
		_, err := tx.RegistryRepositoryDeployRule.Delete(rule)
		return err
	})
}

func (l Deploy) Rules(repositoryID int32) ([]*entity.RegistryRepositoryDeployRule, error) {
	query := dao.Q.RegistryRepositoryDeployRule
	rules, err := query.Where(query.RepositoryID.Eq(repositoryID)).Order(query.ID.Desc()).Find()
	if err != nil {
		return nil, err
	}
	for _, rule := range rules {
		rule.K8sConfig, err = l.DecryptK8sConfig(rule.K8sConfig)
		if err != nil {
			return nil, err
		}
	}
	return rules, nil
}

func (l Deploy) RuleLogs(ruleID int32, page, pageSize int) ([]*entity.RegistryRepositoryDeployRuleMatchLog, int64, error) {
	query := dao.Q.RegistryRepositoryDeployRuleMatchLog
	return query.Where(query.RuleID.Eq(ruleID)).Order(query.ID.Desc()).FindByPage((page-1)*pageSize, pageSize)
}

func (l Deploy) Namespaces(ctx context.Context, kubeconfig string) ([]string, error) {
	client, err := l.GetK8sClient(kubeconfig)
	if err != nil {
		return nil, err
	}
	return (service.Kubernetes{Client: client}).Namespaces(ctx)
}

func (l Deploy) Applications(ctx context.Context, kubeconfig, namespace, controllerType string) ([]string, error) {
	client, err := l.GetK8sClient(kubeconfig)
	if err != nil {
		return nil, err
	}
	return (service.Kubernetes{Client: client}).Applications(ctx, namespace, controllerType)
}

func (l Deploy) Containers(ctx context.Context, kubeconfig, namespace, controllerType, appName string) ([]string, error) {
	client, err := l.GetK8sClient(kubeconfig)
	if err != nil {
		return nil, err
	}
	return (service.Kubernetes{Client: client}).Containers(ctx, namespace, controllerType, appName)
}

func (l Deploy) Proxy(writer http.ResponseWriter, request *http.Request, path string) error {
	config, err := l.MakeK8sConfig("")
	if err != nil {
		return err
	}
	return (service.Kubernetes{}).Proxy(config, writer, request, path)
}
