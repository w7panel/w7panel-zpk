package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/w7panel/w7panel-zpk/app/registry/types"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	"github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sTypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type K8sPodInfo struct {
}

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

func (l Deploy) OnRepositoryPushed(payload types.RegistryRepositoryWebHookPayLoad) {
	repositoryName, namespace := Repository{}.ParseRepositoryNameAndNamespace(payload.Event.Target.Repository)
	repositoryModel, _ := Repository{}.GetByNameAndNamespace(repositoryName, namespace)
	if repositoryModel == nil {
		return
	}

	tag, _ := dao.Q.RegistryRepositoryTag.Where(dao.Q.RegistryRepositoryTag.RepositoryID.Eq(repositoryModel.ID)).
		Where(dao.Q.RegistryRepositoryTag.Name.Eq(payload.Event.Target.Tag)).First()
	if tag == nil {
		return
	}

	deployType := K8sDeployWhenTagAdd
	if !tag.LatestPushAt.IsZero() {
		deployType = K8sDeployWhenTagUpdate
	}

	deployRules, _ := dao.Q.RegistryRepositoryDeployRule.
		Where(dao.RegistryRepositoryDeployRule.RepositoryID.Eq(repositoryModel.ID)).
		Where(dao.Q.RegistryRepositoryDeployRule.DeployType.Eq(deployType)).Find()
	if len(deployRules) == 0 {
		return
	}

	imgName := fmt.Sprintf("%s/%s@%s", facade.GetConfig().GetString("registry_cli.default.external_domain"), Repository{}.BuildRepositoryName(repositoryName, namespace), payload.Event.Target.Digest)
	for _, rule := range deployRules {
		match := false
		if deployType == K8sDeployWhenTagAdd && rule.MatchType == K8sDeployRuleTagMatchRegular {
			reNum := regexp.MustCompile(rule.TagName)
			numbers := reNum.FindAllString(payload.Event.Target.Tag, 1)
			match = len(numbers) > 0
		} else if deployType == K8sDeployWhenTagAdd && rule.MatchType == K8sDeployRuleTagMatchPrefix {
			for _, itemRule := range strings.Split(rule.TagName, ",") {
				if strings.HasPrefix(payload.Event.Target.Tag, itemRule) {
					match = true
					break
				}
			}
		} else {
			match = slices.Contains(strings.Split(rule.TagName, ","), payload.Event.Target.Tag)
		}
		if match {
			l.deployK8sApp(rule, imgName)
		}
	}
}

func (l Deploy) deployK8sApp(rule *entity.RegistryRepositoryDeployRule, imgName string) {
	log := "prepare patch image: " + imgName + "\n"

	clientSet, err := l.GetK8sClient(rule.K8sConfig)
	if err != nil {
		log += fmt.Sprintf("get k8s client error: %v\n", err)
	}

	ctx := context.Background()
	containers := strings.Split(rule.K8sContainerName, ",")
	for _, container := range containers {
		if rule.K8sControllerType == K8sControllerTypeDaemonsets {
			patch := []byte(`{"spec":{"template":{"spec":{"containers":[{"name":"` + container + `","image":"` + imgName + `"}]}}}}`)
			result, err := clientSet.AppsV1().DaemonSets(rule.K8sNamespace).Patch(
				ctx,
				rule.K8sAppName,
				k8sTypes.StrategicMergePatchType,
				patch,
				metav1.PatchOptions{},
			)
			if err != nil {
				log += fmt.Sprintf("patch daemonsets container %s image err: %v \n", container, err)
			} else {
				statusContent, _ := json.Marshal(result.Status)
				log += fmt.Sprintf("patch daemonsets container %s image success, status: %s \n", container, string(statusContent))
			}
		}
		if rule.K8sControllerType == K8sControllerTypeDeployments {
			patch := []byte(`{"spec":{"template":{"spec":{"containers":[{"name":"` + container + `","image":"` + imgName + `"}]}}}}`)
			result, err := clientSet.AppsV1().Deployments(rule.K8sNamespace).Patch(
				ctx,
				rule.K8sAppName,
				k8sTypes.StrategicMergePatchType,
				patch,
				metav1.PatchOptions{},
			)
			if err != nil {
				log += fmt.Sprintf("patch deployments container %s image err: %v \n", container, err)
			} else {
				statusContent, _ := json.Marshal(result.Status)
				log += fmt.Sprintf("patch deployments container %s image success, status: %s \n", container, string(statusContent))
			}
		}
		if rule.K8sControllerType == K8sControllerTypeStatefulSet {
			patch := []byte(`{"spec":{"template":{"spec":{"containers":[{"name":"` + container + `","image":"` + imgName + `"}]}}}}`)
			result, err := clientSet.AppsV1().StatefulSets(rule.K8sNamespace).Patch(
				ctx,
				rule.K8sAppName,
				k8sTypes.StrategicMergePatchType,
				patch,
				metav1.PatchOptions{},
			)
			if err != nil {
				log += fmt.Sprintf("patch statefulsets container %s image err: %v \n", container, err)
			} else {
				statusContent, _ := json.Marshal(result.Status)
				log += fmt.Sprintf("patch statefulsets container %s image success, status: %s \n", container, string(statusContent))
			}
		}
	}

	err = dao.Q.Transaction(func(tx *dao.Query) error {
		_, err = tx.RegistryRepositoryDeployRule.Where(tx.RegistryRepositoryDeployRule.ID.Eq(rule.ID)).Updates(entity.RegistryRepositoryDeployRule{
			LatestTriggerAt: time.Now(),
		})
		if err != nil {
			return err
		}

		err = tx.RegistryRepositoryDeployRuleMatchLog.Create(&entity.RegistryRepositoryDeployRuleMatchLog{
			RuleID:    rule.ID,
			ImageName: imgName,
			K8sLog:    log,
		})
		return err
	})
	if err != nil {
		slog.Error("add registry repository deploy rule log err", "rule", rule, "image", imgName, "err", err)
	}
}

func (l Deploy) MakeK8sConfig(k8sConfig string) (*rest.Config, error) {
	var config *rest.Config
	var err error
	if k8sConfig != "" {
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

	return kubernetes.NewForConfig(config)
}
