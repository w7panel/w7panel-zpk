package middleware

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/modules/document/app/document/logic"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/zpk"
)

type Auth struct{}

func (m Auth) Process(c *gin.Context) {
	token := c.GetHeader(logic.TokenHeader)
	accessToken := c.GetHeader("X-W7Panel-Access-Token")
	if token == "" && accessToken == "" {
		c.AbortWithStatusJSON(401, gin.H{"code": 401, "error": "token 错误"})
		return
	}
	var principal zpk.User
	var err error
	if token != "" {
		principal, err = (logic.Session{}).GetUserInfo(c)
	} else {
		principal, err = service.ZPKSdk.CurrentUser(c.Request.Context(), accessToken)
	}
	if err != nil {
		status := 503
		if errors.Is(err, zpk.ErrUnauthenticated) {
			status = 401
		}
		message := "身份验证服务不可用"
		if status == 401 {
			message = "token 错误"
		}
		c.AbortWithStatusJSON(status, gin.H{"code": status, "error": message})
		return
	}
	c.Set("user", principal)
	c.Next()
}

type OptionalAuth struct{}

func (OptionalAuth) Process(c *gin.Context) {
	if c.GetHeader(logic.TokenHeader) == "" && c.GetHeader("X-W7Panel-Access-Token") == "" {
		c.Set("user", zpk.User{})
		c.Next()
		return
	}
	Auth{}.Process(c)
}
