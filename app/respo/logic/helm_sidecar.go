package logic

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	copy2 "github.com/otiai10/copy"
	"github.com/w7panel/w7panel-zpk/common/function"
	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
	zpkservice "github.com/w7panel/w7panel-zpk/common/service/w7/zpk"
	"sigs.k8s.io/yaml"
)

const (
	sidecarHostContractAnnotation           = "w7.cc/sidecar-host-contract"
	sidecarHostContractV1                   = "v1"
	sidecarPodAnnotationsTemplateAnnotation = "w7.cc/sidecar-pod-annotations-template"
	sidecarHostAliasesTemplateAnnotation    = "w7.cc/sidecar-host-aliases-template"
	sidecarInitTemplateAnnotation           = "w7.cc/sidecar-init-template"
	sidecarContainerTemplateAnnotation      = "w7.cc/sidecar-container-template"
	sidecarJobContainerTemplateAnnotation   = "w7.cc/sidecar-job-container-template"
	sidecarVolumesTemplateAnnotation        = "w7.cc/sidecar-volumes-template"
	sidecarResourcesTemplateAnnotation      = "w7.cc/sidecar-resources-template"
)

// HelmSidecar describes the named-template contract exported by a sidecar
// artifact. The sidecar chart is packaged as a local library dependency.
type HelmSidecar struct {
	Chart                  string `yaml:"chart" json:"chart"`
	PodAnnotationsTemplate string `yaml:"podAnnotationsTemplate,omitempty" json:"podAnnotationsTemplate,omitempty"`
	HostAliasesTemplate    string `yaml:"hostAliasesTemplate,omitempty" json:"hostAliasesTemplate,omitempty"`
	InitTemplate           string `yaml:"initTemplate" json:"initTemplate"`
	ContainerTemplate      string `yaml:"containerTemplate" json:"containerTemplate"`
	JobContainerTemplate   string `yaml:"jobContainerTemplate" json:"jobContainerTemplate"`
	VolumesTemplate        string `yaml:"volumesTemplate" json:"volumesTemplate"`
	ResourcesTemplate      string `yaml:"resourcesTemplate,omitempty" json:"resourcesTemplate,omitempty"`
}

func (hc *HelmPack) prepareSidecarCharts(chartsDir string) error {
	infoURLs := requiredSidecarInfoURLs(hc.Manifest.Application)
	if len(infoURLs) == 0 {
		return nil
	}

	for _, infoURL := range infoURLs {
		workDir, err := os.MkdirTemp("", "w7panel-zpk-sidecar-")
		if err != nil {
			return fmt.Errorf("创建 sidecar 打包目录失败: %w", err)
		}
		defer os.RemoveAll(workDir)

		sourceDir, err := downloadRemoteSidecarChart(context.Background(), infoURL, workDir)
		if err != nil {
			return fmt.Errorf("下载 sidecar 制品 %s 失败: %w", infoURL, err)
		}
		sidecar, err := loadHelmSidecar(sourceDir)
		if err != nil {
			return fmt.Errorf("读取 sidecar 制品 %s 的 Chart 失败: %w", infoURL, err)
		}
		targetDir := filepath.Join(chartsDir, sidecar.Chart)
		if function.FileExists(targetDir) {
			if err := os.RemoveAll(targetDir); err != nil {
				return fmt.Errorf("清理已有 sidecar Chart %s 失败: %w", sidecar.Chart, err)
			}
		}
		if err := copy2.Copy(sourceDir, targetDir); err != nil {
			return fmt.Errorf("复制 sidecar Chart %s 失败: %w", sidecar.Chart, err)
		}
		hc.Sidecars = append(hc.Sidecars, sidecar)
	}
	return nil
}

func downloadRemoteSidecarChart(ctx context.Context, infoURL, workDir string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	requestCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	remoteInfo, err := (zpkservice.ZpkService{}).GetRemoteFormulaInfo(requestCtx, infoURL)
	if err != nil {
		return "", fmt.Errorf("获取制品信息失败: %w", err)
	}

	archivePath := filepath.Join(workDir, "sidecar.tgz")
	if err := function.DownloadFile(requestCtx, remoteInfo.HelmURL, archivePath); err != nil {
		return "", fmt.Errorf("下载 Helm 包失败: %w", err)
	}
	chartDir := filepath.Join(workDir, "chart")
	if err := os.MkdirAll(chartDir, 0o755); err != nil {
		return "", fmt.Errorf("创建 Helm 解包目录失败: %w", err)
	}
	if err := function.UnzipHelmPackage(archivePath, chartDir); err != nil {
		return "", fmt.Errorf("解包 Helm 包失败: %w", err)
	}
	return chartDir, nil
}

