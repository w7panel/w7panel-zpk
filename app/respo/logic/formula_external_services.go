package logic

import (
	"net/url"
	"strings"
)

type externalService struct {
	Key      string `json:"key"`
	Title    string `json:"title"`
	URL      string `json:"url"`
	OpenMode string `json:"openMode,omitempty"`
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
	marketURL.Fragment = "/orders?tab=orders&order_sn=" + url.QueryEscape(orderSN)

	return []externalService{{
		Key:      "billing",
		Title:    "授权与续费",
		URL:      marketURL.String(),
		OpenMode: "iframe",
	}}
}
