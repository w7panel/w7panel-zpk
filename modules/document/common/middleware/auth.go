package middleware

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/modules/document/app/document/logic"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/zpk"
	"github.com/w7panel/w7panel-zpk/modules/document/common/types"
)

type Auth struct {
	Optional bool
}

func (m Auth) Process(c *gin.Context) {
	token := c.GetHeader(logic.TokenHeader)
	accessToken := c.GetHeader("X-W7Panel-Access-Token")
	if token == "" && accessToken == "" && m.Optional {
		c.Set("document.actor", types.Actor{})
		c.Next()
		return
	}
	if token == "" && accessToken == "" {
		c.AbortWithStatusJSON(401, gin.H{"code": 401, "error": "token 错误"})
		return
	}
	var principal zpk.User
	var err error
	if token != "" {
		principal, err = (logic.Session{}).GetUserInfo(c, token)
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
	c.Set("document.actor", types.Actor{Principal: principal})
	c.Next()
}
