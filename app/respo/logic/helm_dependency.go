package logic

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	copy2 "github.com/otiai10/copy"
	"github.com/w7panel/w7panel-zpk/common/function"
	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
	zpkservice "github.com/w7panel/w7panel-zpk/common/service/w7/zpk"
)

type helmChartDownload struct {
	InfoURL string
	Chart   string
	Sidecar bool
}

func (hc *HelmPack) prepareRemoteHelmCharts(chartsDir string) error {
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
		if !isValidHelmChartDirectoryName(chartName) {
			return fmt.Errorf("远程 Helm 制品 %s 缺少有效的 Chart 标识", infoURL)
		}
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
		if chartDownload.Sidecar {
			hc.Sidecars = append(hc.Sidecars, HelmSidecar{Chart: chartName})
		}
	}
	return nil
}

// downloadRemoteHelmChart downloads a ZPK Helm artifact from its formula info
// endpoint and extracts it into a temporary chart directory. It is shared by
// sidecar charts and regular embedded Helm dependencies.
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

	archivePath := filepath.Join(workDir, "remote-helm.tgz")
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

	for _, dependency := range manifest.Platform.Depends {
		if !isEmbeddedHelmDependency(dependency) {
			continue
		}
		chartName := strings.TrimSpace(dependency.Identifie)
		if _, exists := seen[chartName]; exists {
			continue
		}
		infoURL, err := helmDependencyInfoURL(dependency.From, chartName)
		if err != nil {
			return nil, fmt.Errorf("依赖 %s 的 Helm 制品来源无效: %w", chartName, err)
		}
		seen[chartName] = struct{}{}
		charts = append(charts, helmChartDownload{
			InfoURL: infoURL,
			Chart:   chartName,
		})
	}
	return charts, nil
}

// helmDependencyInfoURL accepts either a complete formula-info endpoint or a
// ZPK service base URL. The latter is still used by existing dependency data,
// while the downloader itself always receives one canonical endpoint.
func helmDependencyInfoURL(from, chartName string) (string, error) {
	from = strings.TrimSpace(from)
	chartName = strings.TrimSpace(chartName)
	if from == "" {
		return "", fmt.Errorf("来源 URL 不能为空")
	}
	if !isValidHelmChartDirectoryName(chartName) {
		return "", fmt.Errorf("Chart 标识无效: %q", chartName)
	}

	parsed, err := url.Parse(from)
	if err != nil {
		return "", fmt.Errorf("解析来源 URL 失败: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("来源 URL 必须包含协议和主机")
	}

	const infoPath = "/zpk/respo/info"
	path := strings.TrimRight(parsed.Path, "/")
	if path == infoPath || strings.HasPrefix(path, infoPath+"/") {
		// A complete endpoint is authoritative; do not append the identifier a
		// second time when the caller already provided it.
		return parsed.String(), nil
	}
	if path == "" {
		path = infoPath
	} else if path == "/zpk" {
		path += "/respo/info"
	} else {
		path += infoPath
	}
	// chartName has already been checked for path separators. Keeping it in
	// URL.Path lets url.URL.String perform the escaping exactly once.
	parsed.Path = path + "/" + chartName
	parsed.RawPath = ""
	return parsed.String(), nil
}

func isValidHelmChartDirectoryName(name string) bool {
	return name != "" && name != "." && name != ".." && !strings.ContainsAny(name, `/\\`)
}
