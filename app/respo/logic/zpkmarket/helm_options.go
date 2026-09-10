package zpkmarket

import (
	"net/url"
	"strings"

	"github.com/w7panel/w7panel-zpk/app/respo/logic/helm"
	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
)

const microAppChartName = "zpk-market"

func BuildHelmOptions(application commonlogic.Application, marketURL string, goodsID int32, orderSN string) []helm.DynamicHelmPackageOption {
	bindings := buildBindings(marketURL, goodsID, orderSN)
	if len(bindings) == 0 {
		return nil
	}

	for _, binding := range bindings {
		if strings.TrimSpace(binding.Title) != "" {
			application.Name = binding.Title
			break
		}
	}
	if strings.TrimSpace(application.Name) == "" {
		application.Name = "Cloud Service"
	}

	return []helm.DynamicHelmPackageOption{
		helm.WithMicroAppSubchart(microAppChartName, application, bindings),
	}
}

func buildBindings(marketURL string, goodsID int32, orderSN string) []commonlogic.Bindings {
	menus := buildMenus(goodsID, orderSN)
	if len(menus) == 0 {
		return nil
	}

	return []commonlogic.Bindings{{
		Name:     "other",
		Title:    "云服务",
		Support:  "thirdparty_cd",
		Menu:     menus,
		LoadMode: "iframe",
		BackendConfig: commonlogic.BackendConfig{
			Type:       "external",
			BackendUrl: strings.TrimRight(marketURL, "/") + "/",
			RequestProxy: commonlogic.RequestProxy{
				Headers: map[string]string{},
				Query:   map[string]string{},
			},
			FrontendProps: map[string]string{},
		},
	}}
}

func buildMenus(goodsID int32, orderSN string) []commonlogic.Menu {
	orderSN = strings.TrimSpace(orderSN)
	if goodsID <= 0 || orderSN == "" {
		return nil
	}

	return []commonlogic.Menu{{
		Title:        "授权与续费",
		Do:           "#/user-orders?tab=orders&order_sn=" + url.QueryEscape(orderSN),
		Location:     "left",
		DisplayOrder: 0,
	}}
}
