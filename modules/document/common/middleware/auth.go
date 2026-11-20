package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/permission"
	"github.com/w7panel/w7panel-zpk/modules/document/common/types"
)

type Auth struct {
	Permissions permission.Service
	Optional    bool
}

func (m Auth) Process(c *gin.Context) {
	token := c.GetHeader("X-Document-Token")
	if token == "" {
		token = c.GetHeader("X-Zpk-Token")
	}
	if token == "" && m.Optional {
		c.Set("document.actor", types.Actor{})
		c.Next()
		return
	}
	if token == "" {
		c.AbortWithStatusJSON(401, gin.H{"code": 401, "error": "token 错误"})
		return
	}
	p, e := m.Permissions.Identity(c.Request.Context(), token)
	if e != nil {
		c.AbortWithStatusJSON(401, gin.H{"code": 401, "error": "身份验证失败"})
		return
	}
	c.Set("document.actor", types.Actor{Principal: p, Token: token})
	c.Next()
}
