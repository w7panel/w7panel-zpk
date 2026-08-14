package w7

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/viper"
	opencloud "github.com/w7corp/sdk-open-cloud-go"
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
var OpenCloudSdk *opencloud.Client

type appCredential struct {
	Appid  string
	Secret string
}

func InitW7Sdk(config *viper.Viper) error {
	if config.GetString("w7.user_agent") != "" {
		base.DefaultUserAgent = config.GetString("w7.user_agent")
	}

	credential, err := resolveAppCredential(config)
	if err != nil {
		return err
	}

	DevCenterGoodsSdk = &devcenter.GoodsService{
		Base: base.Base{
			BaseUrl: "http://dev.w7.cc",
			Appid:   credential.Appid,
			Secret:  credential.Secret,
		},
	}
	DevCenterNotAppSdk = &devcenter.NotAppService{
		Base: base.Base{
			BaseUrl: "http://dev.w7.cc",
			Appid:   credential.Appid,
			Secret:  credential.Secret,
		},
	}
	IpGoodsSdk = &ip.GoodsService{
		Base: base.Base{
			BaseUrl: "http://ip.w7.cc",
			Appid:   credential.Appid,
			Secret:  credential.Secret,
		},
	}

	attachBaseUrl := "http://ds.api.w7.cc"
	if base.DefaultUserAgent == "we7test-develop" {
		attachBaseUrl = "http://api.w7.cc"
	}
	W7CloudAttach = &cloudapi.Attach{
		HttpClient: w7.NewClient(credential.Appid, credential.Secret, w7.Option{
			ApiUrl: "http://api.w7.cc",
		}).GetHttpClient(),
		BaseUrl: attachBaseUrl,
	}

	ZpkMarketSdk = &zpk_market.ZpkMarketService{
		Base: base.Base{
			BaseUrl: config.GetString("setting.depot_market.base_url"),
			Appid:   credential.Appid,
			Secret:  credential.Secret,
		},
	}

	OpenCloudSdk = opencloud.NewClient(credential.Appid, credential.Secret, w7.Option{
		ApiUrl: "http://api.w7.cc",
	})

	return nil
}

func resolveAppCredential(config *viper.Viper) (appCredential, error) {
	credential, err := fetchAppCredential("http://api.w7.cc/api/app/info")
	slog.Info("resolveAppCredential", "credential", credential, "err", err)
	if err != nil && strings.Contains(err.Error(), "Route not found") {
		return appCredential{
			Appid:  config.GetString("zpk.appid"),
			Secret: config.GetString("zpk.secret"),
		}, nil
	}
	if err != nil {
		return appCredential{}, fmt.Errorf("fetch w7 app info: %w", err)
	}
	if credential.Appid == "" || credential.Secret == "" {
		return appCredential{}, fmt.Errorf("fetch w7 app info: empty appid or appsecret")
	}

	config.Set("zpk.appid", credential.Appid)
	config.Set("zpk.secret", credential.Secret)
	return credential, nil
}

func fetchAppCredential(appInfoURL string) (appCredential, error) {
	req, err := http.NewRequest(http.MethodGet, appInfoURL, nil)
	if err != nil {
		return appCredential{}, err
	}
	if base.DefaultUserAgent != "" {
		req.Header.Set("User-Agent", base.DefaultUserAgent)
	}

	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return appCredential{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return appCredential{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return appCredential{}, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var wrapped struct {
		Code  int    `json:"code"`
		Error string `json:"error"`
		Data  struct {
			Appid     string `json:"appid"`
			Appsecret string `json:"appsecret"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return appCredential{}, err
	}
	if wrapped.Code != 0 && wrapped.Code != http.StatusOK {
		if wrapped.Error != "" {
			return appCredential{}, fmt.Errorf("unexpected code %d: %s", wrapped.Code, wrapped.Error)
		}
		return appCredential{}, fmt.Errorf("unexpected code %d", wrapped.Code)
	}
	return appCredential{
		Appid:  strings.TrimSpace(wrapped.Data.Appid),
		Secret: strings.TrimSpace(wrapped.Data.Appsecret),
	}, nil
}
