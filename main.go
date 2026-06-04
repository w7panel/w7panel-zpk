package main

import (
	"bytes"
	_ "embed"
	nethttp "net/http"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	registry2 "github.com/w7panel/w7panel-zpk/app/registry"
	"github.com/w7panel/w7panel-zpk/app/respo"
	"github.com/w7panel/w7panel-zpk/app/system"
	systemLogic "github.com/w7panel/w7panel-zpk/app/system/logic"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/service/w7"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	app "github.com/we7coreteam/w7-rangine-go/v2/src"
	"github.com/we7coreteam/w7-rangine-go/v2/src/core/err_handler"
	"github.com/we7coreteam/w7-rangine-go/v2/src/core/helper"
	ranginehttp "github.com/we7coreteam/w7-rangine-go/v2/src/http"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/middleware"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/response"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/session"
)

//go:embed config.yaml
var ConfigFileContent []byte

func main() {
	app := app.NewApp(app.Option{
		Name: "w7-zpk",
		DefaultConfigLoader: func(config *viper.Viper) {
			config.SetConfigType("yaml")
			err := config.MergeConfig(bytes.NewReader(helper.ParseConfigContentEnv(ConfigFileContent)))
			if err != nil {
				panic(err)
			}

			if err := systemLogic.ApplyDBSwitchConfig(config); err != nil {
				panic(err)
			}
		},
	})
	// 业务中需要使用 http server，这里需要先实例化111
	httpServer := new(ranginehttp.Provider).Register(app.GetConfig(), app.GetConsole(), app.GetServerManager()).Export()
	// 注册一些全局中间件，路由或是其它一些全局操作
	httpServer.Use(middleware.GetPanicHandlerMiddleware())
	httpServer.Use(middleware.GetSessionMiddleware(app.GetConfig(), session.GetGormStore, []byte("secret")))
	httpServer.Use(func(ctx *gin.Context) {
		if ctx.Request.Method == nethttp.MethodGet || ctx.Request.Method == nethttp.MethodHead || ctx.Request.Method == nethttp.MethodOptions {
			ctx.Next()
			return
		}
		if !systemLogic.ShouldBlockWrites(app.GetConfig()) {
			ctx.Next()
			return
		}
		ctx.JSON(503, map[string]interface{}{
			"code":  503,
			"error": "数据库迁移进行中，当前已禁止写操作",
		})
		ctx.Abort()
	})

	response.SetErrResponseHandler(func(ctx *gin.Context, env string, err error, statusCode int) {
		ctx.JSON(statusCode, map[string]interface{}{
			"code":  statusCode,
			"error": err.Error(),
		})
	})
	db, err := facade.GetDbFactory().Channel("default")
	if err_handler.Found(err) {
		panic(err)
	}
	dao.SetDefault(db)

	w7.InitW7Sdk(app.GetConfig())

	// 注册业务 provider，此模块中需要使用 http server 和 console
	new(respo.Provider).Register(httpServer, app.GetConsole())
	new(registry2.Provider).Register(httpServer)
	new(system.Provider).Register(httpServer)

	app.RunConsole()
}
