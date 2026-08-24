package logic

import (
	"fmt"
	"strings"

	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
	v1 "k8s.io/api/core/v1"
)

const (
	siteManagerPersistentVolumeClaim   = "w7-sitemanager-site-manager"
	environmentAppStorageVolumeName    = "site-storage"
	environmentAppCodeInstallerImage   = "busybox:1.36.1"
	environmentAppSiteManagerImage     = "zpk.w7.cc/public/site-manager:v1.3.3"
	environmentImageLanguageAnnotation = "w7.cc/image_language"
	environmentNginxVhostAnnotation    = "w7.cc/nginx_vhost_template"
)

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

func withEnvironmentAppCodeStorage(platform logic2.Platform) logic2.Platform {
	volumes := make([]v1.Volume, 0, len(platform.Volumes)+1)
	for _, volume := range platform.Volumes {
		if volume.Name != environmentAppStorageVolumeName {
			volumes = append(volumes, volume)
		}
	}
	volumes = append(volumes, v1.Volume{
		Name: environmentAppStorageVolumeName,
		VolumeSource: v1.VolumeSource{PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
			ClaimName: siteManagerPersistentVolumeClaim,
		}},
	})
	platform.Volumes = volumes
	for index := range platform.ContainerV2s {
		if platform.ContainerV2s[index].IsInitContainer {
			continue
		}
		mounts := make([]v1.VolumeMount, 0, len(platform.ContainerV2s[index].VolumeMounts)+1)
		for _, mount := range platform.ContainerV2s[index].VolumeMounts {
			if mount.Name != environmentAppStorageVolumeName {
				mounts = append(mounts, mount)
			}
		}
		mounts = append(mounts, v1.VolumeMount{
			Name: environmentAppStorageVolumeName, MountPath: `{{ print "/www/wwwroot/" .Values.DOMAIN_URL }}`,
			SubPath: `{{ print "nginx-web-dir/" .Values.DOMAIN_URL }}`,
		})
		platform.ContainerV2s[index].VolumeMounts = mounts
		break
	}
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

func (hc *HelmPack) addEnvironmentAppValues(values map[string]interface{}) error {
	volumeName := environmentAppStorageVolumeName
	for _, volume := range hc.Manifest.Platform.Volumes {
		if volume.PersistentVolumeClaim != nil {
			volumeName = volume.Name
			break
		}
	}
	codeEnabled := strings.TrimSpace(hc.Manifest.Source.Url) != ""
	if codeEnabled && volumeName == "" {
		return fmt.Errorf("环境应用必须配置站点存储卷")
	}

	values["environment"] = map[string]interface{}{
		"code": map[string]interface{}{
			"enabled":    codeEnabled,
			"image":      environmentAppCodeInstallerImage,
			"packageUrl": environmentAppCodePackageURL(hc.Manifest),
			"volumeName": volumeName,
		},
		"site": map[string]interface{}{
			"image":              environmentAppSiteManagerImage,
			"title":              environmentAppTitle(hc.Manifest),
			"group":              strings.ReplaceAll(hc.Manifest.Application.Identifie, "_", "-"),
			"language":           environmentAppAnnotation(hc.Manifest, environmentImageLanguageAnnotation),
			"nginxVhostTemplate": environmentAppAnnotation(hc.Manifest, environmentNginxVhostAnnotation),
		},
	}
	return nil
}

func (hc *HelmPack) generateEnvironmentAppTemplates(rootDir string) error {
	if strings.TrimSpace(hc.Manifest.Source.Url) != "" {
		if err := writeHelmTemplateFile(rootDir, "environment-install-code-job.yaml", "environment-install-code-job.yaml.tpl"); err != nil {
			return err
		}
	}
	if err := writeHelmTemplateFile(rootDir, "environment-create-site-job.yaml", "environment-create-site-job.yaml.tpl"); err != nil {
		return err
	}
	return writeHelmTemplateFile(rootDir, "environment-ingress.yaml", "environment-ingress.yaml.tpl")
}

func environmentAppTitle(manifest logic2.Manifest) string {
	title := strings.TrimSpace(manifest.Application.Name)
	if title == "" {
		title = strings.TrimSpace(manifest.Platform.BaseInfo.Name)
	}
	if title == "" {
		title = manifest.Application.Identifie
	}
	return title
}

func environmentAppAnnotation(manifest logic2.Manifest, key string) string {
	if value, ok := manifest.Application.Annotation[key].(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
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
