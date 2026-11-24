package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/permission"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/zpk"
)

type Admin struct{}

func (m Admin) Process(ctx *gin.Context) {
	value, _ := ctx.Get("user")
	user, ok := value.(zpk.User)
	if !ok || !user.Admin {
		ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": http.StatusForbidden, "error": permission.ErrDenied.Error()})
		return
	}
	ctx.Next()
}
