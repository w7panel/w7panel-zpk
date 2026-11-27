package openapi

import (
	"errors"

	"github.com/gin-gonic/gin"

	registrycommon "github.com/w7panel/w7panel-zpk/app/registry/logic/common"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/controller"
	"gorm.io/gorm"
)

type NamespaceService struct{ controller.Abstract }

func (c NamespaceService) Get(ctx *gin.Context) {
	type ParamsValidate struct {
		Name string `json:"name" binding:"required,max=96"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	query := dao.Q.RegistryNamespace
	namespace, err := query.WithContext(ctx.Request.Context()).Where(query.Name.Eq(params.Name)).First()
	if err != nil {
		c.respondError(ctx, err)
		return
	}
	c.JsonResponseWithoutError(ctx, namespace)
}

func (c NamespaceService) Create(ctx *gin.Context) {
	type ParamsValidate struct {
		Name        string `json:"name" binding:"required,max=96"`
		VisibleType int32  `json:"visible_type" binding:"required,oneof=1 2 4"`
		Visible     *bool  `json:"visible"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	if err := (registrycommon.Namespace{}).Create(ctx.Request.Context(), params.Name, params.VisibleType, params.Visible); err != nil {
		c.respondError(ctx, err)
		return
	}
	c.JsonSuccessResponse(ctx)
}

func (c NamespaceService) Rename(ctx *gin.Context) {
	type ParamsValidate struct {
		ID   int32  `json:"id" binding:"required,min=1"`
		Name string `json:"name" binding:"required,max=96"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	if err := (registrycommon.Namespace{}).Rename(ctx.Request.Context(), params.ID, params.Name); err != nil {
		c.respondError(ctx, err)
		return
	}
	c.JsonSuccessResponse(ctx)
}

func (c NamespaceService) Delete(ctx *gin.Context) {
	type ParamsValidate struct {
		ID int32 `json:"id" binding:"required,min=1"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	if err := (registrycommon.Namespace{}).Delete(ctx.Request.Context(), params.ID); err != nil {
		c.respondError(ctx, err)
		return
	}
	c.JsonSuccessResponse(ctx)
}

func (c NamespaceService) respondError(ctx *gin.Context, err error) {
	status := 503
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		status = 404
	case errors.Is(err, registrycommon.ErrNamespaceConflict):
		status = 409
	case errors.Is(err, registrycommon.ErrNamespaceProtected):
		status = 400
	}
	c.JsonResponseWithError(ctx, err, status)
}
