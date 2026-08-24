package logic

import (
	"fmt"
	"strings"

	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
	v1 "k8s.io/api/core/v1"
)

const traditionAppStorageVolumeName = "site-storage"

func withTraditionAppStorage(platform logic2.Platform) logic2.Platform {
	volumes := make([]v1.Volume, 0, len(platform.Volumes)+1)
	for _, volume := range platform.Volumes {
		if volume.Name != traditionAppStorageVolumeName {
			volumes = append(volumes, volume)
		}
	}
	platform.Volumes = append([]v1.Volume{{
		Name: traditionAppStorageVolumeName,
		VolumeSource: v1.VolumeSource{PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
			ClaimName: siteManagerPersistentVolumeClaim,
		}},
	}}, volumes...)
	return platform
}

func (hc *HelmPack) getTraditionShellJobContainerValues() map[string]interface{} {
	return map[string]interface{}{
		"name":  strings.ReplaceAll(hc.Manifest.Application.Identifie, "_", "-"),
		"image": imageValues(hc.getTraditionRuntimeImage(), v1.PullIfNotPresent), "env": []v1.EnvVar{},
		"resources":       v1.ResourceRequirements{},
		"volumeMounts":    []v1.VolumeMount{{Name: traditionAppStorageVolumeName, MountPath: "/www/wwwroot/{{ include \"tradition.codeInstallDirectory\" . }}", SubPath: "nginx-web-dir/{{ include \"tradition.codeInstallDirectory\" . }}"}},
		"securityContext": map[string]interface{}{},
	}
}

func (hc *HelmPack) addTraditionAppValues(values map[string]interface{}) error {
	if strings.TrimSpace(hc.Manifest.Platform.Tradition.EnvironmentName) == "" {
		return fmt.Errorf("传统应用必须指定环境应用")
	}
	if strings.TrimSpace(hc.Manifest.Platform.Tradition.EnvironmentVersion) == "" {
		return fmt.Errorf("传统应用必须指定环境应用版本")
	}
	if strings.TrimSpace(hc.Manifest.Source.Url) == "" {
		return fmt.Errorf("传统应用必须配置代码包")
	}
	codePackageURL := ""
	depot, _ := NewDepot()
	codePackageURL, _ = depot.GetFormulaBackendZipDownloadUrlByApplication(
		hc.Manifest.Application,
		strings.TrimPrefix(hc.Manifest.Source.Url, "file://"),
		false,
	)

	values["tradition"] = map[string]interface{}{
		"codePackageUrl":       codePackageURL,
		"affinity":             environmentAppPodAffinityValues(),
		"environmentIdentifie": hc.Manifest.Platform.Tradition.EnvironmentName,
	}
	return nil
}

func (hc *HelmPack) generateTraditionAppTemplates(rootDir string) error {
	if err := writeHelmTemplateFile(rootDir, "_tradition.tpl", "tradition-helpers.tpl"); err != nil {
		return err
	}
	if err := writeHelmTemplateFile(rootDir, "install-code-job.yaml", "tradition-install-code-job.yaml.tpl"); err != nil {
		return err
	}
	return writeHelmTemplateFile(rootDir, "shell-job.yaml", "shell-job.yaml.tpl")
}

func (hc *HelmPack) getTraditionAppShells() []logic2.Shell {
	shells := append([]logic2.Shell(nil), hc.Manifest.Platform.Shells...)
	if image := hc.getTraditionRuntimeImage(); image != "" {
		for index := range shells {
			if strings.TrimSpace(shells[index].Image) == "" {
				shells[index].Image = image
			}
		}
	}
	return shells
}

func (hc *HelmPack) getTraditionRuntimeImage() string {
	if hc.Manifest.Application.Type != logic2.Tradition_App {
		return ""
	}
	imageTemplate := strings.TrimSpace(hc.Manifest.Platform.Tradition.EnvironmentImageTemplate)
	if imageTemplate == "" {
		return ""
	}
	return strings.ReplaceAll(imageTemplate, "{version}", hc.Manifest.Platform.Tradition.EnvironmentVersion)
}
