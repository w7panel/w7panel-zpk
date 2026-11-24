package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/modules/document/app/document/logic"
)

type Category struct {
	Abstract
}

func (c Category) List(ctx *gin.Context) {
	result, e := logic.Category{}.Categories(ctx.Request.Context())
	c.Respond(ctx, result, e)
}

func (c Category) Create(ctx *gin.Context) {
	type ParamsValidate struct {
		Identifie   string `json:"identifie" binding:"required,document_identifie"`
		Name        string `json:"name" binding:"required,notblank,maxbytes=255"`
		Description string `json:"description" binding:"maxbytes=4096"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	e := logic.Category{}.CreateCategory(ctx.Request.Context(), currentUser(ctx).ID, params.Identifie, params.Name, params.Description)
	c.Respond(ctx, nil, e)
}

func (c Category) Edit(ctx *gin.Context) {
	type ParamsValidate struct {
		Identifie    string `json:"identifie" binding:"required,document_identifie"`
		NewIdentifie string `json:"new_identifie" binding:"required,document_identifie"`
		Name         string `json:"name" binding:"required,notblank,maxbytes=255"`
		Description  string `json:"description" binding:"maxbytes=4096"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	e := logic.Category{}.EditCategory(ctx.Request.Context(), params.Identifie, params.NewIdentifie, params.Name, params.Description)
	c.Respond(ctx, nil, e)
}

func (c Category) Delete(ctx *gin.Context) {
	type ParamsValidate struct {
		Identifie string `json:"identifie" binding:"required,document_identifie"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	c.Respond(ctx, nil, logic.Category{}.DeleteCategory(ctx.Request.Context(), params.Identifie))
}
