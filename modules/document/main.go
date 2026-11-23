package main

import (
	"bytes"
	_ "embed"

	"github.com/spf13/viper"
	"github.com/w7panel/w7panel-zpk/modules/document/app/document"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service"
	application "github.com/we7coreteam/w7-rangine-go/v2/src"
	"github.com/we7coreteam/w7-rangine-go/v2/src/core/helper"
	ranginehttp "github.com/we7coreteam/w7-rangine-go/v2/src/http"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/middleware"
)

//go:embed config.yaml
var ConfigFileContent []byte

func main() {
	app := application.NewApp(application.Option{
		Name: "document",
		DefaultConfigLoader: func(config *viper.Viper) {
			config.SetConfigType("yaml")
			if err := config.MergeConfig(bytes.NewReader(helper.ParseConfigContentEnv(ConfigFileContent))); err != nil {
				panic(err)
			}
		},
	})
	httpServer := new(ranginehttp.Provider).Register(app.GetConfig(), app.GetConsole(), app.GetServerManager()).Export()
	httpServer.Use(middleware.GetPanicHandlerMiddleware())

	if err := service.InitSDK(app.GetConfig()); err != nil {
		panic(err)
	}
	defer service.DocumentStorage.Close()

	new(document.Provider).Register(httpServer)
	app.RunConsole()
}
