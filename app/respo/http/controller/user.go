package controller

import (
	"github.com/gin-gonic/gin"
	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
)

type User struct {
	Abstract
}

func (c User) Info(ctx *gin.Context) {
	consoleUid := logic2.User{}.GetConsoleUid(ctx)

	c.JsonResponseWithoutError(ctx, map[string]interface{}{
		"console_uid": consoleUid,
	})
}
