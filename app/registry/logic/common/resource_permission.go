package common

import (
	"context"
	"errors"
	"slices"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/common/accessor"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	userlogic "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/w7panel/w7panel-zpk/common/types/registry"
	"gorm.io/gorm"
)

type PermissionResourceType string

const (
	PermissionResourceTypeNamespace  PermissionResourceType = "namespace"
	PermissionResourceTypeRepository PermissionResourceType = "repository"
)

type PermissionActionType string

const (
	PermissionActionTypePush PermissionActionType = "push"
	PermissionActionTypePull PermissionActionType = "pull"
)

type ResourcePermissionResource struct {
	Type          string   `json:"type" binding:"required,max=64,excludesall= \t\x00\r\n"`
	Value         string   `json:"value" binding:"required,max=512,excludesall=\x00\r\n"`
	PublicActions []string `json:"-"`
}

type ResourcePermissionGrant struct {
	UserID   int32                      `json:"user_id" binding:"min=0"`
	Resource ResourcePermissionResource `json:"resource" binding:"required"`
	Actions  []string                   `json:"actions" binding:"dive,required,max=64,excludesall= \t\x00\r\n"`
}

type ResourcePermissionDecision struct {
	Allowed  bool `json:"allowed"`
	Explicit bool `json:"explicit"`
}

type ResourcePermission struct{}

// CheckPermission 只校验准备好的公共动作和用户授权，不查询或继承其他资源权限。
func (l ResourcePermission) CheckPermission(resource ResourcePermissionResource, permission *entity.RegistryUserPermission, actions []string) ResourcePermissionDecision {
	if len(actions) > 0 && ValidatePermissionActions(actions, resource.PublicActions) {
		return ResourcePermissionDecision{Allowed: true}
	}
	if permission == nil {
		return ResourcePermissionDecision{}
	}
	return ResourcePermissionDecision{
		Allowed:  permission.Action != nil && ValidatePermissionActions(actions, *permission.Action),
		Explicit: true,
	}
}

// CheckResource 按资源完整路径组合精确授权和 namespace 授权。
func (l ResourcePermission) CheckResource(ctx context.Context, user *entity.RegistryUser, resource ResourcePermissionResource, actions []string) (ResourcePermissionDecision, error) {
	if user == nil || (len(actions) > 0 && ValidatePermissionActions(actions, resource.PublicActions)) {
		return l.CheckPermission(resource, nil, actions), nil
	}
	permission, err := l.GetUserPermission(ctx, user.ID, resource)
	if err != nil {
		return ResourcePermissionDecision{}, err
	}
	if permission == nil && resource.Type != string(PermissionResourceTypeNamespace) {
		// 任何精确授权都会阻止 push 使用 namespace 兜底，pull 保持原有规则。
		configured := false
		if slices.Contains(actions, string(PermissionActionTypePush)) {
			query := dao.Q.RegistryUserPermission
			grant, err := query.WithContext(ctx).Select(query.ID).Where(query.ResourceType.Eq(resource.Type), query.ResourceValue.Eq(resource.Value)).First()
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return ResourcePermissionDecision{}, err
			}
			configured = grant != nil
		}
		if !configured {
			_, namespace := userlogic.ParseRepositoryNameAndNamespace(resource.Value)
			permission, err = l.GetUserPermission(ctx, user.ID, ResourcePermissionResource{Type: string(PermissionResourceTypeNamespace), Value: namespace})
			if err != nil {
				return ResourcePermissionDecision{}, err
			}
		}
	}
	return l.CheckPermission(resource, permission, actions), nil
}

func (l ResourcePermission) GetUserPermission(ctx context.Context, userID int32, resource ResourcePermissionResource) (*entity.RegistryUserPermission, error) {
	query := dao.Q.RegistryUserPermission
	permission, err := query.WithContext(ctx).Where(query.UserID.Eq(userID), query.ResourceType.Eq(resource.Type), query.ResourceValue.Eq(resource.Value)).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return permission, err
}

