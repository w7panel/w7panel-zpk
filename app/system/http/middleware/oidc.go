package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	systemlogic "github.com/w7panel/w7panel-zpk/app/system/logic"
	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/middleware"
)

// OIDC resolves the current panel identity into the existing ZPK user system.
type OIDC struct{ middleware.Abstract }

func (m OIDC) Process(ctx *gin.Context) {
	accessToken := ctx.GetHeader("X-W7Panel-Access-Token")
	if accessToken == "" {
		m.JsonResponseWithError(ctx, errors.New("token 错误"), http.StatusUnauthorized)
		ctx.Abort()
		return
	}
	info, err := (systemlogic.OIDC{}).UserInfo(ctx.Request.Context(), accessToken)
	if err != nil {
		status := http.StatusServiceUnavailable
		if errors.Is(err, systemlogic.ErrOIDCUnauthenticated) {
			status = http.StatusUnauthorized
		}
		message := errors.New("面板身份验证失败")
		if status == http.StatusUnauthorized {
			message = errors.New("token 错误")
		}
		m.JsonResponseWithError(ctx, message, status)
		ctx.Abort()
		return
	}
	if info.Subject == "" || info.Username == "" {
		m.JsonResponseWithError(ctx, errors.New("用户信息异常"), http.StatusUnauthorized)
		ctx.Abort()
		return
	}
	user, err := (commonlogic.User{}).GetOrCreatePanelUser(info.Subject, info.Username, info.Role)
	if err != nil || user == nil || user.ID <= 0 {
		m.JsonResponseWithError(ctx, errors.New("获取用户信息失败"), http.StatusServiceUnavailable)
		ctx.Abort()
		return
	}
	ctx.Set("user", user)
	ctx.Next()
}
