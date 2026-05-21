package types

import (
	"github.com/w7panel/w7panel-zpk/common/entity"
)

var RegistryRepositoryPrepareEvent = "registry_repository"
var RegistryRepositoryPushedEvent = "registry_repository_pushed"
var RegistryRepositoryPulledEvent = "registry_repository_pulled"
var RegistryRepositoryDeletedEvent = "registry_repository_deleted"
var RegistryRepositoryMountedEvent = "registry_repository_mounted"

var RegistryRepositoryAfterPushedEvent = "registry_repository_after_pushed"

type RegistryRepositoryPayLoad struct {
	User  *entity.RegistryUser
	Scope PermissionScope
}

type RegistryRepositoryWebHookPayLoad struct {
	Event Event
}
