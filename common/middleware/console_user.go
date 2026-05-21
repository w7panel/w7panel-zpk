package middleware

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/middleware"
)

type ConsoleUser struct {
	middleware.Abstract
	CanSkip bool
}

func (m ConsoleUser) Process(ctx *gin.Context) {
	uid := logic.User{}.GetConsoleUid(ctx)
	if uid <= 0 && m.CanSkip {
		ctx.Next()
		return
	}
	if uid <= 0 {
		m.JsonResponseWithError(ctx, errors.New("请先绑定控制台用户"), 401)
		ctx.Abort()
		return
	}

	ctx.Next()
}
