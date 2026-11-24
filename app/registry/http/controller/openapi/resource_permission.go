package openapi

import (
	"errors"

	"github.com/gin-gonic/gin"
	registrylogic "github.com/w7panel/w7panel-zpk/app/registry/logic"
	"github.com/w7panel/w7panel-zpk/common/entity"
	userlogic "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/controller"
	"gorm.io/gorm"
)

type ResourcePermission struct{ controller.Abstract }

func (c ResourcePermission) Check(ctx *gin.Context) {
	type ParamsValidate struct {
		UserID   int32                                    `json:"user_id" binding:"min=0"`
		Resource registrylogic.ResourcePermissionResource `json:"resource" binding:"required"`
		Action   string                                   `json:"action" binding:"required,max=64,excludesall= \t\x00\r\n"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	var user *entity.RegistryUser
	if params.UserID > 0 {
		var err error
		user, err = (userlogic.User{}).GetById(int(params.UserID))
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JsonResponseWithError(ctx, errors.New("用户不存在"), 403)
			return
		}
		if err != nil {
			c.JsonResponseWithError(ctx, err, 503)
			return
		}
		if user == nil {
			c.JsonResponseWithError(ctx, errors.New("用户不存在"), 403)
			return
		}
	}
	permissions := registrylogic.ResourcePermission{Admin: user != nil && (userlogic.User{}).IsAdminUser(user)}
	if !permissions.Admin {
		publicPermission, err := permissions.GetUserPermission(ctx.Request.Context(), 0, params.Resource)
		if err != nil {
			c.respondError(ctx, err)
			return
		}
		if publicPermission != nil && publicPermission.Action != nil {
			params.Resource.PublicActions = []string(*publicPermission.Action)
		}
	}
	decision, err := permissions.CheckResource(ctx.Request.Context(), user, params.Resource, []string{params.Action})
	if err != nil {
		c.respondError(ctx, err)
		return
	}
	c.JsonResponseWithoutError(ctx, decision)
}

func (c ResourcePermission) Set(ctx *gin.Context) {
	type ParamsValidate struct {
		Grants []registrylogic.ResourcePermissionGrant `json:"grants" binding:"required,min=1,max=500,dive"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	if err := (registrylogic.ResourcePermission{}).Set(ctx.Request.Context(), params.Grants); err != nil {
		c.respondError(ctx, err)
		return
	}
	c.JsonSuccessResponse(ctx)
}

func (c ResourcePermission) Get(ctx *gin.Context) {
	type ParamsValidate struct {
		Resource registrylogic.ResourcePermissionResource `json:"resource" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	grants, err := (registrylogic.ResourcePermission{}).Get(ctx.Request.Context(), params.Resource)
	if err != nil {
		c.respondError(ctx, err)
		return
	}
	c.JsonResponseWithoutError(ctx, grants)
}

func (c ResourcePermission) respondError(ctx *gin.Context, err error) {
	status := 503
	if errors.Is(err, registrylogic.ErrResourcePermissionInvalid) {
		status = 400
	}
	c.JsonResponseWithError(ctx, err, status)
}
