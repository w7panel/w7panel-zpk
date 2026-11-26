package middleware

import (
	"bytes"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/zpk"
)

// AppUser reads the user identity after the App middleware verifies the signed request.
type AppUser struct{}

func (AppUser) Process(ctx *gin.Context) {
	type ParamsValidate struct {
		UserID int32 `json:"user_id" binding:"required,min=1"`
	}
	params := ParamsValidate{}
	if err := ctx.ShouldBindBodyWith(&params, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "error": "用户 ID 无效"})
		return
	}
	ctx.Request.Body = io.NopCloser(bytes.NewReader(ctx.MustGet(gin.BodyBytesKey).([]byte)))
	ctx.Set("user", zpk.User{ID: params.UserID})
	ctx.Next()
}
