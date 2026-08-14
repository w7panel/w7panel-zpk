package logic

import (
	"errors"
	"fmt"
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
