package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/app"
)

type App struct{}

func (App) Process(ctx *gin.Context) {
	if err := service.AppSdk.Verify(ctx.Request); err != nil {
		status := http.StatusUnauthorized
		message := "应用签名无效"
		if errors.Is(err, app.ErrCredential) {
			status = http.StatusServiceUnavailable
			message = "应用签名服务未初始化"
		}
		ctx.AbortWithStatusJSON(status, gin.H{"code": status, "error": message})
		return
	}
	ctx.Next()
}
