package logic

import (
	"net/url"
	"strings"

	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
)

func BuildArtifactMarketBindings(baseURL string, goodsID int32, orderSN string) []logic2.Bindings {
	bindings := make([]logic2.Bindings, 0)
	services := BuildArtifactMarketExternalServices(goodsID, orderSN)
	if len(services) == 0 {
		return bindings
	}

	return append(bindings, logic2.Bindings{
		Name:     "other",
		Title:    "云服务",
		Support:  "thirdparty_cd",
		Menu:     services,
		LoadMode: "iframe",
		BackendConfig: logic2.BackendConfig{
			Type:       "external",
			BackendUrl: strings.TrimRight(baseURL, "/") + "/",
			RequestProxy: logic2.RequestProxy{
				Headers: map[string]string{},
				Query:   map[string]string{},
			},
			FrontendProps: map[string]string{},
		},
	})
}

func BuildArtifactMarketExternalServices(goodsID int32, orderSN string) []logic2.Menu {
	orderSN = strings.TrimSpace(orderSN)
	if goodsID <= 0 || orderSN == "" {
		return nil
	}

	return []logic2.Menu{{
		Title:        "授权与续费",
		Do:           "#/user-orders?tab=orders&order_sn=" + url.QueryEscape(orderSN),
		Location:     "left",
		DisplayOrder: 0,
	}}
}
