package logic

import (
	"fmt"
	"strings"

	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
)

const (
	traditionInstallTypeSite      = "site"
	traditionInstallTypeExtension = "extension"
)

func normalizeTraditionInstall(tradition logic2.Tradition) (string, error) {
	installType := strings.TrimSpace(tradition.InstallType)
	if installType == "" {
		installType = traditionInstallTypeSite
	}
	if installType != traditionInstallTypeSite && installType != traditionInstallTypeExtension {
		return "", fmt.Errorf("不支持的传统应用安装类型: %s", installType)
	}
	return installType, nil
}

func (hc *HelmPack) addTraditionAppValues(values map[string]interface{}) error {
	if strings.TrimSpace(hc.Manifest.Platform.Tradition.EnvironmentName) == "" {
		return fmt.Errorf("传统应用必须指定环境应用")
	}
	if strings.TrimSpace(hc.Manifest.Platform.Tradition.EnvironmentVersion) == "" {
		return fmt.Errorf("传统应用必须指定环境应用版本")
	}
	if _, err := normalizeTraditionInstall(hc.Manifest.Platform.Tradition); err != nil {
		return err
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
		"codePackageUrl": codePackageURL,
		"affinity":       environmentAppPodAffinityValues(),
	}
	return nil
}

func (hc *HelmPack) generateTraditionAppTemplates(rootDir string) error {
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
	if hc.Manifest.Application.Type != logic2.Tradition_App || hc.SubManifest == nil {
		return ""
	}
	environmentName := hc.Manifest.Platform.Tradition.EnvironmentName
	environment, ok := hc.SubManifest[environmentName]
	if !ok {
		return ""
	}
	for _, container := range environment.Platform.ContainerV2s {
		if container.IsInitContainer || strings.TrimSpace(container.Image) == "" {
			continue
		}
		return strings.ReplaceAll(container.Image, "{version}", hc.Manifest.Platform.Tradition.EnvironmentVersion)
	}
	return ""
}

func traditionRuntimeImageValues(image string) map[string]interface{} {
	repository, tag := image, "latest"
	if index := strings.LastIndex(image, ":"); index > strings.LastIndex(image, "/") {
		repository, tag = image[:index], image[index+1:]
	}
	return map[string]interface{}{"repository": repository, "tag": tag, "pullPolicy": "IfNotPresent"}
}
