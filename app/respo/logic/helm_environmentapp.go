package logic

import (
	"strings"

	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
	v1 "k8s.io/api/core/v1"
)

const (
	environmentAppCodeInstallerImage   = "busybox:1.36.1"
	environmentAppSiteManagerImage     = "zpk.w7.cc/public/site-manager:v1.3.5"
	environmentImageLanguageAnnotation = "w7.cc/image_language"
	environmentNginxVhostAnnotation    = "w7.cc/nginx_vhost_template"
)

func withEnvironmentAppStorage(platform logic2.Platform) logic2.Platform {
	platform.ContainerV2s = append([]logic2.ContainerV2(nil), platform.ContainerV2s...)
	for index := range platform.ContainerV2s {
		platform.ContainerV2s[index].VolumeMounts = append([]v1.VolumeMount(nil), platform.ContainerV2s[index].VolumeMounts...)
	}
	volumes := make([]v1.Volume, 0, len(platform.Volumes)+1)
	for _, volume := range platform.Volumes {
		if volume.Name != siteManagerStorageVolumeName {
			volumes = append(volumes, volume)
		}
	}
	volumes = append(volumes, v1.Volume{
		Name: siteManagerStorageVolumeName,
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
			if mount.Name != siteManagerStorageVolumeName {
				mounts = append(mounts, mount)
			}
		}
		mounts = append(mounts, v1.VolumeMount{
			Name: siteManagerStorageVolumeName, MountPath: `{{ print "/www/wwwroot/" .Values.DOMAIN_URL }}`,
			SubPath: `{{ print "nginx-web-dir/" .Values.DOMAIN_URL }}`,
		})
		platform.ContainerV2s[index].VolumeMounts = mounts
		break
	}
	return platform
}

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
	codeEnabled := strings.TrimSpace(hc.Manifest.Source.Url) != ""

	values["environment"] = map[string]interface{}{
		"code": map[string]interface{}{
			"enabled":    codeEnabled,
			"image":      environmentAppCodeInstallerImage,
			"packageUrl": environmentAppCodePackageURL(hc.Manifest),
			"volumeName": siteManagerStorageVolumeName,
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

func (hc *HelmPack) environmentAppHelmValuesOptions() helmValuesOptions {
	platform := withEnvironmentAppImages(withEnvironmentAppStorage(hc.Manifest.Platform))
	return helmValuesOptions{
		platform:                platform,
		workloadAffinity:        siteManagerPodAffinityValues(),
		jobAffinity:             siteManagerPodAffinityValues(),
		addValues:               hc.addEnvironmentAppValues,
		shellJobContainerValues: hc.getShellJobContainerValues,
	}
}

func (hc *HelmPack) packEnvironmentApp(rootDir, templatesDir string) error {
	// Environment applications use a dedicated Ingress routed through
	// site-manager nginx, so the standard application Ingress is disabled.
	return hc.packWorkloadApplication(
		rootDir,
		templatesDir,
		hc.environmentAppHelmValuesOptions(),
		false,
		hc.generateEnvironmentAppTemplates,
	)
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
