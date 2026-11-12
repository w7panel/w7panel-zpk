package logic

import (
	workflow "github.com/w7panel/w7panel-zpk/app/workflow/logic"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/w7panel/w7panel-zpk/common/types/registry"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"log/slog"
	"net/url"
	"strings"
)

// Compatibility adapters for existing callers; implementation belongs to workflow.
type Deploy struct{ workflow.Deploy }

const (
	K8sDeployWhenTagUpdate       = workflow.K8sDeployWhenTagUpdate
	K8sDeployWhenTagAdd          = workflow.K8sDeployWhenTagAdd
	K8sDeployRuleTagMatchPrefix  = workflow.K8sDeployRuleTagMatchPrefix
	K8sDeployRuleTagMatchRegular = workflow.K8sDeployRuleTagMatchRegular
	K8sControllerTypeDeployments = workflow.K8sControllerTypeDeployments
	K8sControllerTypeDaemonsets  = workflow.K8sControllerTypeDaemonsets
	K8sControllerTypeStatefulSet = workflow.K8sControllerTypeStatefulSet
)

func (Deploy) OnRepositoryPushed(payload registry.RegistryRepositoryWebHookPayLoad) {
	if payload.Event.Target.Tag == "" {
		return
	}
	name, namespace := commonlogic.ParseRepositoryNameAndNamespace(payload.Event.Target.Repository)
	repository, err := (Repository{}).GetByNameAndNamespace(name, namespace)
	if err != nil || repository == nil {
		return
	}
	tag, err := dao.Q.RegistryRepositoryTag.Where(dao.Q.RegistryRepositoryTag.RepositoryID.Eq(repository.ID), dao.Q.RegistryRepositoryTag.Name.Eq(payload.Event.Target.Tag)).First()
	if err != nil || tag == nil {
		return
	}
	eventType := K8sDeployWhenTagAdd
	if tag.LatestPushAt != nil && !tag.LatestPushAt.IsZero() {
		eventType = K8sDeployWhenTagUpdate
	}
	owner := Repository{}
	host := strings.TrimRight(facade.GetConfig().GetString("registry_cli.default.external_domain"), "/")
	imageRepository := host + "/" + payload.Event.Target.Repository
	nativeURL, _ := url.Parse(facade.GetConfig().GetString("registry_cli.default.url"))
	image := workflow.ImageContext{
		Reference:  imageRepository + "@" + payload.Event.Target.Digest,
		Repository: imageRepository, Host: host,
		PlainHTTP: nativeURL != nil && nativeURL.Scheme == "http",
	}
	event := workflow.ImagePushed{
		RepositoryID: repository.ID, Repository: payload.Event.Target.Repository,
		Tag: payload.Event.Target.Tag, Digest: payload.Event.Target.Digest,
		Actor: payload.Event.Actor.Name, EventID: payload.Event.ID, EventType: int(eventType), Image: image,
	}
	runID, err := event.RunID()
	if err != nil {
		slog.Error("identify image workflow", "error", err)
		return
	}
	event.RegistryToken, err = (Permission{}).CreateToken(&entity.RegistryUser{Username: workflow.ActorPrefix + runID}, []registry.PermissionScope{{Type: "repository", Name: event.Repository, Actions: []string{"pull", "push"}}})
	if err != nil {
		slog.Error("create workflow registry token", "error", err)
		return
	}
	helper := workflow.Workflow{}
	// OnImagePushed enqueues work; Helm execution happens in the background executor.
	result, err := helper.OnImagePushed(event)
	if err != nil {
		slog.Error("process workflow image push", "error", err)
		return
	}
	if err := owner.SetRebuilt(repository.ID, event.Tag, result.Rebuilt); err != nil {
		slog.Error("record rebuilt image", "error", err)
		return
	}
	if !result.Rebuilt {
		(workflow.Deploy{}).DeployLegacy(repository.ID, payload.Event.Target.Tag, image.Reference, int(eventType))
	}
}

// Historical ciphertext marker; encryption implementation belongs to workflow.
const encryptedK8sConfigPrefix = "aes:"
