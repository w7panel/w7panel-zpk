package logic

import (
	workflow "github.com/w7panel/w7panel-zpk/app/workflow/logic"
	"github.com/w7panel/w7panel-zpk/common/dao"
	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/w7panel/w7panel-zpk/common/types/registry"
	"log/slog"
)

// Compatibility adapters for existing callers; implementation belongs to workflow.
type Deploy struct{ workflow.Deploy }

const (
	K8sDeployWhenTagUpdate = workflow.K8sDeployWhenTagUpdate
	K8sDeployWhenTagAdd    = workflow.K8sDeployWhenTagAdd
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
	event := workflow.ImagePushed{
		Repository: payload.Event.Target.Repository,
		Tag:        payload.Event.Target.Tag, Digest: payload.Event.Target.Digest,
		Actor: payload.Event.Actor.Name, EventID: payload.Event.ID, EventType: int(eventType),
	}

	workflowLogic := workflow.Workflow{}
	// OnImagePushed enqueues work; Helm execution happens in the background executor.
	pushResult, err := workflowLogic.OnImagePushed(event)
	if err != nil {
		slog.Error("process workflow image push", "error", err)
		return
	}
	if err := (Repository{}).SetRebuilt(repository.ID, event.Tag, pushResult.Rebuilt); err != nil {
		slog.Error("record rebuilt image", "error", err)
		return
	}
}

// Historical ciphertext marker; encryption implementation belongs to workflow.
const encryptedK8sConfigPrefix = "aes:"
