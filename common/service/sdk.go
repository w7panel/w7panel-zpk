package service

import (
	"github.com/spf13/viper"
	"github.com/w7panel/w7panel-zpk/common/service/document"
)

var DocumentSdk *document.Client

func InitDocumentSdk(config *viper.Viper) {
	DocumentSdk = document.New(config.GetString("setting.document.base_url"))
}
