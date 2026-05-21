package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/controller"
)

type Static struct {
	controller.Abstract
}

func (self Static) File(ctx *gin.Context) {
	local := facade.GetConfig().GetString("setting.depot.storage.local.path") + "/Static"
	path := ctx.Param("path")
	ctx.File(local + "/" + path)
}
