package logic

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/w7panel/w7panel-zpk/common/function"
	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
	zpkservice "github.com/w7panel/w7panel-zpk/common/service/w7/zpk"
	"sigs.k8s.io/yaml"
)

// ImportRemoteFormulaDependency downloads one remote dependency and returns
// every application manifest contained in its complete info response. It is
// used by the child-application import API. It does not mutate or persist any
// local formula state; callers decide how to attach the returned manifests to
// their own formula.
func ImportRemoteFormulaDependency(
	ctx context.Context,
	dependency commonlogic.Depend,
) ([]*commonlogic.Manifest, error) {
	identifie := dependency.Identifie
	if identifie == "" {
		return nil, fmt.Errorf("远程依赖标识无效")
	}

	infoURL, err := zpkservice.RemoteFormulaInfoURL(dependency.From, identifie)
	if err != nil {
		return nil, err
	}
	infoURL, err = zpkservice.WithCompleteManifest(infoURL)
	if err != nil {
		return nil, err
	}
	requestCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	remoteInfo, err := (zpkservice.ZpkService{}).GetRemoteFormulaInfo(requestCtx, infoURL)
	if err != nil {
		return nil, fmt.Errorf("获取制品信息失败: %w", err)
	}
	if strings.TrimSpace(remoteInfo.Manifest) == "" {
		return nil, fmt.Errorf("制品信息缺少 manifest")
	}

	entries, err := parseRemoteFormulaManifests(*remoteInfo)
	if err != nil {
		return nil, err
	}
	if entries[0].Application.Identifie != identifie {
		return nil, fmt.Errorf("制品标识为 %q，与依赖标识 %q 不一致", entries[0].Application.Identifie, identifie)
	}
	imported := make([]*commonlogic.Manifest, 0, len(entries))
	for _, entry := range entries {
		manifest, err := importRemoteFormulaManifest(requestCtx, *remoteInfo, entry)
		if err != nil {
			return nil, fmt.Errorf("导入远程应用 %s 失败: %w", entry.Application.Identifie, err)
		}
		imported = append(imported, manifest)
	}
	return imported, nil
}

func parseRemoteFormulaManifests(remoteInfo zpkservice.FormulaInfoResp) ([]commonlogic.Manifest, error) {
	entries := make([]commonlogic.Manifest, 0)
	seen := make(map[string]struct{})
	add := func(raw string) error {
		if strings.TrimSpace(raw) == "" {
			return nil
		}
		manifest := commonlogic.Manifest{}
		if err := yaml.Unmarshal([]byte(raw), &manifest); err != nil {
			return fmt.Errorf("解析制品 manifest 失败: %w", err)
		}
		identifie := manifest.Application.Identifie
		if identifie == "" {
			return fmt.Errorf("制品 manifest 缺少应用标识")
		}
		if _, exists := seen[identifie]; exists {
			return nil
		}
		seen[identifie] = struct{}{}
		entries = append(entries, manifest)
		return nil
	}

	if err := add(remoteInfo.Manifest); err != nil {
		return nil, err
	}
	for _, raw := range remoteInfo.ChildManifests {
		if err := add(raw); err != nil {
			return nil, err
		}
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("制品信息没有返回 manifest")
	}
	return entries, nil
}

