package helm

import (
	"strings"

	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
)

func (hc *HelmPack) systemImageAppHelmValuesOptions() helmValuesOptions {
	options := hc.defaultHelmValuesOptions()
	options.platform = withSystemImageAppImages(options.platform)
	options.addValues = hc.addSystemImageAppValues
	return options
}

func (hc *HelmPack) addSystemImageAppValues(values map[string]interface{}) error {
	values["preserveRuntimeConfig"] = true
	return nil
}

func (hc *HelmPack) packSystemImageApp(rootDir, templatesDir string) error {
	return hc.packWorkloadApplication(
		rootDir,
		templatesDir,
		hc.systemImageAppHelmValuesOptions(),
		true,
		nil,
	)
}

// Keep the selected version dynamic without changing the source manifest.
func withSystemImageAppImages(platform logic2.Platform) logic2.Platform {
	platform.ContainerV2s = append([]logic2.ContainerV2(nil), platform.ContainerV2s...)
	for index := range platform.ContainerV2s {
		container := &platform.ContainerV2s[index]
		container.Image = strings.ReplaceAll(container.Image, "{version}", "{{ .Values.IMAGE_VERSION }}")
	}
	return platform
}
