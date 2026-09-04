package logic

import (
	"strings"

	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
)

const (
	environmentStorageVolumeName = "site-storage"
	managedCodeInstallShellImage = "busybox:1.36.1"
)

// environmentCodeInstallShell is added to the generated chart as an internal
// shell task.  It deliberately uses the same volume mount as the workload;
// the environment editor persists that mount at /www/wwwroot backed by the
// shared site-storage PVC. DOMAIN_URL and the package URL are rendered from
// the final chart values, so installer-selected values are used at runtime.
const environmentCodeInstallShell = `set -eu
domain_url={{ .Values.DOMAIN_URL | quote }}
: "${domain_url:?DOMAIN_URL is required}"
code_package_url={{ .Values.environment.code.packageUrl | quote }}
test -n "$code_package_url"
code_install_path="/www/wwwroot/$domain_url"
mkdir -p "$code_install_path"
tmp_zip="$(mktemp /tmp/environment-code.XXXXXX)"
trap 'rm -f "$tmp_zip"' EXIT
wget -q -O "$tmp_zip" "$code_package_url"
unzip -oq "$tmp_zip" -d "$code_install_path"`

func withEnvironmentAppImages(platform logic2.Platform) logic2.Platform {
	platform.ContainerV2s = append([]logic2.ContainerV2(nil), platform.ContainerV2s...)
	for index := range platform.ContainerV2s {
		platform.ContainerV2s[index].Image = strings.ReplaceAll(
			platform.ContainerV2s[index].Image,
			"{version}",
			"{{ .Values.IMAGE_VERSION }}",
		)
	}
	return platform
}

func (hc *HelmPack) addEnvironmentAppValues(values map[string]interface{}) error {
	values["environment"] = map[string]interface{}{
		"code": map[string]interface{}{
			"packageUrl": environmentAppCodePackageURL(hc.Manifest),
		},
	}
	return nil
}

func (hc *HelmPack) environmentAppHelmValuesOptions() helmValuesOptions {
	options := hc.defaultHelmValuesOptions()
	// Environment storage, Sysbox settings, and ingress are persisted by the
	// editor in the manifest. The packer only performs image placeholder
	// substitution and adds environment-specific values required by templates.
	options.platform = withEnvironmentAppImages(hc.Manifest.Platform)
	if strings.TrimSpace(hc.Manifest.Source.Url) != "" {
		// Keep the legacy installer lifecycle: one pre-install,pre-upgrade
		// hook with weight -3, rather than separate install/upgrade jobs.
		options.platform.Shells = append(
			append([]logic2.Shell(nil), options.platform.Shells...),
			logic2.Shell{
				Title: "安装环境代码",
				Type:  "pre-install,pre-upgrade",
				Image: managedCodeInstallShellImage,
				Shell: environmentCodeInstallShell,
			},
		)
	}
	options.addValues = hc.addEnvironmentAppValues
	return options
}

func (hc *HelmPack) packEnvironmentApp(rootDir, templatesDir string) error {
	return hc.packWorkloadApplication(
		rootDir,
		templatesDir,
		hc.environmentAppHelmValuesOptions(),
		true,
		nil,
	)
}

func environmentAppCodePackageURL(manifest logic2.Manifest) string {
	if strings.TrimSpace(manifest.Source.Url) == "" {
		return ""
	}
	depot, _ := NewDepot()
	codePackageURL, _ := depot.GetFormulaBackendZipDownloadUrlByApplication(
		manifest.Application,
		strings.TrimPrefix(manifest.Source.Url, "file://"),
		false,
	)
	return codePackageURL
}
