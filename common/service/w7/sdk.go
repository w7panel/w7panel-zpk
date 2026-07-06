package w7

import (
	"github.com/spf13/viper"
	w7 "github.com/w7corp/sdk-open-cloud-go"
	"github.com/w7panel/w7panel-zpk/common/service/w7/base"
	"github.com/w7panel/w7panel-zpk/common/service/w7/cloudapi"
	"github.com/w7panel/w7panel-zpk/common/service/w7/devcenter"
	"github.com/w7panel/w7panel-zpk/common/service/w7/ip"
	zpk_market "github.com/w7panel/w7panel-zpk/common/service/w7/zpk-market"
)

var DevCenterNotAppSdk *devcenter.NotAppService
var DevCenterGoodsSdk *devcenter.GoodsService
var W7CloudAttach *cloudapi.Attach
var IpGoodsSdk *ip.GoodsService
var ZpkMarketSdk *zpk_market.ZpkMarketService

func InitW7Sdk(config *viper.Viper) {
	if config.GetString("w7.user_agent") != "" {
		base.DefaultUserAgent = config.GetString("w7.user_agent")
	}

	DevCenterGoodsSdk = &devcenter.GoodsService{
		Base: base.Base{
			BaseUrl: "http://dev.w7.cc",
			Appid:   config.GetString("zpk.appid"),
			Secret:  config.GetString("zpk.secret"),
		},
	}
	DevCenterNotAppSdk = &devcenter.NotAppService{
		Base: base.Base{
			BaseUrl: "http://dev.w7.cc",
			Appid:   config.GetString("zpk.appid"),
			Secret:  config.GetString("zpk.secret"),
		},
	}
	IpGoodsSdk = &ip.GoodsService{
		Base: base.Base{
			BaseUrl: "http://ip.w7.cc",
			Appid:   config.GetString("zpk.appid"),
			Secret:  config.GetString("zpk.secret"),
		},
	}

	attachBaseUrl := "http://ds.api.w7.cc"
	if base.DefaultUserAgent == "we7test-develop" {
		attachBaseUrl = "http://api.w7.cc"
	}
	W7CloudAttach = &cloudapi.Attach{
		HttpClient: w7.NewClient(config.GetString("zpk.appid"), config.GetString("zpk.secret"), w7.Option{
			ApiUrl: "http://api.w7.cc",
		}).GetHttpClient(),
		BaseUrl: attachBaseUrl,
	}

	ZpkMarketSdk = &zpk_market.ZpkMarketService{
		Base: base.Base{
			BaseUrl: config.GetString("setting.depot_market.base_url"),
			Appid:   config.GetString("zpk.appid"),
			Secret:  config.GetString("zpk.secret"),
		},
	}
}
