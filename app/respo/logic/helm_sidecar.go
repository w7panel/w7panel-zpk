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
	sidecarHostContractAnnotation = "w7.cc/sidecar-host-contract"
	sidecarHostContractV1         = "v1"
)

// HelmSidecar identifies a sidecar chart packaged as a local library
// dependency. The chart name is the sidecar artifact identifier.
type HelmSidecar struct {
	Chart string `yaml:"chart" json:"chart"`
}

type helmChartDownload struct {
	InfoURL string
	Chart   string
	Sidecar bool
}

type helmSidecarInfo struct {
	InfoURL string
	Chart   string
}

func (hc *HelmPack) prepareRemoteHelmSidecars(chartsDir string) error {
	chartDownloads, err := requiredRemoteHelmCharts(hc.Manifest)
	if err != nil {
		return err
	}
	if len(chartDownloads) == 0 {
		return nil
	}

	for _, chartDownload := range chartDownloads {
		infoURL := chartDownload.InfoURL
		chartName := chartDownload.Chart
		workDir, err := os.MkdirTemp("", "w7panel-zpk-remote-helm-")
		if err != nil {
			return fmt.Errorf("创建远程 Helm 打包目录失败: %w", err)
		}
		defer os.RemoveAll(workDir)

		sourceDir, err := downloadRemoteHelmChart(context.Background(), infoURL, workDir)
		if err != nil {
			return fmt.Errorf("下载远程 Helm 制品 %s 失败: %w", infoURL, err)
		}
		targetDir := filepath.Join(chartsDir, chartName)
		if function.FileExists(targetDir) {
			if err := os.RemoveAll(targetDir); err != nil {
				return fmt.Errorf("清理已有远程 Helm Chart %s 失败: %w", chartName, err)
			}
		}
		if err := copy2.Copy(sourceDir, targetDir); err != nil {
			return fmt.Errorf("复制远程 Helm Chart %s 失败: %w", chartName, err)
		}
		hc.Sidecars = append(hc.Sidecars, HelmSidecar{Chart: chartName})
	}
	return nil
}

// downloadRemoteHelmChart downloads a ZPK Helm artifact from its formula info
// endpoint and extracts it into a temporary chart directory.
func downloadRemoteHelmChart(ctx context.Context, infoURL, workDir string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	requestCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	remoteInfo, err := (zpkservice.ZpkService{}).GetRemoteFormulaInfo(requestCtx, infoURL)
	if err != nil {
		return "", fmt.Errorf("获取制品信息失败: %w", err)
	}
	helmURL := strings.TrimSpace(remoteInfo.HelmURL)
	if helmURL == "" {
		return "", fmt.Errorf("制品信息缺少 helm_url")
	}

	archivePath := filepath.Join(workDir, "remote-helm.tgz")
	if err := function.DownloadFile(requestCtx, helmURL, archivePath); err != nil {
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

func requiredRemoteHelmCharts(manifest commonlogic.Manifest) ([]helmChartDownload, error) {
	charts := make([]helmChartDownload, 0)
	seen := make(map[string]struct{})
	for _, sidecar := range requiredSidecarInfoURLs(manifest.Application) {
		if _, exists := seen[sidecar.Chart]; exists {
			continue
		}
		seen[sidecar.Chart] = struct{}{}
		charts = append(charts, helmChartDownload{
			InfoURL: sidecar.InfoURL,
			Chart:   sidecar.Chart,
			Sidecar: true,
		})
	}
	return charts, nil
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
