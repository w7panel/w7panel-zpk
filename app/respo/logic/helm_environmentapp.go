package logic

import (
	"fmt"
	"strings"

	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
)

const (
	environmentStorageVolumeName              = "site-storage"
	managedCodeInstallShellImage              = "busybox:1.36.1"
	environmentImageLanguageAnnotation        = "w7.cc/image_language"
	environmentNginxVhostAnnotation           = "w7.cc/nginx_vhost_template"
	environmentNginxRestartRevisionAnnotation = "w7.cc/nginx-restart-revision"
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

// environmentNginxVhostShell writes the rendered site-manager vhost into the
// same nginx-dir subtree consumed by the embedded w7-sitemanager-nginx chart. It is kept
// as a shell job (like environmentCodeInstallShell) so both lifecycle tasks
// use the normal Helm shell-job renderer and hook handling.
const environmentNginxVhostShell = `{{- $rawDomain := toString .Values.DOMAIN_URL -}}
{{- $domain := replace "https://" "" $rawDomain -}}
{{- $domain = replace "http://" "" $domain -}}
{{- $domain = trimSuffix "/" $domain -}}
{{- $serverName := replace "," " " $domain -}}
{{- $primaryDomain := first (splitList " " (trimAll " " $serverName)) -}}
{{- $rootDir := print "/www/wwwroot/" $primaryDomain -}}
{{- $k8sDomain := print (include "common.fullname" .) "." .Release.Namespace ".svc.cluster.local" -}}
{{- $upstream := print .Release.Name "-" $primaryDomain | sha256sum | trunc 16 -}}
{{- $config := .Values.environment.site.nginxVhostTemplate -}}
{{- $config = replace "{UPSTREAM_APP_NAME}" $upstream $config -}}
{{- $config = replace "{SERVER_NAME}" $serverName $config -}}
{{- $config = replace "{LOG_DIR}" $primaryDomain $config -}}
{{- $config = replace "{ROOT_DIR}" $rootDir $config -}}
{{- $config = replace "{K8S_DOMAIN}" $k8sDomain $config -}}
set -eu
nginx_vhost_file={{ print $primaryDomain ".conf" | quote }}
nginx_vhost_config_b64={{ $config | b64enc | quote }}
mkdir -p /www/server/nginx/conf.d
echo -n "$nginx_vhost_config_b64" | base64 -d > "/www/server/nginx/conf.d/$nginx_vhost_file"
test -s "/www/server/nginx/conf.d/$nginx_vhost_file"`

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
	annotation := hc.Manifest.Application.Annotation
	if annotation == nil {
		annotation = map[string]interface{}{}
	}
	language := ""
	if value, ok := annotation[environmentImageLanguageAnnotation]; ok {
		language = strings.TrimSpace(environmentAnnotationString(value))
	}
	nginxVhostTemplate := ""
	if value, ok := annotation[environmentNginxVhostAnnotation]; ok {
		nginxVhostTemplate = strings.TrimSpace(environmentAnnotationString(value))
	}
	values["environment"] = map[string]interface{}{
		"code": map[string]interface{}{
			"packageUrl": environmentAppCodePackageURL(hc.Manifest),
		},
		"site": map[string]interface{}{
			"language":           language,
			"nginxVhostTemplate": nginxVhostTemplate,
		},
	}
	return nil
}

func (hc *HelmPack) environmentAppHelmValuesOptions() helmValuesOptions {
	options := hc.defaultHelmValuesOptions()
	// The environment editor persists storage, Sysbox, and gateway ingress
	// changes directly in the manifest. Helm packaging must consume that
	// contract instead of injecting a second, potentially conflicting shape.
	platform := hc.Manifest.Platform
	// Keep image placeholders dynamic while preserving the storage/runtime
	// contract already persisted by the environment editor.
	options.platform = withEnvironmentAppImages(platform)
	nginxVhostTemplate := ""
	if value, ok := hc.Manifest.Application.Annotation[environmentNginxVhostAnnotation]; ok {
		nginxVhostTemplate = strings.TrimSpace(environmentAnnotationString(value))
	}
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
	if nginxVhostTemplate != "" {
		options.platform.Shells = append(options.platform.Shells, logic2.Shell{
			Title: "安装环境 Nginx 配置",
			Type:  "pre-install,pre-upgrade",
			Image: managedCodeInstallShellImage,
			Shell: environmentNginxVhostShell,
		})
	}
	options.addValues = hc.addEnvironmentAppValues
	return options
}

func (hc *HelmPack) packEnvironmentApp(rootDir, templatesDir string) error {
	hc.prepareEnvironmentAppSubManifests()
	return hc.packWorkloadApplication(
		rootDir,
		templatesDir,
		hc.environmentAppHelmValuesOptions(),
		true,
		nil,
	)
}

// prepareEnvironmentAppSubManifests applies environment-only metadata before
// the generic sub-chart packer runs. This keeps Nginx knowledge out of the
// shared generateSubCharts loop while still making the imported child roll on
// every Helm upgrade of an environment application.
func (hc *HelmPack) prepareEnvironmentAppSubManifests() {
	for identify, child := range hc.SubManifest {
		hc.SubManifest[identify] = withEnvironmentNginxRestartAnnotation(hc.Manifest, child)
	}
}

// withEnvironmentNginxRestartAnnotation adds an upgrade marker only to the
// imported w7-sitemanager-nginx child application. The child chart renders application
// annotations into its Pod template, so changing the Helm release revision
// causes that workload to roll by default.
func withEnvironmentNginxRestartAnnotation(parent, child logic2.Manifest) logic2.Manifest {
	if parent.Application.Type != logic2.EnvironmentApp {
		return child
	}
	identify := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(child.Application.Identifie), "_", "-"))
	if identify != "w7-sitemanager-nginx" {
		return child
	}
	annotations := make(map[string]interface{}, len(child.Application.Annotation)+1)
	for key, value := range child.Application.Annotation {
		annotations[key] = value
	}
	annotations[environmentNginxRestartRevisionAnnotation] = "{{ .Release.Revision }}"
	child.Application.Annotation = annotations
	return child
}

func environmentAnnotationString(value interface{}) string {
	switch item := value.(type) {
	case string:
		return item
	case bool:
		if item {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprint(item)
	}
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