func ValidatePermissionActions(requested, permitted []string) bool {
	for _, action := range requested {
		if !slices.Contains(permitted, action) {
			return false
		}
	}
	return true
}

func (l ResourcePermission) UserHasPermissionByResource(userId int, resourceValue string, resourceType PermissionResourceType) bool {
	exists, _ := dao.Q.RegistryUserPermission.
		Where(dao.Q.RegistryUserPermission.UserID.Eq(int32(userId))).
		Where(dao.Q.RegistryUserPermission.ResourceType.Eq(string(resourceType))).
		Where(dao.Q.RegistryUserPermission.ResourceValue.Eq(resourceValue)).First()

	return exists != nil
}

func (l ResourcePermission) AddUserPermission(userId int32, resourceValue string, resourceType PermissionResourceType, actions accessor.PermissionActionOption) error {
	return dao.Q.RegistryUserPermission.Create(&entity.RegistryUserPermission{
		UserID:        userId,
		ResourceType:  string(resourceType),
		ResourceValue: resourceValue,
		Action:        &actions,
	})
}

func (l ResourcePermission) DelUserPermission(userId int32, resourceValue string, resourceType PermissionResourceType) error {
	_, err := dao.Q.RegistryUserPermission.
		Where(dao.Q.RegistryUserPermission.UserID.Eq(userId)).
		Where(dao.Q.RegistryUserPermission.ResourceType.Eq(string(resourceType))).
		Where(dao.Q.RegistryUserPermission.ResourceValue.Eq(resourceValue)).Delete()
	return err
}

func (l ResourcePermission) ClearUserPermission(userId int32) error {
	_, err := dao.Q.RegistryUserPermission.
		Where(dao.Q.RegistryUserPermission.UserID.Eq(userId)).Delete()
	return err
}

func (l ResourcePermission) DelPermissionByResource(tx *dao.Query, resourceValue string, resourceType PermissionResourceType) error {
	_, err := tx.RegistryUserPermission.
		Where(tx.RegistryUserPermission.ResourceType.Eq(string(resourceType))).
		Where(tx.RegistryUserPermission.ResourceValue.Eq(resourceValue)).Delete()
	return err
}

func (l ResourcePermission) GetNamespacePermissions(namespace string) ([]*entity.RegistryUserPermission, error) {
	return dao.Q.RegistryUserPermission.
		Where(dao.Q.RegistryUserPermission.ResourceType.Eq(string(PermissionResourceTypeNamespace))).
		Where(dao.Q.RegistryUserPermission.ResourceValue.Eq(namespace)).
		Find()
}

func (l ResourcePermission) ReplaceNamespacePermissions(namespace string, permissions []registry.NamespacePermissionItem) error {
	return dao.Q.Transaction(func(tx *dao.Query) error {
		_, err := tx.RegistryUserPermission.
			Where(tx.RegistryUserPermission.ResourceType.Eq(string(PermissionResourceTypeNamespace))).
			Where(tx.RegistryUserPermission.ResourceValue.Eq(namespace)).
			Delete()
		if err != nil {
			return err
		}

		if len(permissions) == 0 {
			return nil
		}

		userPermissions := make([]*entity.RegistryUserPermission, 0, len(permissions))
		for _, item := range permissions {
			userPermissions = append(userPermissions, &entity.RegistryUserPermission{
				UserID:        int32(item.UserID),
				ResourceValue: namespace,
				ResourceType:  string(PermissionResourceTypeNamespace),
				Action:        (*accessor.PermissionActionOption)(&item.Actions),
			})
		}

		return tx.RegistryUserPermission.CreateInBatches(userPermissions, 10)
	})
}

func (l ResourcePermission) IsCanOperate(ctx *gin.Context, curUid int32) bool {
	user := userlogic.User{}.GetUser(ctx)
	if user == nil || (userlogic.User{}.IsAdminUser(user)) || curUid == user.ID {
		return true
	}
	return false
}
