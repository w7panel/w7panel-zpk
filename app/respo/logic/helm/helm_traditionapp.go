package helm

import (
	"fmt"
	"strings"

	formulalogic "github.com/w7panel/w7panel-zpk/app/respo/logic/formula"
	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
	v1 "k8s.io/api/core/v1"
)

const (
	traditionStorageVolumeName             = "site-storage"
	managedCodeInstallShellImage           = "busybox:stable-uclibc"
	traditionImageLanguageAnnotation       = "w7.cc/image_language"
	traditionNginxVhostAnnotation          = "w7.cc/nginx_vhost_template"
	traditionToolRestartRevisionAnnotation = "w7.cc/tradition-tool-restart-revision"
	traditionCodeInstallJobTitle           = "安装传统应用代码"
	traditionCodeUninstallJobTitle         = "卸载传统应用代码"
	traditionNginxVhostJobTitle            = "安装传统应用 NGINX 配置"
	traditionNginxVhostUninstallJobTitle   = "卸载传统应用 NGINX 配置"
)

// traditionCodeInstallShell is added to the generated chart as an internal
// shell task.  It deliberately uses the same volume mount as the workload;
// the traditional application editor persists that mount at /www/wwwroot,
// backed by the matching domain subPath in the shared site-storage PVC.
// DOMAIN_URL and the package URL are rendered from
// the final chart values, so installer-selected values are used at runtime.
const traditionCodeInstallShell = `set -eu
domain_url={{ .Values.DOMAIN_URL | quote }}
: "${domain_url:?DOMAIN_URL is required}"
code_package_url={{ .Values.tradition.code.packageUrl | quote }}
test -n "$code_package_url"
code_install_path="/www/wwwroot"
case "$domain_url" in
  .|..|*[!A-Za-z0-9._,-]*) echo "invalid traditional application code path" >&2; exit 1 ;;
esac
plugin_state_dir="/var/lib/w7-tradition-plugin/.w7-tradition-plugin/$domain_url"
policy_json="${TRADITION_PLUGIN_POLICY:-}"
if [ -z "$policy_json" ]; then
  policy_json='{"platform":{"tradition":{"plugins":[]}}}'
fi
mkdir -p "$code_install_path"
mkdir -p "$plugin_state_dir"
tmp_dir="$(mktemp -d /tmp/tradition-code.XXXXXX)"
trap 'rm -rf "$tmp_dir"' EXIT
tmp_zip="$tmp_dir/application.zip"
package_dir="$tmp_dir/package"
policy_file="$tmp_dir/policy.json"
mkdir -p "$package_dir"
wget -q -O "$tmp_zip" "$code_package_url"
unzip -oq "$tmp_zip" -d "$package_dir"
unzip -oq "$tmp_zip" -d "$code_install_path"
echo "$policy_json" > "$policy_file"
w7-tradition-plugin app update \
  --plugin-state-dir "$plugin_state_dir" \
  --site-dir "$code_install_path" \
  --package-dir "$package_dir" \
  --policy-file "$policy_file"`

// traditionCodeUninstallShell clears only the mounted domain directory from
// the shared site-storage PVC. The mount point and PVC remain intact.
const traditionCodeUninstallShell = `set -eu
domain_url={{ .Values.DOMAIN_URL | quote }}
: "${domain_url:?DOMAIN_URL is required}"
case "$domain_url" in
  .|..|*[!A-Za-z0-9._,-]*) echo "refusing to remove invalid traditional application code path" >&2; exit 1 ;;
esac
code_install_path="/www/wwwroot"
rm -rf -- "$code_install_path"/* "$code_install_path"/.[!.]* "$code_install_path"/..?*`

