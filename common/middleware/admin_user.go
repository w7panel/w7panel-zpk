package middleware

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/middleware"
)

type AdminUser struct {
	middleware.Abstract
}

func (m AdminUser) Process(ctx *gin.Context) {
	user := logic.User{}.GetUser(ctx)
	if user == nil {
		m.JsonResponseWithServerError(ctx, errors.New("非法操作"))
		ctx.Abort()
		return
	}
	if !(logic.User{}.IsAdminUser(user)) {
		m.JsonResponseWithServerError(ctx, errors.New("非法操作"))
		ctx.Abort()
		return
	}

	ctx.Next()
}
