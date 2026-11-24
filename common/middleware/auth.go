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

	loginUser, err := logic.Login{}.GetUserInfo(ctx)
	if err != nil {
		if errors.Is(err, logic.ErrInvalidToken) {
			m.JsonResponseWithError(ctx, err, 401)
		} else {
			m.JsonResponseWithServerError(ctx, err)
		}
		ctx.Abort()
		return
	}

	user, err := logic.User{}.GetById(int(loginUser.UserID))
	if err != nil || user == nil {
		m.JsonResponseWithError(ctx, errors.New("用户不存在"), 401)
		ctx.Abort()
		return
	}

	nextToken, err := (logic.Login{}).RefreshExpire(ctx)
	if err != nil {
		m.JsonResponseWithServerError(ctx, err)
		ctx.Abort()
		return
	}
	if nextToken != "" {
		ctx.Header(logic.ZpkTokenHeader, nextToken)
	}

	ctx.Set("console_uid", loginUser.ConsoleUid)
	ctx.Set("user", user)
	ctx.Next()
}