func importRemoteFormulaManifest(
	ctx context.Context,
	remoteInfo zpkservice.FormulaInfoResp,
	remoteManifest commonlogic.Manifest,
) (*commonlogic.Manifest, error) {
	depot, err := NewDepot()
	if err != nil {
		return nil, fmt.Errorf("获取本地制品仓库失败: %w", err)
	}

	identifie := remoteManifest.Application.Identifie
	remoteVersion := strings.TrimSpace(remoteManifest.Application.Version)
	if remoteVersion == "" {
		remoteVersion = "0"
	}

	helmURL := remoteInfo.HelmURLs[identifie]
	needsHelmAsset := remoteManifest.Application.Type == commonlogic.Help_App ||
		remoteManifest.Platform.Helm.ChartName != "" ||
		remoteManifest.Platform.Helm.Repository != "" ||
		len(remoteManifest.Platform.Helm.DependYamls) > 0
	logicalHelmPath := ""
	if needsHelmAsset {
		if helmURL == "" {
			return nil, fmt.Errorf("制品信息缺少 %s 的 helm_url", identifie)
		}
		logicalHelmPath = remoteDependencyStoragePath(identifie+":helm", remoteVersion, ".tgz")
		if err = downloadRemoteDependencyAsset(ctx, depot, helmURL, logicalHelmPath); err != nil {
			return nil, fmt.Errorf("下载完整 Helm 包失败: %w", err)
		}
	}
	logicalBackendPath := ""
	backendURL := remoteInfo.ZipURLs[identifie]
	if backendURL != "" {
		logicalBackendPath = remoteDependencyStoragePath(identifie+":backend", remoteVersion, ".zip")
		if err = downloadRemoteDependencyAsset(ctx, depot, backendURL, logicalBackendPath); err != nil {
			return nil, fmt.Errorf("下载后端代码包失败: %w", err)
		}
	}

	logicalFrontendPaths := make(map[string]string)
	if frontendURL := remoteInfo.WebZipURL[identifie]; frontendURL != "" {
		logicalFrontendPath := remoteDependencyStoragePath(identifie+":frontend", remoteVersion, ".zip")
		if err = downloadRemoteDependencyAsset(ctx, depot, frontendURL, logicalFrontendPath); err != nil {
			return nil, fmt.Errorf("下载前端代码包 %s 失败: %w", identifie, err)
		}
		logicalFrontendPaths[identifie] = logicalFrontendPath
	}

	return newImportedRemoteDependencyManifest(remoteManifest, logicalHelmPath, logicalBackendPath, logicalFrontendPaths), nil
}

// newImportedRemoteDependencyManifest converts a downloaded remote formula
// into the manifest used by the imported dependency. It starts from the
// complete remote manifest so no application configuration is lost; only
// artifact URLs and the local Helm source change. The original application
// type is intentionally preserved.
// The original Helm archive is kept intact, so any nested chart packages stay
// inside the dependency chart rather than being flattened into the parent.
func newImportedRemoteDependencyManifest(
	remoteManifest commonlogic.Manifest,
	logicalHelmPath string,
	logicalBackendPath string,
	logicalFrontendPaths map[string]string,
) *commonlogic.Manifest {
	manifest := remoteManifest
	if logicalHelmPath != "" {
		manifest.Platform.Helm.ChartName = "file://" + logicalHelmPath
		// A local ChartName must not retain the remote repository; otherwise the
		// Helm packer treats it as a repository chart and tries to download it again.
		manifest.Platform.Helm.Repository = ""
	}
	if logicalBackendPath != "" {
		manifest.Source = commonlogic.Source{Type: "zip", Url: "file://" + logicalBackendPath}
	} else {
		// Do not leave an inaccessible remote source URL in the imported
		// manifest when the remote formula has no backend attachment.
		manifest.Source = commonlogic.Source{}
	}
	if logicalFrontendPath := logicalFrontendPaths[manifest.Application.Identifie]; logicalFrontendPath != "" {
		manifest.Web = commonlogic.Source{Type: "zip", Url: "file://" + logicalFrontendPath}
	} else {
		// As with the backend package, remove a remote frontend URL when no
		// downloadable frontend attachment was returned.
		manifest.Web = commonlogic.Source{}
	}

	// Normalize the complete manifest after replacing source URLs. This keeps
	// legacy container fields and CodeAttachUrl consistent with the local code
	// package while preserving all modern fields.
	manifest = commonlogic.GetManifestV2(manifest)
	return &manifest
}

func remoteDependencyStoragePath(assetName, version, extension string) string {
	storageName := function.GetMd5(assetName + ":" + version)
	return fmt.Sprintf("/Storage/%s/%s%s", time.Now().Format("200601"), storageName, extension)
}

func downloadRemoteDependencyAsset(ctx context.Context, depot *Depot, sourceURL, logicalPath string) error {
	targetPath := depotAttachmentPath(depot, logicalPath)
	function.CreateDirIfNotExist(filepath.Dir(targetPath), os.ModePerm)
	return function.DownloadFile(ctx, sourceURL, targetPath)
}

func depotAttachmentPath(depot *Depot, logicalPath string) string {
	return filepath.Join(depot.GetBasePath(), filepath.FromSlash(strings.TrimLeft(logicalPath, "/")))
}
