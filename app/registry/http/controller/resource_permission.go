package controller

import (
	"errors"

	"github.com/gin-gonic/gin"
	registrylogic "github.com/w7panel/w7panel-zpk/app/registry/logic"
	userlogic "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/controller"
)

type ResourcePermission struct{ controller.Abstract }

func (c ResourcePermission) Identity(ctx *gin.Context) {
	user := userlogic.User{}.GetUser(ctx)
	c.JsonResponseWithoutError(ctx, gin.H{"id": user.ID, "admin": (userlogic.User{}).IsAdminUser(user), "username": user.Username, "role": user.Role})
}

func (c ResourcePermission) Check(ctx *gin.Context) {
	type ParamsValidate struct {
		Resources []registrylogic.ResourcePermissionResource `json:"resources" binding:"required"`
		Action    string                                     `json:"action" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	decision, err := (registrylogic.ResourcePermission{}).Check(ctx.Request.Context(), userlogic.User{}.GetUser(ctx), params.Resources, params.Action)
	if err != nil {
		c.JsonResponseWithError(ctx, err, 503)
		return
	}
	c.JsonResponseWithoutError(ctx, decision)
}

func (c ResourcePermission) Evaluate(ctx *gin.Context) {
	type ParamsValidate struct {
		UserID    int32                                      `json:"user_id" binding:"required,min=1"`
		Resources []registrylogic.ResourcePermissionResource `json:"resources" binding:"required"`
		Action    string                                     `json:"action" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	user, err := (userlogic.User{}).GetById(int(params.UserID))
	if err != nil || user == nil {
		c.JsonResponseWithError(ctx, errors.New("用户不存在"), 403)
		return
	}
	decision, err := (registrylogic.ResourcePermission{}).Check(ctx.Request.Context(), user, params.Resources, params.Action)
	if err != nil {
		c.JsonResponseWithError(ctx, err, 503)
		return
	}
	c.JsonResponseWithoutError(ctx, decision)
}

func (c ResourcePermission) Replace(ctx *gin.Context) {
	type ParamsValidate struct {
		Grants []registrylogic.ResourcePermissionGrant `json:"grants" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	if err := (registrylogic.ResourcePermission{}).Replace(ctx.Request.Context(), params.Grants); err != nil {
		c.JsonResponseWithError(ctx, err, 503)
		return
	}
	c.JsonSuccessResponse(ctx)
}

func (c ResourcePermission) Initialize(ctx *gin.Context) {
	type ParamsValidate struct {
		Grants []registrylogic.ResourcePermissionGrant `json:"grants" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	if err := (registrylogic.ResourcePermission{}).Initialize(ctx.Request.Context(), params.Grants); err != nil {
		c.JsonResponseWithError(ctx, err, 503)
		return
	}
	c.JsonSuccessResponse(ctx)
}

func (c ResourcePermission) CheckBatch(ctx *gin.Context) {
	type ParamsValidate struct {
		Requests []registrylogic.ResourcePermissionRequest `json:"requests" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	decisions, err := (registrylogic.ResourcePermission{}).CheckBatch(ctx.Request.Context(), (userlogic.User{}).GetUser(ctx), params.Requests)
	if err != nil {
		c.JsonResponseWithError(ctx, err, 503)
		return
	}
	c.JsonResponseWithoutError(ctx, decisions)
}

func (c ResourcePermission) EvaluateBatch(ctx *gin.Context) {
	type ParamsValidate struct {
		UserID   int32                                     `json:"user_id" binding:"required,min=1"`
		Requests []registrylogic.ResourcePermissionRequest `json:"requests" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	user, err := (userlogic.User{}).GetById(int(params.UserID))
	if err != nil || user == nil {
		c.JsonResponseWithError(ctx, errors.New("用户不存在"), 403)
		return
	}
	decisions, err := (registrylogic.ResourcePermission{}).CheckBatch(ctx.Request.Context(), user, params.Requests)
	if err != nil {
		c.JsonResponseWithError(ctx, err, 503)
		return
	}
	c.JsonResponseWithoutError(ctx, decisions)
}
