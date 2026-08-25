package logic

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
)

type gatewayPluginRenderer interface {
	Validate(plugin commonlogic.GatewayPlugin) error
	RuntimeValues(plugin commonlogic.GatewayPlugin) map[string]interface{}
	TemplateName() string
	OutputName() string
}

var gatewayPluginRenderers = map[string]gatewayPluginRenderer{
	commonlogic.GatewayPluginDriverHigressWasmV1: higressWasmV1Renderer{},
}

func getGatewayPluginRenderer(plugin commonlogic.GatewayPlugin) (gatewayPluginRenderer, error) {
	plugin = plugin.Normalize()
	renderer, ok := gatewayPluginRenderers[plugin.Runtime.Driver]
	if !ok {
		return nil, fmt.Errorf("不支持的网关插件运行时驱动: %s", plugin.Runtime.Driver)
	}
	if err := renderer.Validate(plugin); err != nil {
		return nil, err
	}
	return renderer, nil
}

type higressWasmV1Renderer struct{}

func (higressWasmV1Renderer) Validate(plugin commonlogic.GatewayPlugin) error {
	url := plugin.Runtime.Config.URL
	if strings.TrimSpace(url) == "" {
		return errors.New("Higress Wasm 插件镜像地址不能为空")
	}
	phase := plugin.Runtime.Config.Phase
	if phase == "" {
		phase = "UNSPECIFIED_PHASE"
	}
	validPhases := map[string]struct{}{
		"UNSPECIFIED_PHASE": {},
		"AUTHN":             {},
		"AUTHZ":             {},
		"STATS":             {},
	}
	if _, ok := validPhases[phase]; !ok {
		return fmt.Errorf("不支持的 Higress Wasm 插件执行阶段: %s", phase)
	}
	priority := plugin.Runtime.Config.Priority
	if priority < 0 || priority > 1000 {
		return errors.New("Higress Wasm 插件优先级必须在 0 到 1000 之间")
	}
	return nil
}

func (higressWasmV1Renderer) RuntimeValues(plugin commonlogic.GatewayPlugin) map[string]interface{} {
	phase := plugin.Runtime.Config.Phase
	if phase == "" {
		phase = "UNSPECIFIED_PHASE"
	}
	return map[string]interface{}{
		"url":      plugin.Runtime.Config.URL,
		"phase":    phase,
		"priority": plugin.Runtime.Config.Priority,
	}
}

func (higressWasmV1Renderer) TemplateName() string {
	return "higress-gateway-plugin.yaml.tpl"
}

func (higressWasmV1Renderer) OutputName() string {
	return "higress-gateway-plugin.yaml"
}

func (hc *HelmPack) packGatewayPluginApp(rootDir, templatesDir string) error {
	renderer, err := getGatewayPluginRenderer(hc.Manifest.Platform.GatewayPlugin)
	if err != nil {
		return err
	}
	if err := hc.generateGatewayPluginValuesYaml(rootDir, renderer); err != nil {
		return err
	}
	return hc.generateGatewayPluginTemplate(templatesDir, renderer)
}

func (hc *HelmPack) generateGatewayPluginValuesYaml(rootDir string, renderer gatewayPluginRenderer) error {
	plugin := hc.Manifest.Platform.GatewayPlugin.Normalize()
	defaultConfig := plugin.DefaultConfig
	if defaultConfig == nil {
		defaultConfig = make(map[string]interface{})
	}
	appName := hc.Manifest.Application.Name
	if appName == "" {
		appName = hc.Manifest.Application.Identifie
	}
	values := map[string]interface{}{
		"app": map[string]interface{}{
			"title":    appName,
			"identify": hc.Manifest.Application.Identifie,
		},
		"gatewayPlugin": map[string]interface{}{
			"defaultEnabled": plugin.IsEnabledByDefault(),
			"supportGlobal":  plugin.IsSupportGlobal(),
			"supportRule":    plugin.Supports.Rule,
			"defaultConfig":  defaultConfig,
			"configSchema":   plugin.ConfigSchema,
			"hasFrontend":    strings.TrimSpace(hc.Manifest.Web.Url) != "",
			"runtime":        renderer.RuntimeValues(plugin),
		},
	}
	return writeYAMLFile(filepath.Join(rootDir, "values.yaml"), values)
}

func (hc *HelmPack) generateGatewayPluginTemplate(rootDir string, renderer gatewayPluginRenderer) error {
	template, err := loadHelmTemplate(renderer.TemplateName())
	if err != nil {
		return err
	}
	template = renderHelmTemplatePlaceholders(template, map[string]string{
		"__APPLICATION_DESCRIPTION__": strconv.Quote(hc.Manifest.Application.Description),
		"__APPLICATION_IDENTIFY__":    hc.Manifest.Application.Identifie,
		"__APPLICATION_VERSION__":     hc.Manifest.Application.Version,
	})
	return writeFile(filepath.Join(rootDir, renderer.OutputName()), template)
}
