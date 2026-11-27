package openapi

import (
	"errors"
	"github.com/gin-gonic/gin"

	registrycommon "github.com/w7panel/w7panel-zpk/app/registry/logic/common"
	"github.com/w7panel/w7panel-zpk/common/entity"
	userlogic "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/w7panel/w7panel-zpk/common/types/registry"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/controller"
	"gorm.io/gorm"
)

type Resource struct{ controller.Abstract }

func (c Resource) Get(ctx *gin.Context) {
	var params struct {
		Resource registrycommon.ResourcePermissionResource `json:"resource" binding:"required"`
	}
	if !c.Validate(ctx, &params) {
		return
	}
	result, err := (registrycommon.Resource{}).Get(ctx.Request.Context(), params.Resource)
	if err != nil {
		c.respondError(ctx, err)
		return
	}
	c.JsonResponseWithoutError(ctx, result)
}
func (c Resource) Set(ctx *gin.Context) {
	var params struct {
		Resource    registrycommon.ResourcePermissionResource `json:"resource" binding:"required"`
		UserID      int32                                     `json:"user_id" binding:"min=0"`
		VisibleType int32                                     `json:"visible_type" binding:"required,oneof=1 2 3 4"`
	}
	if !c.Validate(ctx, &params) {
		return
	}
	if err := (registrycommon.Resource{}).Set(ctx.Request.Context(), params.Resource, params.UserID, params.VisibleType); err != nil {
		c.respondError(ctx, err)
		return
	}
	c.JsonSuccessResponse(ctx)
}
func (c Resource) Delete(ctx *gin.Context) {
	var params struct {
		Resource registrycommon.ResourcePermissionResource `json:"resource" binding:"required"`
	}
	if !c.Validate(ctx, &params) {
		return
	}
	if err := (registrycommon.Resource{}).Delete(ctx.Request.Context(), params.Resource); err != nil {
		c.respondError(ctx, err)
		return
	}
	c.JsonSuccessResponse(ctx)
}
func (c Resource) Check(ctx *gin.Context) {
	type ParamsValidate struct {
		Token    string                                    `json:"token"`
		UserID   int32                                     `json:"user_id" binding:"min=0"`
		Resource registrycommon.ResourcePermissionResource `json:"resource" binding:"required"`
		Action   string                                    `json:"action" binding:"required,max=64,excludesall= \t\x00\r\n"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	if params.Token != "" {
		err := (registrycommon.Auth{}).VerifyToken(params.Token, registry.PermissionScope{Type: params.Resource.Type, Name: params.Resource.Value, Actions: []string{params.Action}})
		c.JsonResponseWithoutError(ctx, registrycommon.ResourcePermissionDecision{Allowed: err == nil})
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
	decision, err := (registrycommon.Resource{}).Check(ctx.Request.Context(), user, params.Resource, []string{params.Action})
	if err != nil {
		c.JsonResponseWithError(ctx, err, 503)
		return
	}
	c.JsonResponseWithoutError(ctx, decision)
}
func (c Resource) respondError(ctx *gin.Context, err error) {
	status := 503
	if errors.Is(err, gorm.ErrRecordNotFound) {
		status = 404
	}
	c.JsonResponseWithError(ctx, err, status)
}
