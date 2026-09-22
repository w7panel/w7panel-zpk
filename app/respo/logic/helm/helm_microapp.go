package helm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
	"k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/yaml"
)

const (
	MicroAppPresentationModeSingleton = "singleton"
	MicroAppPresentationModeMultiple  = "multiple"
)

type MicroAppPresentation struct {
	Key  string `json:"key"`
	Mode string `json:"mode"`
}

type microAppTemplateConfig struct {
	Name      string `json:"name"`
	Identifie string `json:"identifie"`
	Type      string `json:"type"`
	Version   string `json:"version"`
	Order     int    `json:"order"`
}

func newMicroAppTemplateConfig(application logic2.Application) microAppTemplateConfig {
	return microAppTemplateConfig{
		Name:      application.Name,
		Identifie: application.Identifie,
		Type:      application.Type,
		Version:   application.Version,
		Order:     application.Order,
	}
}

// WithMicroAppSubchart adds an independent MicroApp chart without changing
// the root chart's values or MicroApp template.
func WithMicroAppSubchart(chartName string, application logic2.Application, bindings []logic2.Bindings) DynamicHelmPackageOption {
	return WithPresentedMicroAppSubchart(chartName, application, bindings, MicroAppPresentation{})
}

// WithPresentedMicroAppSubchart adds an independent MicroApp chart and
// declares how panels should group MicroApps that provide the same capability.
func WithPresentedMicroAppSubchart(chartName string, application logic2.Application, bindings []logic2.Bindings, presentation MicroAppPresentation) DynamicHelmPackageOption {
	chartName = strings.TrimSpace(chartName)
	presentation.Key = strings.TrimSpace(presentation.Key)
	presentation.Mode = strings.TrimSpace(presentation.Mode)
	bindingsCopy := append([]logic2.Bindings(nil), bindings...)
	templateConfig := newMicroAppTemplateConfig(application)
	cacheValue := struct {
		Kind           string                 `json:"kind"`
		ChartName      string                 `json:"chart_name"`
		TemplateConfig microAppTemplateConfig `json:"template_config"`
		Bindings       []logic2.Bindings      `json:"bindings"`
		Presentation   MicroAppPresentation   `json:"presentation"`
	}{
		Kind:           "microapp-subchart",
		ChartName:      chartName,
		TemplateConfig: templateConfig,
		Bindings:       bindingsCopy,
		Presentation:   presentation,
	}
	return func(options *dynamicHelmPackageOptions) error {
		if chartName == "" || chartName == "." || filepath.Base(chartName) != chartName {
			return fmt.Errorf("MicroApp Chart 名称无效: %q", chartName)
		}
		if err := validateMicroAppPresentation(presentation); err != nil {
			return err
		}
		return options.addTransform(cacheValue, func(chartDir string) error {
			return writeMicroAppSubchart(chartDir, chartName, application, bindingsCopy, presentation)
		})
	}
}

func validateMicroAppPresentation(presentation MicroAppPresentation) error {
	if presentation.Key == "" && presentation.Mode == "" {
		return nil
	}
	if presentation.Key == "" {
		return fmt.Errorf("MicroApp presentation-key 不能为空")
	}
	if errs := validation.IsValidLabelValue(presentation.Key); len(errs) > 0 {
		return fmt.Errorf("MicroApp presentation-key 无效 %q: %s", presentation.Key, strings.Join(errs, "; "))
	}
	if presentation.Mode != MicroAppPresentationModeSingleton && presentation.Mode != MicroAppPresentationModeMultiple {
		return fmt.Errorf("MicroApp presentation-mode 无效: %q", presentation.Mode)
	}
	return nil
}

func writeMicroAppSubchart(chartDir, chartName string, application logic2.Application, bindings []logic2.Bindings, presentation MicroAppPresentation) error {
	subchartDir := filepath.Join(chartDir, "charts", chartName)
	if err := os.MkdirAll(filepath.Join(subchartDir, "templates"), 0755); err != nil {
		return fmt.Errorf("创建 MicroApp Chart 目录失败: %w", err)
	}
	if err := writeYAMLFile(filepath.Join(subchartDir, "Chart.yaml"), ChartYAML{
		APIVersion: "v2",
		Name:       chartName,
		Version:    "0.1.0",
		Type:       "application",
		AppVersion: application.Version,
	}); err != nil {
		return err
	}

	manifest := logic2.Manifest{
		Application: application,
		Bindings:    bindings,
	}
	return (&HelmPack{IsSubFormula: true, MicroAppPresentation: presentation}).generateMicroAppTemplate(
		filepath.Join(subchartDir, "templates"),
		manifest,
	)
}

func buildMicroAppValues(bindings []logic2.Bindings) ([]map[string]interface{}, []map[string]interface{}) {
	menuConfigs := make([]map[string]interface{}, 0, len(bindings))
	backendConfigs := make([]map[string]interface{}, 0, len(bindings))
	for _, binding := range bindings {
		frontendProps := renderHelmValuesPlaceholdersMap(binding.BackendConfig.FrontendProps)
		values := make(map[string]string, len(frontendProps)+1)
		for key, value := range frontendProps {
			values[key] = value
		}
		values["app_name"] = `{{ include "common.fullname" . }}`
		frontendProps = values
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
			"frontend_props": frontendProps,
		})
	}
	return menuConfigs, backendConfigs
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
