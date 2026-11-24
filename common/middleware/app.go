package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/common/service/w7"
	"github.com/w7panel/w7panel-zpk/common/service/w7/base"
)

// App only authenticates service calls; user authorization remains separate.
type App struct{}

func (App) Process(ctx *gin.Context) {
	if err := w7.AppSdk.Verify(ctx.Request); err != nil {
		status := http.StatusUnauthorized
		message := "应用签名无效"
		if errors.Is(err, base.ErrCredential) {
			status = http.StatusServiceUnavailable
			message = "应用签名服务未初始化"
		}
		ctx.AbortWithStatusJSON(status, gin.H{"code": status, "error": message})
		return
	}
	ctx.Next()
}
