package common

import (
	"errors"
	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
	"gorm.io/gorm"
	"slices"

	"github.com/gin-gonic/gin"

	registrycommon "github.com/w7panel/w7panel-zpk/app/registry/logic/common"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	"github.com/w7panel/w7panel-zpk/common/types/registry"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/controller"
)

type NamespacePermission struct {
	controller.Abstract
}

func (c NamespacePermission) GetNamespacePermission(ctx *gin.Context) {
	type ParamsValidate struct {
		Namespace string `json:"namespace" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	namespace, err := registrycommon.Namespace{}.GetByName(params.Namespace)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	if namespace == nil {
		c.JsonResponseWithServerError(ctx, errors.New("namespace 不存在"))
		return
	}

	permissions, err := registrycommon.ResourcePermission{}.GetNamespacePermissions(params.Namespace)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	userIDs := make([]int32, 0, len(permissions))
	for _, item := range permissions {
		userIDs = append(userIDs, item.UserID)
	}

	userMap := make(map[int32]*entity.RegistryUser, len(userIDs))
	if len(userIDs) > 0 {
		users, userErr := dao.Q.RegistryUser.
			Where(dao.Q.RegistryUser.ID.In(userIDs...)).
			Find()
		if userErr != nil {
			c.JsonResponseWithServerError(ctx, userErr)
			return
		}
		for _, user := range users {
			userMap[user.ID] = user
		}
	}

	list := make([]registry.NamespacePermissionItem, 0, len(permissions))
	for _, item := range permissions {
		user := userMap[item.UserID]
		if user == nil {
			continue
		}

		actions := make([]string, 0)
		if item.Action != nil {
			actions = *item.Action
		}

		list = append(list, registry.NamespacePermissionItem{
			UserID:   int(item.UserID),
			UserName: user.Username,
			Actions:  actions,
		})
	}

	c.JsonResponseWithoutError(ctx, map[string]interface{}{
		"namespace":   params.Namespace,
		"permissions": list,
	})
}

func (c NamespacePermission) SetNamespacePermission(ctx *gin.Context) {
	type ParamsValidate struct {
		Namespace   string                             `json:"namespace" binding:"required"`
		Permissions []registry.NamespacePermissionItem `json:"permissions"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	namespace, err := registrycommon.Namespace{}.GetByName(params.Namespace)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	if namespace == nil {
		c.JsonResponseWithServerError(ctx, errors.New("namespace 不存在"))
		return
	}

	if err = (registrycommon.ResourcePermission{}).ReplaceNamespacePermissions(params.Namespace, params.Permissions); err != nil {
		c.JsonResponseWithServerError(ctx, errors.New("设置 namespace 权限失败, err:"+err.Error()))
		return
	}

	c.JsonSuccessResponse(ctx)
}

func CheckNamespacePermission(ctx *gin.Context, namespace string, requirePush bool) error {
	namespaceModel, err := registrycommon.Namespace{}.GetByName(namespace)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("namespace 不存在")
		}
		return err
	}
	if namespaceModel == nil {
		return errors.New("namespace 不存在")
	}

	user := logic2.User{}.GetUser(ctx)
	if user == nil {
		return errors.New("用户信息异常")
	}
	if (logic2.User{}).IsAdminUser(user) || namespaceModel.UserID == user.ID {
		return nil
	}

	permission, err := dao.Q.RegistryUserPermission.
		Where(dao.Q.RegistryUserPermission.UserID.Eq(user.ID)).
		Where(dao.Q.RegistryUserPermission.ResourceType.Eq(string(registrycommon.PermissionResourceTypeNamespace))).
		Where(dao.Q.RegistryUserPermission.ResourceValue.Eq(namespace)).
		First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user no permission")
		}
		return err
	}
	if permission == nil {
		return errors.New("user no permission")
	}
	if requirePush {
		if permission.Action == nil || !slices.Contains(*permission.Action, string(registrycommon.PermissionActionTypePush)) {
			return errors.New("invalid action")
		}
	}

	return nil
}
