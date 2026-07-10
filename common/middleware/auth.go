package middleware

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/middleware"
)

type Auth struct {
	middleware.Abstract
	CanSkip bool
}

func (m Auth) Process(ctx *gin.Context) {
	zpkToken := ctx.Request.Header.Get(logic.ZpkTokenHeader)
	if zpkToken == "" {
		if m.CanSkip {
			ctx.Next()
			return
		}

		m.JsonResponseWithError(ctx, errors.New("token 错误"), 401)
		ctx.Abort()
		return
	}

	sessionUser, err := logic.Session{}.GetUserInfo(ctx, zpkToken)
	if err != nil {
		m.JsonResponseWithServerError(ctx, err)
		ctx.Abort()
		return
	}
	if sessionUser == nil || sessionUser.UserID <= 0 {
		m.JsonResponseWithError(ctx, errors.New("token 错误"), 401)
		ctx.Abort()
		return
	}

	user, err := logic.User{}.GetById(int(sessionUser.UserID))
	if err != nil || user == nil {
		m.JsonResponseWithError(ctx, errors.New("用户不存在"), 401)
		ctx.Abort()
		return
	}

	if err := (logic.Session{}).RefreshExpire(ctx); err != nil {
		m.JsonResponseWithServerError(ctx, err)
		ctx.Abort()
		return
	}

	ctx.Set("console_uid", sessionUser.ConsoleUid)
	ctx.Set("user", user)
	ctx.Next()
}
