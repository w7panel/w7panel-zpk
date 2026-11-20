package main

import (
	"bytes"
	_ "embed"

	"github.com/spf13/viper"
	"github.com/w7panel/w7panel-zpk/modules/document/app/document"
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

	provider := new(document.Provider)
	defer provider.Close()
	provider.Register(httpServer)
	app.RunConsole()
}
