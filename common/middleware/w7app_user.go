package middleware

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/middleware"
)

type W7AppUser struct {
	middleware.Abstract
}

func (m W7AppUser) Process(ctx *gin.Context) {
	appid := ctx.GetString("appid")
	if appid == "" {
		m.JsonResponseWithServerError(ctx, errors.New("appid is empty"))
		ctx.Abort()
		return
	}

	user, err := getOrCreateUser("0", appid, founderRole, true)
	if err != nil {
		m.JsonResponseWithServerError(ctx, err)
		ctx.Abort()
		return
	}
	if user == nil {
		m.JsonResponseWithServerError(ctx, errors.New("user is nil"))
		ctx.Abort()
		return
	}

	ctx.Set("user", user)
	ctx.Next()
}
