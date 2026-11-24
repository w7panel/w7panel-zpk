package main

import (
	"bytes"
	_ "embed"
	"os"

	"github.com/spf13/viper"
	"github.com/w7panel/w7panel-zpk/cli/app/application"
	app "github.com/we7coreteam/w7-rangine-go/v2/src"
	"github.com/we7coreteam/w7-rangine-go/v2/src/console"
	"github.com/we7coreteam/w7-rangine-go/v2/src/core/helper"
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
		},
	})

	// 注册业务 provider，此模块中需要使用 http server 和 console
	new(application.Provider).Register(app.GetConsole())

	if err := app.GetConsole().(*console.Console).GetHandler().Execute(); err != nil {
		os.Exit(1)
	}
}
