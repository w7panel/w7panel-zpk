package middleware

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/common/service/w7"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/middleware"
)

type CloudAccessToken struct {
	middleware.Abstract
}

func (m CloudAccessToken) Process(ctx *gin.Context) {
	cloudAccessToken := ctx.Request.Header.Get("X-Cloud-AccessToken")
	if cloudAccessToken == "" {
		ctx.Next()
		return
	}

	userInfo, errResult := w7.OpenCloudSdk.OauthService.GetUserInfo(cloudAccessToken)
	if errResult.IsError() {
		m.JsonResponseWithServerError(ctx, errResult.ToError())
		ctx.Abort()
		return
	}
	if userInfo == nil {
		m.JsonResponseWithServerError(ctx, errors.New("oauth user info is empty"))
		ctx.Abort()
		return
	}

	ctx.Set("console_uid", int32(userInfo.UserId))

	ctx.Next()
}
