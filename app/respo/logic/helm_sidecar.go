package logic

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
	"sigs.k8s.io/yaml"
)

const (
	sidecarHostContractAnnotation = "w7.cc/sidecar-host-contract"
	sidecarHostContractV1         = "v1"
)

// HelmSidecar identifies a sidecar chart packaged as a local library
// dependency. The chart name is the sidecar artifact identifier.
type HelmSidecar struct {
	Chart string `yaml:"chart" json:"chart"`
}

type helmSidecarInfo struct {
	InfoURL string
	Chart   string
}

func sidecarChartReferences(sidecars []HelmSidecar) []map[string]string {
	refs := make([]map[string]string, 0, len(sidecars))
	for _, sidecar := range sidecars {
		refs = append(refs, map[string]string{"chart": sidecar.Chart})
	}
	return refs
}

func requiredSidecarInfoURLs(application commonlogic.Application) []helmSidecarInfo {
	if !application.RegisterSite {
		return nil
	}
	return []helmSidecarInfo{{
		InfoURL: "https://zpk.w7.cc/zpk/respo/info/w7panel-cloudnoauth",
		Chart:   "w7panel-cloudnoauth",
	}}
}

func (*HelmPack) generateSidecarHelpersTemplate(templatesDir string) error {
	return writeHelmTemplateFile(templatesDir, "_w7panel-sidecars.tpl", "_w7panel-sidecars.tpl")
}

func (*HelmPack) generateSidecarResourcesTemplate(templatesDir string) error {
	return writeHelmTemplateFile(templatesDir, "w7panel-sidecar-resources.yaml", "sidecar-resources.yaml.tpl")
}

// configureHelmSidecarHost registers sidecar chart references in a user
// supplied Helm chart. The workload templates themselves are never rewritten:
// the chart must explicitly opt in to the v1 host contract and provide the
// required include points. The generated sidecar helpers read each sidecar's
// contract directly from the child Chart metadata at render time.
func (hc *HelmPack) configureHelmSidecarHost(chartDir string) error {
	if len(hc.Sidecars) == 0 {
		return nil
	}

	chartPath := filepath.Join(chartDir, "Chart.yaml")
	chartData, err := os.ReadFile(chartPath)
	if err != nil {
		return fmt.Errorf("读取宿主 Helm Chart.yaml 失败: %w", err)
	}
	metadata := ChartYAML{}
	if err := yaml.Unmarshal(chartData, &metadata); err != nil {
		return fmt.Errorf("解析宿主 Helm Chart.yaml 失败: %w", err)
	}
	if metadata.Annotations[sidecarHostContractAnnotation] != sidecarHostContractV1 {
		return nil
	}
	valuesPath := filepath.Join(chartDir, "values.yaml")
	values := make(map[string]interface{})
	if data, readErr := os.ReadFile(valuesPath); readErr == nil {
		if len(strings.TrimSpace(string(data))) != 0 {
			if err := yaml.Unmarshal(data, &values); err != nil {
				return fmt.Errorf("解析宿主 Helm values.yaml 失败: %w", err)
			}
		}
	} else if !os.IsNotExist(readErr) {
		return fmt.Errorf("读取宿主 Helm values.yaml 失败: %w", readErr)
	}

	values["w7panelSidecars"] = sidecarChartReferences(hc.Sidecars)
	return writeYAMLFile(valuesPath, values)
}
