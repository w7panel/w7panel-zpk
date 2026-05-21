package controller

import (
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/controller"
)

type Info struct {
	controller.Abstract
}

func (c Info) RegistryInfo(ctx *gin.Context) {
	intranetUrlInfo, err := url.Parse(facade.GetConfig().GetString("setting.registry.intranet_url"))
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	c.JsonResponseWithoutError(ctx, map[string]interface{}{
		"server": map[string]interface{}{
			"intranet_domain": intranetUrlInfo.Host,
			"external_domain": facade.GetConfig().GetString("setting.registry.external_domain"),
		},
	})
}
