package controller

import (
	"errors"

	"github.com/gin-gonic/gin"
	registrylogic "github.com/w7panel/w7panel-zpk/app/registry/logic"
	"github.com/w7panel/w7panel-zpk/app/registry/types"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/controller"
)

type Permission struct {
	controller.Abstract
}

func (c Permission) GetNamespacePermission(ctx *gin.Context) {
	type ParamsValidate struct {
		Namespace string `json:"namespace" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	namespace, err := registrylogic.Namespace{}.GetByName(params.Namespace)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	if namespace == nil {
		c.JsonResponseWithServerError(ctx, errors.New("namespace 不存在"))
		return
	}

	permissions, err := registrylogic.Permission{}.GetNamespacePermissions(params.Namespace)
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

	list := make([]types.NamespacePermissionItem, 0, len(permissions))
	for _, item := range permissions {
		user := userMap[item.UserID]
		if user == nil {
			continue
		}

		actions := make([]string, 0)
		if item.Action != nil {
			actions = *item.Action
		}

		list = append(list, types.NamespacePermissionItem{
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

func (c Permission) SetNamespacePermission(ctx *gin.Context) {
	type ParamsValidate struct {
		Namespace   string                          `json:"namespace" binding:"required"`
		Permissions []types.NamespacePermissionItem `json:"permissions"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	namespace, err := registrylogic.Namespace{}.GetByName(params.Namespace)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	if namespace == nil {
		c.JsonResponseWithServerError(ctx, errors.New("namespace 不存在"))
		return
	}

	if err = (registrylogic.Permission{}).ReplaceNamespacePermissions(params.Namespace, params.Permissions); err != nil {
		c.JsonResponseWithServerError(ctx, errors.New("设置 namespace 权限失败, err:"+err.Error()))
		return
	}

	c.JsonSuccessResponse(ctx)
}
