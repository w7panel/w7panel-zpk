package logic

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
	"sigs.k8s.io/yaml"
)

func WithMicroAppBindings(names []string, bindings []logic2.Bindings) DynamicHelmPackageOption {
	replacementNames := append([]string(nil), names...)
	replacements := append([]logic2.Bindings(nil), bindings...)
	cacheValue := struct {
		Names    []string          `json:"names"`
		Bindings []logic2.Bindings `json:"bindings"`
	}{
		Names:    replacementNames,
		Bindings: replacements,
	}
	return func(options *dynamicHelmPackageOptions) error {
		return options.addTransform(cacheValue, func(chartDir string) error {
			return replaceHelmChartMicroAppBindings(chartDir, replacementNames, replacements)
		})
	}
}

func buildMicroAppValues(bindings []logic2.Bindings) ([]map[string]interface{}, []map[string]interface{}) {
	menuConfigs := make([]map[string]interface{}, 0, len(bindings))
	backendConfigs := make([]map[string]interface{}, 0, len(bindings))
	for _, binding := range bindings {
		menuConfigs = append(menuConfigs, map[string]interface{}{
			"title":   binding.Title,
			"name":    binding.Name,
			"status":  binding.Status,
			"support": binding.Support,
			"menu":    binding.Menu,
		})
		backendConfigs = append(backendConfigs, map[string]interface{}{
			"role":         binding.Name,
			"load_mode":    binding.LoadMode,
			"type":         binding.BackendConfig.Type,
			"backend_url":  renderHelmValuesPlaceholders(binding.BackendConfig.BackendUrl),
			"backend_port": binding.BackendConfig.BackendPort,
			"proxy_request": logic2.RequestProxy{
				Headers: renderHelmValuesPlaceholdersMap(binding.BackendConfig.RequestProxy.Headers),
				Query:   renderHelmValuesPlaceholdersMap(binding.BackendConfig.RequestProxy.Query),
			},
			"frontend_props": renderHelmValuesPlaceholdersMap(binding.BackendConfig.FrontendProps),
		})
	}
	return menuConfigs, backendConfigs
}

func replaceHelmChartMicroAppBindings(chartDir string, names []string, replacements []logic2.Bindings) error {
	valuesPath := filepath.Join(chartDir, "values.yaml")
	valuesContent, err := os.ReadFile(valuesPath)
	if err != nil {
		return fmt.Errorf("读取 Helm values.yaml 失败: %w", err)
	}

	values := make(map[string]interface{})
	if err = yaml.Unmarshal(valuesContent, &values); err != nil {
		return fmt.Errorf("解析 Helm values.yaml 失败: %w", err)
	}

	nameSet := make(map[string]struct{}, len(names))
	for _, name := range names {
		if name = strings.TrimSpace(name); name != "" {
			nameSet[name] = struct{}{}
		}
	}
	menuValues := removeMicroAppValues(values["bindings"], "name", nameSet)
	backendValues := removeMicroAppValues(values["backend_config"], "role", nameSet)
	replacementMenus, replacementBackends := buildMicroAppValues(replacements)
	for _, replacement := range replacementMenus {
		menuValues = append(menuValues, replacement)
	}
	for _, replacement := range replacementBackends {
		backendValues = append(backendValues, replacement)
	}
	values["bindings"] = menuValues
	values["backend_config"] = backendValues

	valuesContent, err = yaml.Marshal(values)
	if err != nil {
		return fmt.Errorf("序列化 Helm values.yaml 失败: %w", err)
	}
	if err = os.WriteFile(valuesPath, valuesContent, 0644); err != nil {
		return fmt.Errorf("写入 Helm values.yaml 失败: %w", err)
	}
	return nil
}

func removeMicroAppValues(value interface{}, key string, names map[string]struct{}) []interface{} {
	result := make([]interface{}, 0)
	items, ok := value.([]interface{})
	if !ok {
		return result
	}
	for _, item := range items {
		itemMap, isMap := item.(map[string]interface{})
		if isMap {
			if name, isString := itemMap[key].(string); isString {
				if _, exists := names[name]; exists {
					continue
				}
			}
		}
		result = append(result, item)
	}
	return result
}

func readHelmChartName(chartDir string) (string, error) {
	chartContent, err := os.ReadFile(filepath.Join(chartDir, "Chart.yaml"))
	if err != nil {
		return "", fmt.Errorf("读取 Helm Chart.yaml 失败: %w", err)
	}
	chart := struct {
		Name string `json:"name"`
	}{}
	if err = yaml.Unmarshal(chartContent, &chart); err != nil {
		return "", fmt.Errorf("解析 Helm Chart.yaml 失败: %w", err)
	}
	chart.Name = strings.TrimSpace(chart.Name)
	if chart.Name == "" || filepath.Base(chart.Name) != chart.Name || chart.Name == "." {
		return "", fmt.Errorf("Helm Chart 名称无效: %q", chart.Name)
	}
	return chart.Name, nil
}
