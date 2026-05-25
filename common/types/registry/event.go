package registry

import (
	"github.com/w7panel/w7panel-zpk/common/entity"
)

var RegistryRepositoryPrepareEvent = "registry_repository"
var RegistryRepositoryPushedEvent = "registry_repository_pushed"
var RegistryRepositoryPulledEvent = "registry_repository_pulled"
var RegistryRepositoryDeletedEvent = "registry_repository_deleted"
var RegistryRepositoryMountedEvent = "registry_repository_mounted"

var RegistryRepositoryAfterPushedEvent = "registry_repository_after_pushed"

var AddUserPermissionEvent = "add_user_permission"
var DeleteUserPermissionEvent = "delete_user_permission"
var ClearUserPermissionEvent = "clear_user_permission"

type RegistryRepositoryPayLoad struct {
	User  *entity.RegistryUser
	Scope PermissionScope
}

type RegistryRepositoryWebHookPayLoad struct {
	Event Event
}

type ClearUserPermissionPayload struct {
	UserID int32
}

type AddUserPermissionPayload struct {
	UserID        int32
	ResourceType  string
	ResourceValue string
	Actions       []string
}

type DelUserPermissionPayload struct {
	UserID        int32
	ResourceType  string
	ResourceValue string
}