func requiredSidecarInfoURLs(application commonlogic.Application) []string {
	if !application.RegisterSite {
		return nil
	}
	return []string{"https://zpk.w7.cc/zpk/respo/info/w7panel-cloudnoauth"}
}

func loadHelmSidecar(chartDir string) (HelmSidecar, error) {
	data, err := os.ReadFile(filepath.Join(chartDir, "Chart.yaml"))
	if err != nil {
		return HelmSidecar{}, err
	}
	metadata := ChartYAML{}
	if err := yaml.Unmarshal(data, &metadata); err != nil {
		return HelmSidecar{}, err
	}
	if metadata.Type != "library" || metadata.Annotations["w7.cc/manifest-type"] != commonlogic.SidecarApp {
		return HelmSidecar{}, fmt.Errorf("Chart 必须是 manifest-type=sidecar 的 library Chart")
	}
	sidecar := HelmSidecar{
		Chart:                  metadata.Name,
		PodAnnotationsTemplate: metadata.Annotations[sidecarPodAnnotationsTemplateAnnotation],
		HostAliasesTemplate:    metadata.Annotations[sidecarHostAliasesTemplateAnnotation],
		InitTemplate:           metadata.Annotations[sidecarInitTemplateAnnotation],
		ContainerTemplate:      metadata.Annotations[sidecarContainerTemplateAnnotation],
		JobContainerTemplate:   metadata.Annotations[sidecarJobContainerTemplateAnnotation],
		VolumesTemplate:        metadata.Annotations[sidecarVolumesTemplateAnnotation],
		ResourcesTemplate:      metadata.Annotations[sidecarResourcesTemplateAnnotation],
	}
	if sidecar.Chart == "" || sidecar.ContainerTemplate == "" {
		return HelmSidecar{}, fmt.Errorf("Chart 缺少必需的 sidecar container 模板契约注解")
	}
	return sidecar, nil
}

func (*HelmPack) generateSidecarHelpersTemplate(templatesDir string) error {
	return writeHelmTemplateFile(templatesDir, "_w7panel-sidecars.tpl", "_w7panel-sidecars.tpl")
}

func (*HelmPack) generateSidecarResourcesTemplate(templatesDir string) error {
	return writeHelmTemplateFile(templatesDir, "w7panel-sidecar-resources.yaml", "sidecar-resources.yaml.tpl")
}

// configureHelmSidecarHost applies sidecar values to a user supplied Helm
// chart. The workload templates themselves are never rewritten: the chart
// must explicitly opt in to the v1 host contract and provide the required
// include points.
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
	if err := validateHelmSidecarHostSlots(filepath.Join(chartDir, "templates"), hc.Sidecars); err != nil {
		return err
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

	values["w7panelSidecars"] = hc.Sidecars
	return writeYAMLFile(valuesPath, values)
}

func validateHelmSidecarHostSlots(templatesDir string, sidecars []HelmSidecar) error {
	required := map[string]bool{"w7panel.sidecars.containers": true}
	for _, sidecar := range sidecars {
		required["w7panel.sidecars.podAnnotations"] = required["w7panel.sidecars.podAnnotations"] || sidecar.PodAnnotationsTemplate != ""
		required["w7panel.sidecars.hostAliases"] = required["w7panel.sidecars.hostAliases"] || sidecar.HostAliasesTemplate != ""
		required["w7panel.sidecars.volumes"] = required["w7panel.sidecars.volumes"] || sidecar.VolumesTemplate != ""
		required["w7panel.sidecars.initContainers"] = required["w7panel.sidecars.initContainers"] || sidecar.InitTemplate != ""
	}

	var templates strings.Builder
	err := filepath.WalkDir(templatesDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || (!strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml") && !strings.HasSuffix(entry.Name(), ".tpl")) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		templates.Write(data)
		templates.WriteByte('\n')
		return nil
	})
	if err != nil {
		return fmt.Errorf("检查宿主 Helm sidecar 插槽失败: %w", err)
	}
	content := templates.String()
	for name, needed := range required {
		if needed && !strings.Contains(content, `include "`+name+`"`) {
			return fmt.Errorf("宿主 Helm Chart 缺少 sidecar 插槽 %s", name)
		}
	}
	return nil
}
