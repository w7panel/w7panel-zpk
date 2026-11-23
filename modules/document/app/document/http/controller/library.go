package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/modules/document/app/document/logic"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/permission"
)

type Library struct{ Abstract }

func (c Library) List(ctx *gin.Context) {
	type ParamsValidate struct {
		Category string `json:"category"`
		Page     int    `json:"page"`
		Limit    int    `json:"limit"`
	}
	params := ParamsValidate{}
	if !c.Bind(ctx, &params) {
		return
	}
	result, e := logic.Document{}.List(ctx.Request.Context(), actor(ctx), params.Category, params.Page, params.Limit)
	c.Respond(ctx, result, e)
}

func (c Library) Create(ctx *gin.Context) {
	type ParamsValidate struct {
		Ref         string `json:"ref" binding:"required"`
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Public      int32  `json:"public" binding:"oneof=0 1"`
	}
	params := ParamsValidate{}
	if !c.Bind(ctx, &params) {
		return
	}
	result, e := logic.Document{}.Create(ctx.Request.Context(), actor(ctx), params.Ref, params.Name, params.Description, "", false, params.Public)
	c.Respond(ctx, result, e)
}

func (c Library) Initialize(ctx *gin.Context) {
	type ParamsValidate struct {
		Ref      string `json:"ref" binding:"required"`
		Name     string `json:"name" binding:"required"`
		OwnerID  int32  `json:"owner_id"`
		Homepage string `json:"homepage"`
	}
	params := ParamsValidate{}
	if !c.Bind(ctx, &params) {
		return
	}
	currentActor := actor(ctx)
	if params.OwnerID > 0 && params.OwnerID != currentActor.Principal.ID {
		if !currentActor.Principal.Admin {
			c.Respond(ctx, nil, permission.ErrDenied)
			return
		}
		currentActor.Principal.ID = params.OwnerID
	}
	result, e := logic.Document{}.Create(ctx.Request.Context(), currentActor, params.Ref, params.Name, "", params.Homepage, true, 0)
	c.Respond(ctx, result, e)
}

func (c Library) Edit(ctx *gin.Context) {
	type ParamsValidate struct {
		Ref         string `json:"ref" binding:"required"`
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Public      *int32 `json:"public"`
		Revision    int32  `json:"revision" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Bind(ctx, &params) {
		return
	}
	c.Respond(ctx, nil, logic.Document{}.EditLibrary(ctx.Request.Context(), actor(ctx), params.Ref, params.Name, params.Description, params.Revision, params.Public))
}

func (c Library) Delete(ctx *gin.Context) {
	type ParamsValidate struct {
		Ref string `json:"ref" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Bind(ctx, &params) {
		return
	}
	c.Respond(ctx, nil, logic.Document{}.DeleteLibrary(ctx.Request.Context(), actor(ctx), params.Ref))
}
