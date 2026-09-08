package logic

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
)

const zpkMarketMicroAppChartName = "zpk-market"

func BuildZpkMarketHelmOptions(application logic2.Application, marketURL string, goodsID int32, orderSN string) []DynamicHelmPackageOption {
	marketBindings := buildZpkMarketBindings(marketURL, goodsID, orderSN)
	if len(marketBindings) == 0 {
		return nil
	}
	return []DynamicHelmPackageOption{
		withZpkMarketMicroApp(application, marketBindings),
	}
}

func buildZpkMarketBindings(marketURL string, goodsID int32, orderSN string) []logic2.Bindings {
	menus := buildZpkMarketMenus(goodsID, orderSN)
	if len(menus) == 0 {
		return nil
	}

	return []logic2.Bindings{{
		Name:     "other",
		Title:    "云服务",
		Support:  "thirdparty_cd",
		Menu:     menus,
		LoadMode: "iframe",
		BackendConfig: logic2.BackendConfig{
			Type:       "external",
			BackendUrl: strings.TrimRight(marketURL, "/") + "/",
			RequestProxy: logic2.RequestProxy{
				Headers: map[string]string{},
				Query:   map[string]string{},
			},
			FrontendProps: map[string]string{},
		},
	}}
}

func buildZpkMarketMenus(goodsID int32, orderSN string) []logic2.Menu {
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

func withZpkMarketMicroApp(application logic2.Application, marketBindings []logic2.Bindings) DynamicHelmPackageOption {
	templateConfig := newMicroAppTemplateConfig(application)
	marketBindingsCopy := append([]logic2.Bindings(nil), marketBindings...)
	cacheValue := struct {
		Kind           string                 `json:"kind"`
		TemplateConfig microAppTemplateConfig `json:"template_config"`
		Bindings       []logic2.Bindings      `json:"bindings"`
	}{
		Kind:           "zpk-market-microapp",
		TemplateConfig: templateConfig,
		Bindings:       marketBindingsCopy,
	}
	return func(options *dynamicHelmPackageOptions) error {
		return options.addTransform(cacheValue, func(chartDir string) error {
			return writeZpkMarketMicroAppChart(chartDir, application, marketBindingsCopy)
		})
	}
}

func writeZpkMarketMicroAppChart(chartDir string, application logic2.Application, marketBindings []logic2.Bindings) error {
	for _, binding := range marketBindings {
		if strings.TrimSpace(binding.Title) != "" {
			application.Name = binding.Title
			break
		}
	}
	if strings.TrimSpace(application.Name) == "" {
		application.Name = "Cloud Service"
	}

	zpkMarketChartDir := filepath.Join(chartDir, "charts", zpkMarketMicroAppChartName)
	if err := os.MkdirAll(filepath.Join(zpkMarketChartDir, "templates"), 0755); err != nil {
		return fmt.Errorf("创建 ZPK Market MicroApp Chart 目录失败: %w", err)
	}
	if err := writeYAMLFile(filepath.Join(zpkMarketChartDir, "Chart.yaml"), ChartYAML{
		APIVersion: "v2",
		Name:       zpkMarketMicroAppChartName,
		Version:    "0.1.0",
		Type:       "application",
		AppVersion: application.Version,
	}); err != nil {
		return err
	}

	manifest := logic2.Manifest{
		Application: application,
		Bindings:    marketBindings,
	}
	return (&HelmPack{IsSubFormula: true}).generateMicroAppTemplate(
		filepath.Join(zpkMarketChartDir, "templates"),
		manifest,
	)
}
