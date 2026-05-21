package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
)

type DenyDomainReq struct{}

func (m DenyDomainReq) Process(ctx *gin.Context) {
	systemDomain := facade.GetConfig().GetString("setting.depot.external_domain")
	if systemDomain == "" {
		ctx.Next()
		return
	}

	if ctx.Request.Host == systemDomain {
		ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"code":  http.StatusForbidden,
			"error": "禁止访问当前接口",
		})
		return
	}

	ctx.Next()
}