// traditionNginxVhostShell writes the rendered site-manager vhost into the
// nginx-dir subtree mounted from the embedded w7-traditiontool application.
const traditionNginxVhostShell = `{{- $rawDomain := toString .Values.DOMAIN_URL -}}
{{- $domain := replace "https://" "" $rawDomain -}}
{{- $domain = replace "http://" "" $domain -}}
{{- $domain = trimSuffix "/" $domain -}}
{{- $serverName := replace "," " " $domain -}}
{{- $primaryDomain := first (splitList " " (trimAll " " $serverName)) -}}
{{- $rootDir := print "/www/wwwroot/" $primaryDomain -}}
{{- $k8sDomain := print (include "common.fullname" .) "." .Release.Namespace ".svc.cluster.local" -}}
{{- $upstream := print .Release.Name "-" $primaryDomain | sha256sum | trunc 16 -}}
{{- $config := .Values.tradition.site.nginxVhostTemplate -}}
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

const traditionNginxVhostUninstallShell = `{{- $rawDomain := toString .Values.DOMAIN_URL -}}
{{- $domain := replace "https://" "" $rawDomain -}}
{{- $domain = replace "http://" "" $domain -}}
{{- $domain = trimSuffix "/" $domain -}}
{{- $serverName := replace "," " " $domain -}}
{{- $primaryDomain := first (splitList " " (trimAll " " $serverName)) -}}
set -eu
nginx_vhost_file={{ print $primaryDomain ".conf" | quote }}
rm -f -- "/www/server/nginx/conf.d/$nginx_vhost_file"`

func withTraditionAppImages(platform logic2.Platform) logic2.Platform {
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

func (hc *HelmPack) addTraditionAppValues(values map[string]interface{}) error {
	annotation := hc.Manifest.Application.Annotation
	if annotation == nil {
		annotation = map[string]interface{}{}
	}
	language := ""
	if value, ok := annotation[traditionImageLanguageAnnotation]; ok {
		language = strings.TrimSpace(traditionAnnotationString(value))
	}
	nginxVhostTemplate := ""
	if value, ok := annotation[traditionNginxVhostAnnotation]; ok {
		nginxVhostTemplate = strings.TrimSpace(traditionAnnotationString(value))
	}
	values["tradition"] = map[string]interface{}{
		"code": map[string]interface{}{
			"packageUrl": traditionAppCodePackageURL(hc.Manifest),
		},
		"site": map[string]interface{}{
			"language":           language,
			"nginxVhostTemplate": nginxVhostTemplate,
		},
	}
	applyTraditionPluginInstallerVolumeMount(values)
	if traditionGatewayEnabled(hc.Manifest) {
		applyTraditionToolJobVolumeMounts(values)
	}
	return nil
}

// applyTraditionPluginInstallerVolumeMount gives the application update task
// access to the PVC root where the per-domain OCI state is persisted.
func applyTraditionPluginInstallerVolumeMount(values map[string]interface{}) {
	jobs, ok := values["jobs"].([]map[string]interface{})
	if !ok {
		return
	}
	for _, job := range jobs {
		if job["title"] != traditionCodeInstallJobTitle {
			continue
		}
		container, ok := job["container"].(map[string]interface{})
		if !ok {
			continue
		}
		mounts, _ := container["volumeMounts"].([]v1.VolumeMount)
		container["volumeMounts"] = append(mounts, v1.VolumeMount{
			Name:      traditionStorageVolumeName,
			MountPath: pluginInstallerDataMount,
		})
	}
}

func traditionGatewayEnabled(manifest logic2.Manifest) bool {
	if manifest.Application.Annotation == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(traditionAnnotationString(
		manifest.Application.Annotation["w7.cc/nginx-gateway"],
	)), "true")
}

// applyTraditionToolJobVolumeMounts mounts the site-manager PVC location used
// by w7-traditiontool for NGINX configuration. The path matches
// w7panel-sitemanager/charts: /www/server/nginx backed by nginx-dir.
func applyTraditionToolJobVolumeMounts(values map[string]interface{}) {
	jobs, ok := values["jobs"].([]map[string]interface{})
	if !ok {
		return
	}
	for _, job := range jobs {
		if job["title"] != traditionNginxVhostJobTitle && job["title"] != traditionNginxVhostUninstallJobTitle {
			continue
		}
		if container, ok := job["container"].(map[string]interface{}); ok {
			container["volumeMounts"] = []v1.VolumeMount{{
				Name:      traditionStorageVolumeName,
				MountPath: "/www/server/nginx",
				SubPath:   "nginx-dir",
			}}
		}
	}
}

func (hc *HelmPack) traditionAppHelmValuesOptions() helmValuesOptions {
	options := hc.defaultHelmValuesOptions()
	// The traditional application editor persists storage, Sysbox, and gateway ingress
	// changes directly in the manifest. Helm packaging must consume that
	// contract instead of injecting a second, potentially conflicting shape.
	platform := hc.Manifest.Platform
	// Keep image placeholders dynamic while preserving the storage/runtime
	// contract already persisted by the traditional application editor.
	options.platform = withTraditionAppImages(platform)
	options.platform.Shells = append([]logic2.Shell(nil), options.platform.Shells...)
	if strings.TrimSpace(hc.Manifest.Source.Url) != "" {
		// Keep one pre-install,pre-upgrade hook rather than
		// separate install/upgrade jobs.
		options.platform.Shells = append(options.platform.Shells,
			logic2.Shell{
				Title: traditionCodeInstallJobTitle,
				Type:  "pre-install,pre-upgrade",
				Image: pluginInstallerImage,
				Shell: traditionCodeInstallShell,
			},
			logic2.Shell{
				Title: traditionCodeUninstallJobTitle,
				Type:  internalPostDeleteShellType,
				Image: managedCodeInstallShellImage,
				Shell: traditionCodeUninstallShell,
			},
		)
	}
	nginxVhostTemplate := ""
	if value, ok := hc.Manifest.Application.Annotation[traditionNginxVhostAnnotation]; ok {
		nginxVhostTemplate = strings.TrimSpace(traditionAnnotationString(value))
	}
	if nginxVhostTemplate != "" {
		options.platform.Shells = append(options.platform.Shells,
			logic2.Shell{
				Title: traditionNginxVhostJobTitle,
				Type:  "pre-install,pre-upgrade",
				Image: managedCodeInstallShellImage,
				Shell: traditionNginxVhostShell,
			},
			logic2.Shell{
				Title: traditionNginxVhostUninstallJobTitle,
				Type:  internalPostDeleteShellType,
				Image: managedCodeInstallShellImage,
				Shell: traditionNginxVhostUninstallShell,
			},
		)
	}
	options.addValues = hc.addTraditionAppValues
	return options
}

func (hc *HelmPack) packTraditionApp(rootDir, templatesDir string) error {
	hc.prepareTraditionAppSubManifests()
	return hc.packWorkloadApplication(
		rootDir,
		templatesDir,
		hc.traditionAppHelmValuesOptions(),
		true,
		nil,
	)
}

func (hc *HelmPack) prepareTraditionAppSubManifests() {
	for identify, child := range hc.SubManifest {
		hc.SubManifest[identify] = withTraditionToolRestartAnnotation(hc.Manifest, child)
	}
}

func withTraditionToolRestartAnnotation(parent, child logic2.Manifest) logic2.Manifest {
	if parent.Application.Type != logic2.TraditionApp {
		return child
	}
	identify := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(child.Application.Identifie), "_", "-"))
	if identify != "w7-traditiontool" {
		return child
	}
	annotations := make(map[string]interface{}, len(child.Application.Annotation)+1)
	for key, value := range child.Application.Annotation {
		annotations[key] = value
	}
	annotations[traditionToolRestartRevisionAnnotation] = "{{ .Release.Revision }}"
	child.Application.Annotation = annotations
	return child
}

func traditionAnnotationString(value interface{}) string {
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

func traditionAppCodePackageURL(manifest logic2.Manifest) string {
	if strings.TrimSpace(manifest.Source.Url) == "" {
		return ""
	}
	depot, _ := formulalogic.NewDepot()
	codePackageURL, _ := depot.GetFormulaBackendZipDownloadUrlByApplication(
		manifest.Application,
		strings.TrimPrefix(manifest.Source.Url, "file://"),
		false,
	)
	return codePackageURL
}
