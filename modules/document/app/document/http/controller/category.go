package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/modules/document/app/document/logic"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/permission"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/controller"
)

type Category struct {
	controller.Abstract
	Logic *logic.Category
}

func (c Category) List(ctx *gin.Context) {
	result, e := c.Logic.Categories(ctx.Request.Context(), actor(ctx))
	c.respond(ctx, result, e)
}

func (c Category) Create(ctx *gin.Context) {
	type ParamsValidate struct {
		Identifie   string `json:"identifie" binding:"required"`
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	e := c.Logic.CreateCategory(ctx.Request.Context(), actor(ctx), params.Identifie, params.Name, params.Description)
	c.respond(ctx, nil, e)
}

func (c Category) Edit(ctx *gin.Context) {
	type ParamsValidate struct {
		Identifie   string `json:"identifie" binding:"required"`
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Revision    int32  `json:"revision" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	e := c.Logic.EditCategory(ctx.Request.Context(), actor(ctx), params.Identifie, params.Name, params.Description, params.Revision)
	c.respond(ctx, nil, e)
}

func (c Category) Delete(ctx *gin.Context) {
	type ParamsValidate struct {
		Identifie string `json:"identifie" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	c.respond(ctx, nil, c.Logic.DeleteCategory(ctx.Request.Context(), actor(ctx), params.Identifie))
}

func (c Category) respond(ctx *gin.Context, data interface{}, err error) {
	if err == nil {
		c.JsonResponseWithoutError(ctx, data)
		return
	}
	status := http.StatusBadRequest
	if errors.Is(err, permission.ErrDenied) {
		status = http.StatusForbidden
	}
	c.JsonResponseWithError(ctx, err, status)
}
