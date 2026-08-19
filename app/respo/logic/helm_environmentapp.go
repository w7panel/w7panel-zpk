package logic

import (
	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
	v1 "k8s.io/api/core/v1"
)

const siteManagerPersistentVolumeClaim = "w7-sitemanager-site-manager"

const environmentAppCodeInstallShellType = "environment-code-install"

func withEnvironmentAppPVCName(platform logic2.Platform) logic2.Platform {
	volumes := append([]v1.Volume(nil), platform.Volumes...)
	for index := range volumes {
		if volumes[index].PersistentVolumeClaim != nil {
			volumes[index].PersistentVolumeClaim.ClaimName = siteManagerPersistentVolumeClaim
		}
	}
	platform.Volumes = volumes
	return platform
}

func environmentAppPodAffinityValues() map[string]interface{} {
	return map[string]interface{}{"podAffinity": map[string]interface{}{
		"requiredDuringSchedulingIgnoredDuringExecution": []interface{}{map[string]interface{}{
			"labelSelector": map[string]interface{}{"matchLabels": map[string]string{
				"app.kubernetes.io/instance": "w7-sitemanager",
				"app.kubernetes.io/name":     "site-manager",
				"w7.cc/identifie":            "w7-sitemanager",
			}},
			"topologyKey": "kubernetes.io/hostname",
		}},
	}}
}

func rewriteEnvironmentAppShells(manifest logic2.Manifest) []logic2.Shell {
	shells := append([]logic2.Shell(nil), manifest.Platform.Shells...)
	return append([]logic2.Shell{{
		Title: "安装环境代码包",
		Type:  environmentAppCodeInstallShellType,
		Image: "busybox:1.36.1",
		Shell: `set -eu
: "${DOMAIN_URL:?DOMAIN_URL is required}"
CODE_PACKAGE_URL='{{ .Values.codePackageUrl }}'
test -n "$CODE_PACKAGE_URL"
CODE_INSTALL_PATH="/www/wwwroot/$DOMAIN_URL"
mkdir -p "$CODE_INSTALL_PATH"
tmp_zip="$(mktemp /tmp/environment-code.XXXXXX.zip)"
trap 'rm -f "$tmp_zip"' EXIT
wget -q -O "$tmp_zip" "$CODE_PACKAGE_URL"
unzip -oq "$tmp_zip" -d "$CODE_INSTALL_PATH"`,
	}}, shells...)
}
