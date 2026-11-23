package logic

import (
	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service"
)

type Login struct{}

func (l Login) FromPanel(ctx *gin.Context, accessToken string) (string, error) {
	principal, err := service.ZPKSdk.CurrentUser(ctx.Request.Context(), accessToken)
	if err != nil {
		return "", err
	}
	return (Session{}).SaveUserInfo(ctx, principal)
}
