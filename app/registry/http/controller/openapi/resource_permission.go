package openapi

import (
	"github.com/gin-gonic/gin"

	registrycommon "github.com/w7panel/w7panel-zpk/app/registry/logic/common"
	"github.com/w7panel/w7panel-zpk/common/accessor"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/controller"
)

type ResourcePermission struct{ controller.Abstract }

func (c ResourcePermission) Add(ctx *gin.Context) {
	params := registrycommon.ResourcePermissionGrant{}
	if !c.Validate(ctx, &params) {
		return
	}
	permission := registrycommon.ResourcePermission{}
	current, err := permission.GetUserPermission(ctx.Request.Context(), params.UserID, params.Resource)
	if err == nil && current == nil {
		err = permission.AddUserPermission(params.UserID, params.Resource.Value, registrycommon.PermissionResourceType(params.Resource.Type), accessor.PermissionActionOption(params.Actions))
	}
	if err != nil {
		c.JsonResponseWithError(ctx, err, 503)
		return
	}
	c.JsonSuccessResponse(ctx)
}
func (c ResourcePermission) Delete(ctx *gin.Context) {
	type ParamsValidate struct {
		UserID   int32                                     `json:"user_id" binding:"min=0"`
		Resource registrycommon.ResourcePermissionResource `json:"resource" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	if err := (registrycommon.ResourcePermission{}).DelUserPermission(params.UserID, params.Resource.Value, registrycommon.PermissionResourceType(params.Resource.Type)); err != nil {
		c.JsonResponseWithError(ctx, err, 503)
		return
	}
	c.JsonSuccessResponse(ctx)
}
func (c ResourcePermission) Clear(ctx *gin.Context) {
	type ParamsValidate struct {
		UserID int32 `json:"user_id" binding:"required,min=1"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	if err := (registrycommon.ResourcePermission{}).ClearUserPermission(params.UserID); err != nil {
		c.JsonResponseWithError(ctx, err, 503)
		return
	}
	c.JsonSuccessResponse(ctx)
}
