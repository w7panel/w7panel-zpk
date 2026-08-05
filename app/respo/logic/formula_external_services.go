package logic

import (
	"net/url"
	"strings"

	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
)

type externalService struct {
	Key      string `json:"key"`
	Title    string `json:"title"`
	URL      string `json:"url"`
	OpenMode string `json:"openMode,omitempty"`
}

func BuildArtifactMarketBindings(baseURL string, goodsID int32, orderSN string) []logic2.Bindings {
	bindings := make([]logic2.Bindings, 0, 1)
	services := BuildArtifactMarketExternalServices(baseURL, goodsID, orderSN)
	if len(services) == 0 {
		return bindings
	}
	marketURL, _ := url.Parse(services[0].URL)
	marketURL.Fragment = ""
	menus := make([]logic2.Menu, 0, len(services))
	for index, service := range services {
		menus = append(menus, logic2.Menu{
			Title:        service.Title,
			Do:           service.URL,
			DisplayOrder: len(services) - index,
			Location:     "back",
		})
	}
	return append(bindings, logic2.Bindings{
		Name:     "zpk-market",
		Title:    "服务中心",
		Support:  "thirdparty_cd",
		Menu:     menus,
		LoadMode: "iframe",
		BackendConfig: logic2.BackendConfig{
			Type:       "external",
			BackendUrl: marketURL.String(),
			RequestProxy: logic2.RequestProxy{
				Headers: map[string]string{},
				Query:   map[string]string{},
			},
			FrontendProps: map[string]string{},
		},
	})
}

func BuildArtifactMarketExternalServices(baseURL string, goodsID int32, orderSN string) []externalService {
	baseURL = strings.TrimSpace(baseURL)
	orderSN = strings.TrimSpace(orderSN)
	if baseURL == "" || goodsID <= 0 || orderSN == "" {
		return nil
	}

	marketURL, err := url.Parse(baseURL)
	if err != nil || marketURL.Host == "" || (marketURL.Scheme != "http" && marketURL.Scheme != "https") {
		return nil
	}
	if marketURL.Path == "" {
		marketURL.Path = "/"
	}
	marketURL.Fragment = "/user-orders?tab=orders&order_sn=" + url.QueryEscape(orderSN)

	return []externalService{{
		Key:      "billing",
		Title:    "授权与续费",
		URL:      marketURL.String(),
		OpenMode: "iframe",
	}}
}
