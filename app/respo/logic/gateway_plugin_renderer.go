package logic

import (
	"errors"
	"fmt"
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
	url := runtimeString(plugin.Runtime.Config, "url")
	if strings.TrimSpace(url) == "" {
		return errors.New("Higress Wasm 插件镜像地址不能为空")
	}
	phase := runtimeString(plugin.Runtime.Config, "phase")
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
	priority := runtimeInt(plugin.Runtime.Config, "priority")
	if priority < 0 || priority > 1000 {
		return errors.New("Higress Wasm 插件优先级必须在 0 到 1000 之间")
	}
	return nil
}

func (higressWasmV1Renderer) RuntimeValues(plugin commonlogic.GatewayPlugin) map[string]interface{} {
	phase := runtimeString(plugin.Runtime.Config, "phase")
	if phase == "" {
		phase = "UNSPECIFIED_PHASE"
	}
	return map[string]interface{}{
		"url":      runtimeString(plugin.Runtime.Config, "url"),
		"phase":    phase,
		"priority": runtimeInt(plugin.Runtime.Config, "priority"),
	}
}

func (higressWasmV1Renderer) TemplateName() string {
	return "higress-gateway-plugin.yaml.tpl"
}

func (higressWasmV1Renderer) OutputName() string {
	return "higress-gateway-plugin.yaml"
}

func runtimeString(config map[string]interface{}, key string) string {
	value, ok := config[key]
	if !ok || value == nil {
		return ""
	}
	if result, ok := value.(string); ok {
		return result
	}
	return fmt.Sprint(value)
}

func runtimeInt(config map[string]interface{}, key string) int {
	value, ok := config[key]
	if !ok || value == nil {
		return 0
	}
	switch current := value.(type) {
	case int:
		return current
	case int32:
		return int(current)
	case int64:
		return int(current)
	case float32:
		return int(current)
	case float64:
		return int(current)
	case string:
		result, _ := strconv.Atoi(current)
		return result
	default:
		return 0
	}
}
