package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/modules/document/app/document/logic"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/permission"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/zpk"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/controller"
	"gorm.io/gorm"
)

type Abstract struct {
	controller.Abstract
}

func currentUser(ctx *gin.Context) zpk.User {
	return ctx.MustGet("user").(zpk.User)
}

func (c Abstract) Respond(ctx *gin.Context, data any, e error) {
	if e == nil {
		c.JsonResponseWithoutError(ctx, data)
		return
	}
	status := http.StatusServiceUnavailable
	var remoteError *zpk.ResponseError
	switch {
	case errors.Is(e, zpk.ErrUnauthenticated):
		status = http.StatusUnauthorized
	case errors.Is(e, permission.ErrDenied):
		status = http.StatusForbidden
	case errors.Is(e, gorm.ErrRecordNotFound):
		status = http.StatusNotFound
	case errors.Is(e, logic.ErrInvalid):
		status = http.StatusBadRequest
	case errors.Is(e, logic.ErrConflict):
		status = http.StatusConflict
	case errors.As(e, &remoteError) && (remoteError.StatusCode == 400 || remoteError.StatusCode == 404 || remoteError.StatusCode == 409):
		status = remoteError.StatusCode
	}
	c.JsonResponseWithError(ctx, e, status)
}
