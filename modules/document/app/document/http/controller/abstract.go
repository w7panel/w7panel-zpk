package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/modules/document/app/document/logic"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/permission"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/controller"
	"gorm.io/gorm"
)

type Abstract struct {
	controller.Abstract
	Logic *logic.Document
}

func actor(c *gin.Context) logic.Actor { return c.MustGet("document.actor").(logic.Actor) }
func (c Abstract) Bind(ctx *gin.Context, params any) bool {
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, 6<<20)
	if e := ctx.ShouldBindJSON(params); e != nil {
		c.Respond(ctx, nil, logic.ErrInvalid)
		return false
	}
	return true
}

func (c Abstract) Respond(ctx *gin.Context, data any, e error) {
	if e == nil {
		c.JsonResponseWithoutError(ctx, data)
		return
	}
	status := 503
	switch {
	case errors.Is(e, permission.ErrDenied):
		status = 403
	case errors.Is(e, gorm.ErrRecordNotFound):
		status = 404
	case errors.Is(e, logic.ErrInvalid):
		status = 400
	case errors.Is(e, logic.ErrConflict):
		status = 409
	}
	c.JsonResponseWithError(ctx, e, status)
}
