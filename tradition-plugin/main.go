package main

import (
	"bytes"
	_ "embed"

	"github.com/spf13/viper"
	"github.com/w7panel/w7-tradition-plugin/app/application"
	app "github.com/we7coreteam/w7-rangine-go/v2/src"
	"github.com/we7coreteam/w7-rangine-go/v2/src/core/helper"
)

//go:embed config.yaml
var ConfigFileContent []byte

func main() {
	applicationApp := app.NewApp(app.Option{
		Name: "w7-tradition-plugin",
		DefaultConfigLoader: func(config *viper.Viper) {
			config.SetConfigType("yaml")
			if err := config.MergeConfig(bytes.NewReader(
				helper.ParseConfigContentEnv(ConfigFileContent),
			)); err != nil {
				panic(err)
			}
		},
	})

	new(application.Provider).Register(applicationApp.GetConsole())
	applicationApp.RunConsole()
}
