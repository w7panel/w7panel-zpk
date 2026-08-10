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
	sidecarInitTemplateAnnotation         = "w7.cc/sidecar-init-template"
	sidecarContainerTemplateAnnotation    = "w7.cc/sidecar-container-template"
	sidecarJobContainerTemplateAnnotation = "w7.cc/sidecar-job-container-template"
	sidecarVolumesTemplateAnnotation      = "w7.cc/sidecar-volumes-template"
	sidecarResourcesTemplateAnnotation    = "w7.cc/sidecar-resources-template"
	sidecarTargetPortValueAnnotation      = "w7.cc/sidecar-target-port-value"
)

// HelmSidecar describes the named-template contract exported by a sidecar
// artifact. The sidecar chart is packaged as a local library dependency.
type HelmSidecar struct {
	Chart                string `yaml:"chart" json:"chart"`
	InitTemplate         string `yaml:"initTemplate" json:"initTemplate"`
	ContainerTemplate    string `yaml:"containerTemplate" json:"containerTemplate"`
	JobContainerTemplate string `yaml:"jobContainerTemplate" json:"jobContainerTemplate"`
	VolumesTemplate      string `yaml:"volumesTemplate" json:"volumesTemplate"`
	ResourcesTemplate    string `yaml:"resourcesTemplate,omitempty" json:"resourcesTemplate,omitempty"`
	TargetPortValue      string `yaml:"targetPortValue,omitempty" json:"targetPortValue,omitempty"`
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
			return fmt.Errorf("sidecar Chart %s 与已有依赖冲突", sidecar.Chart)
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
		Chart:                metadata.Name,
		InitTemplate:         metadata.Annotations[sidecarInitTemplateAnnotation],
		ContainerTemplate:    metadata.Annotations[sidecarContainerTemplateAnnotation],
		JobContainerTemplate: metadata.Annotations[sidecarJobContainerTemplateAnnotation],
		VolumesTemplate:      metadata.Annotations[sidecarVolumesTemplateAnnotation],
		ResourcesTemplate:    metadata.Annotations[sidecarResourcesTemplateAnnotation],
		TargetPortValue:      metadata.Annotations[sidecarTargetPortValueAnnotation],
	}
	if sidecar.Chart == "" || sidecar.ContainerTemplate == "" {
		return HelmSidecar{}, fmt.Errorf("Chart 缺少必需的 sidecar container 模板契约注解")
	}
	return sidecar, nil
}

func (*HelmPack) generateSidecarResourcesTemplate(templatesDir string) error {
	return writeHelmTemplateFile(templatesDir, "w7panel-sidecar-resources.yaml", "sidecar-resources.yaml.tpl")
}

func setNestedSidecarValue(values map[string]interface{}, path string, value interface{}) {
	parts := strings.Split(strings.Trim(path, "."), ".")
	if len(parts) == 0 || parts[0] == "" {
		return
	}
	current := values
	for _, part := range parts[:len(parts)-1] {
		next, ok := current[part].(map[string]interface{})
		if !ok {
			next = make(map[string]interface{})
			current[part] = next
		}
		current = next
	}
	current[parts[len(parts)-1]] = value
}
