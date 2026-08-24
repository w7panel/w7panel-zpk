package logic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	copy2 "github.com/otiai10/copy"
	"github.com/w7panel/w7panel-zpk/common/function"
	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

type Dependency struct {
	Alias        string   `yaml:"alias,omitempty" json:"alias,omitempty"`
	Condition    string   `yaml:"condition,omitempty" json:"condition,omitempty"`
	ImportValues []string `yaml:"import-values,omitempty" json:"importValues,omitempty"`
	Name         string   `yaml:"name" json:"name"`
	Repository   string   `yaml:"repository,omitempty" json:"repository,omitempty"`
	Tags         []string `yaml:"tags,omitempty" json:"tags,omitempty"`
	Version      string   `yaml:"version" json:"version"`
}
type Maintainer struct {
	Name  string `yaml:"name" json:"name"`
	Email string `yaml:"email,omitempty" json:"email,omitempty"`
	URL   string `yaml:"url,omitempty" json:"url"`
}

// ChartYAML 结构体
type ChartYAML struct {
	APIVersion   string            `yaml:"apiVersion" json:"apiVersion"`
	Name         string            `yaml:"name" json:"name"`
	Version      string            `yaml:"version" json:"version"`
	Description  string            `yaml:"description,omitempty" json:"description"`
	Type         string            `yaml:"type,omitempty" json:"type"`
	Condition    string            `yaml:"condition,omitempty" json:"condition,omitempty"`
	Keywords     []string          `yaml:"keywords,omitempty" json:"keywords"`
	Home         string            `yaml:"home,omitempty" json:"home"`
	Sources      []string          `yaml:"sources,omitempty" json:"sources"`
	Maintainers  []Maintainer      `yaml:"maintainers,omitempty" json:"maintainers"`
	Icon         string            `yaml:"icon,omitempty" json:"icon"`
	AppVersion   string            `yaml:"appVersion,omitempty" json:"appVersion"`
	Deprecated   bool              `yaml:"deprecated,omitempty" json:"deprecated"`
	Annotations  map[string]string `yaml:"annotations,omitempty" json:"annotations"`
	KubeVersion  string            `yaml:"kubeVersion,omitempty" json:"kubeVersion"`
	Tags         []string          `yaml:"tags,omitempty" json:"tags,omitempty"`
	Dependencies []Dependency      `yaml:"dependencies,omitempty" json:"dependencies"`
}

type HelmPack struct {
	Manifest               logic2.Manifest
	SubManifest            map[string]logic2.Manifest
	IngressNames           map[string]string
	ShellNames             []string
	OutputDir              string
	ChartVersion           string
	IsSubFormula           bool
	SharedStorageTargetApp string
	Sidecars               []HelmSidecar
}

func NewHelmPack(manifest logic2.Manifest, subManifests []*logic2.Manifest, outputDir, chartVersion string, isSubFormula bool, sharedStorageTargetApp string) *HelmPack {
	ApplyDependencyReleaseStartParams(&manifest)
	subManifestMap := make(map[string]logic2.Manifest)
	if subManifests != nil {
		for _, item := range subManifests {
			if item.Application.Identifie != manifest.Application.Identifie {
				subManifestMap[item.Application.Identifie] = *item
			}
		}
	}

	return &HelmPack{
		Manifest:               manifest,
		SubManifest:            subManifestMap,
		OutputDir:              outputDir,
		ChartVersion:           chartVersion,
		IngressNames:           make(map[string]string),
		ShellNames:             []string{},
		IsSubFormula:           isSubFormula,
		SharedStorageTargetApp: sharedStorageTargetApp,
	}
}

func PackManifestToHelm(manifest logic2.Manifest, subManifest []*logic2.Manifest, outputDir string, isSubFormula bool, sharedStorageTargetApp string) error {
	// 使用 Chart 版本或默认值
	chartVersion := "0.1.0"
	if manifest.Platform.Helm.Version != "" {
		chartVersion = manifest.Platform.Helm.Version
	}

	packer := NewHelmPack(manifest, subManifest, outputDir, chartVersion, isSubFormula, sharedStorageTargetApp)
	return packer.PackToHelm()
}

func PackFormulaToHelmAndPack(formula Formula, rePack bool) (string, error) {
	depot, _ := NewDepot()
	helmDir := filepath.Join(filepath.Join(depot.GetBasePath(), "Helm", "Formula"))
	helmZipPath := filepath.Join(filepath.Dir(helmDir), strings.ReplaceAll(formula.Manifest.Application.Identifie, "-", "_")+"-"+strconv.Itoa(int(formula.VersionId))+".tgz")
	if !rePack && function.FileExists(helmZipPath) {
		return helmZipPath, nil
	}

	err := PackManifestToHelm(*formula.Manifest, formula.AllManifest, helmDir, false, "")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(helmDir)

	formulaHelmDir := filepath.Join(helmDir, formula.Manifest.Application.Identifie)
	err = function.ZipHelmChart(formulaHelmDir, helmZipPath)
	return helmZipPath, err
}

func (hc *HelmPack) PackToHelm() error {
	// 2. 创建主 Chart 目录结构
	helmDir := filepath.Join(hc.OutputDir, hc.Manifest.Application.Identifie)
	templatesDir := filepath.Join(helmDir, "templates")
	chartsDir := filepath.Join(helmDir, "charts")
	filesDir := filepath.Join(helmDir, "files")
	function.CreateDirIfNotExist(templatesDir, os.ModePerm)
	function.CreateDirIfNotExist(chartsDir, os.ModePerm)
	function.CreateDirIfNotExist(filesDir, os.ModePerm)

	if err := hc.generateSubCharts(chartsDir); err != nil {
		return err
	}
	if err := hc.prepareSidecarCharts(chartsDir); err != nil {
		return err
	}

	if hc.Manifest.Application.Type == logic2.GatewayPluginApp {
		renderer, err := getGatewayPluginRenderer(hc.Manifest.Platform.GatewayPlugin)
		if err != nil {
			return err
		}
		if err := hc.generateGatewayPluginValuesYaml(helmDir, renderer); err != nil {
			return err
		}
		if err := hc.generateGatewayPluginTemplate(templatesDir, renderer); err != nil {
			return err
		}
	} else if hc.isHelmPackage() {
		err := hc.processHelmPkg(helmDir)
		if err != nil {
			return err
		}
		if err := hc.configureHelmSidecarHost(helmDir); err != nil {
			return err
		}
	} else if hc.Manifest.Application.Type == logic2.Tradition_App {
		if err := hc.generateValuesYaml(helmDir); err != nil {
			return err
		}
		if err := hc.generateHelpersTpl(templatesDir, hc.Manifest.Application.Identifie); err != nil {
			return err
		}
		if err := hc.generateTraditionAppTemplates(templatesDir); err != nil {
			return err
		}
		if err := hc.generateShellsTemplates(templatesDir); err != nil {
			return err
		}
	} else {
		if err := hc.generateValuesYaml(helmDir); err != nil {
			return err
		}
		if err := hc.generateHelpersTpl(templatesDir, hc.Manifest.Application.Identifie); err != nil {
			return err
		}

		if err := hc.generateWorkloadYaml(templatesDir); err != nil {
			return err
		}
		if err := hc.generateShellsTemplates(templatesDir); err != nil {
			return err
		}
		if hc.Manifest.Application.Type == logic2.EnvironmentApp {
			if err := hc.generateEnvironmentAppTemplates(templatesDir); err != nil {
				return err
			}
		}
		if err := hc.generateBuildImageJobTemplate(templatesDir); err != nil {
			return err
		}
		if err := hc.generateServiceYaml(templatesDir); err != nil {
			return err
		}
		if err := hc.generateNodePortServiceYaml(templatesDir); err != nil {
			return err
		}
		if err := hc.generateIngressesYaml(templatesDir); err != nil {
			return err
		}
		if hc.Manifest.Application.ClusterPrivileged {
			if err := hc.generateServiceAccountYaml(templatesDir); err != nil {
				return err
			}
			if err := hc.generateClusterRoleYaml(templatesDir); err != nil {
				return err
			}
			if err := hc.generateClusterRoleBindingYaml(templatesDir); err != nil {
				return err
			}
			if err := hc.generateSecretYaml(templatesDir); err != nil {
				return err
			}
		}
	}

	if err := hc.generateSidecarHelpersTemplate(templatesDir); err != nil {
		return err
	}
	if err := hc.generateSidecarResourcesTemplate(templatesDir); err != nil {
		return err
	}

	if !hc.IsSubFormula {
		if shouldPackageMicroApp(hc.Manifest) {
			if err := hc.generateMicroAppTemplate(templatesDir, hc.Manifest); err != nil {
				return err
			}
		}
		if shouldGenerateRegisterSite(hc.Manifest) {
			if err := hc.generateRegisterSiteJobTemplate(templatesDir, hc.Manifest); err != nil {
				return err
			}
		}
	}

	if err := hc.generateChartYaml(helmDir); err != nil {
		return err
	}

	return nil
}

func shouldGenerateRegisterSite(manifest logic2.Manifest) bool {
	return manifest.Application.Type != logic2.GatewayPluginApp && manifest.Application.RegisterSite
}

func (hc *HelmPack) isHelmPackage() bool {
	return hc.Manifest.Platform.Helm.ChartName != "" ||
		hc.Manifest.Platform.Helm.Repository != "" ||
		len(hc.Manifest.Platform.Helm.DependYamls) > 0
}

func (hc *HelmPack) processHelmPkg(rootDir string) error {
	dependHelmYamlInHelm := false
	if hc.Manifest.Platform.Helm.ChartName != "" || hc.Manifest.Platform.Helm.Repository != "" {
		depot, _ := NewDepot()
		helmLocalRelativePath := depot.GetHelmLocalRelativePath(hc.Manifest.Platform.Helm)
		localHelmZipPath := ""
		if helmLocalRelativePath != "" {
			localHelmZipPath = filepath.Join(rootDir, path.Base(helmLocalRelativePath))
			err := copy2.Copy(filepath.Join(depot.GetBasePath(), helmLocalRelativePath), localHelmZipPath)
			if err != nil {
				return err
			}
		} else {
			if hc.Manifest.Platform.Helm.Repository != "" {
				localChartPath, err := HelmRepository{}.DownloadChart(hc.Manifest.Platform.Helm.Repository, hc.Manifest.Platform.Helm.ChartName, hc.Manifest.Platform.Helm.Version, rootDir)
				if err != nil {
					return err
				}
				localHelmZipPath = localChartPath
			} else {
				helmDownloadUrl := hc.Manifest.Platform.Helm.ChartName
				urlInfo, err := url.Parse(helmDownloadUrl)
				if err != nil {
					return err
				}
				localHelmZipPath = filepath.Join(rootDir, path.Base(urlInfo.Path))
				err = function.DownloadFile(context.Background(), helmDownloadUrl, localHelmZipPath)
				if err != nil {
					return err
				}
			}
		}
		defer os.Remove(localHelmZipPath)
		err := function.UnzipHelmPackage(localHelmZipPath, rootDir)
		if err != nil {
			return err
		}

		if err = hc.applyHelmConfigOverrides(rootDir); err != nil {
			return err
		}

		dependHelmYamlInHelm = true
	}

	if hc.Manifest.Platform.Helm.DependYamls != nil && len(hc.Manifest.Platform.Helm.DependYamls) > 0 {
		dependHelmTemplatesDir := filepath.Join(rootDir, "templates")

		for _, item := range hc.Manifest.Platform.Helm.DependYamls {
			yamlPath := filepath.Join(dependHelmTemplatesDir, item.Name)
			function.CreateDirIfNotExist(filepath.Dir(yamlPath), os.ModePerm)
			err := writeFile(yamlPath, item.Yaml)
			if err != nil {
				return err
			}
		}

		if !dependHelmYamlInHelm {
			chartYaml := ChartYAML{
				APIVersion:  "v2",
				Name:        strings.ReplaceAll(hc.Manifest.Application.Identifie, "-", "_"),
				Version:     hc.ChartVersion,
				Description: hc.Manifest.Application.Identifie,
				Type:        "application",
				AppVersion:  "1.16.0",
			}

			filePath := filepath.Join(rootDir, "Chart.yaml")
			err := writeYAMLFile(filePath, chartYaml)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (hc *HelmPack) applyHelmConfigOverrides(rootDir string) error {
	if len(hc.Manifest.Platform.Helm.Configs) == 0 {
		return nil
	}

	overrides, err := buildHelmConfigOverrides(hc.Manifest.Platform.Helm.Configs)
	if err != nil {
		return err
	}
	if len(overrides) == 0 {
		return nil
	}

	valuesPath := filepath.Join(rootDir, "values.yaml")
	values := make(map[string]interface{})
	if function.FileExists(valuesPath) {
		data, err := os.ReadFile(valuesPath)
		if err != nil {
			return fmt.Errorf("读取 Helm values.yaml 失败: %w", err)
		}
		if len(bytes.TrimSpace(data)) > 0 {
			if err := yaml.Unmarshal(data, &values); err != nil {
				return fmt.Errorf("解析 Helm values.yaml 失败: %w", err)
			}
		}
	}

	mergeValuesMap(values, overrides)
	return writeYAMLFile(valuesPath, values)
}

func buildHelmConfigOverrides(configs []interface{}) (map[string]interface{}, error) {
	overrides := make(map[string]interface{})
	for _, item := range configs {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("helm kv 配置格式错误: %T", item)
		}

		val, exists := itemMap["name"]
		if !exists {
			continue
		}
		valStr, ok := val.(string)
		if !ok {
			return nil, fmt.Errorf("helm kv name 必须是 string: %T", val)
		}
		valStr = strings.TrimSpace(valStr)
		if valStr == "" {
			continue
		}

		rawValue, exists := itemMap["value"]
		if !exists {
			continue
		}
		rawValueStr, ok := rawValue.(string)
		if !ok {
			return nil, fmt.Errorf("helm kv value 必须是 string: %T", rawValue)
		}
		setNestedValue(overrides, valStr, parseHelmConfigValue(rawValueStr))
	}

	return overrides, nil
}

func parseHelmConfigValue(raw string) interface{} {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return raw
	}

	var value interface{}
	if err := yaml.Unmarshal([]byte(trimmed), &value); err == nil {
		return value
	}

	return raw
}

func setNestedValue(target map[string]interface{}, path string, value interface{}) {
	parts, err := parseHelmConfigPath(path)
	if err != nil || len(parts) == 0 {
		return
	}

	setNestedValueOnMap(target, parts, value)
}

func parseHelmConfigPath(path string) ([]string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("empty path")
	}

	parts := make([]string, 0)
	var current strings.Builder
	inQuotes := false
	for i := 0; i < len(path); i++ {
		ch := path[i]
		if inQuotes {
			if ch == '"' {
				inQuotes = false
				continue
			}
			current.WriteByte(ch)
			continue
		}

		switch ch {
		case '"':
			inQuotes = true
		case '.':
			part := strings.TrimSpace(current.String())
			if part == "" {
				// Allow dots immediately after an array index, e.g. e.f[0].g
				if i > 0 && path[i-1] == ']' {
					continue
				}
				return nil, fmt.Errorf("invalid path: %s", path)
			}
			parts = append(parts, part)
			current.Reset()
		case '[':
			part := strings.TrimSpace(current.String())
			if part != "" {
				parts = append(parts, part)
				current.Reset()
			}

			end := strings.IndexByte(path[i:], ']')
			if end <= 1 {
				return nil, fmt.Errorf("invalid path: %s", path)
			}
			indexPart := strings.TrimSpace(path[i+1 : i+end])
			if !isNumericPathPart(indexPart) {
				return nil, fmt.Errorf("invalid array index: %s", indexPart)
			}
			parts = append(parts, indexPart)
			i += end
		default:
			current.WriteByte(ch)
		}
	}
	if inQuotes {
		return nil, fmt.Errorf("unterminated quote in path: %s", path)
	}

	lastPart := strings.TrimSpace(current.String())
	if lastPart != "" {
		parts = append(parts, lastPart)
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid path: %s", path)
	}

	return parts, nil
}

func setNestedValueOnMap(target map[string]interface{}, parts []string, value interface{}) {
	part := parts[0]
	if len(parts) == 1 {
		target[part] = value
		return
	}

	next := target[part]
	if isNumericPathPart(parts[1]) {
		child, _ := next.([]interface{})
		target[part] = setNestedValueOnSlice(child, parts[1:], value)
		return
	}

	child, ok := next.(map[string]interface{})
	if !ok {
		child = make(map[string]interface{})
	}
	setNestedValueOnMap(child, parts[1:], value)
	target[part] = child
}

func setNestedValueOnSlice(target []interface{}, parts []string, value interface{}) []interface{} {
	index, err := strconv.Atoi(parts[0])
	if err != nil || index < 0 {
		return target
	}

	if len(target) <= index {
		target = append(target, make([]interface{}, index-len(target)+1)...)
	}

	if len(parts) == 1 {
		target[index] = value
		return target
	}

	if isNumericPathPart(parts[1]) {
		child, _ := target[index].([]interface{})
		target[index] = setNestedValueOnSlice(child, parts[1:], value)
		return target
	}

	child, ok := target[index].(map[string]interface{})
	if !ok {
		child = make(map[string]interface{})
	}
	setNestedValueOnMap(child, parts[1:], value)
	target[index] = child
	return target
}

func isNumericPathPart(part string) bool {
	if part == "" {
		return false
	}
	_, err := strconv.Atoi(part)
	return err == nil
}

func mergeValuesMap(dst, src map[string]interface{}) {
	for key, value := range src {
		dst[key] = mergeValuesValue(dst[key], value)
	}
}

func mergeValuesValue(dst, src interface{}) interface{} {
	srcMap, srcIsMap := src.(map[string]interface{})
	if srcIsMap {
		dstMap, ok := dst.(map[string]interface{})
		if !ok {
			dstMap = make(map[string]interface{})
		}
		mergeValuesMap(dstMap, srcMap)
		return dstMap
	}

	srcSlice, srcIsSlice := src.([]interface{})
	if srcIsSlice {
		dstSlice, ok := dst.([]interface{})
		if !ok {
			dstSlice = make([]interface{}, 0)
		}
		if len(dstSlice) < len(srcSlice) {
			dstSlice = append(dstSlice, make([]interface{}, len(srcSlice)-len(dstSlice))...)
		}
		for index, item := range srcSlice {
			if item == nil {
				continue
			}
			dstSlice[index] = mergeValuesValue(dstSlice[index], item)
		}
		return dstSlice
	}

	return src
}

// ==================== 子 Chart 生成 ====================

// generateSubCharts 生成所有子应用的 Chart
func (hc *HelmPack) generateSubCharts(rootDir string) error {
	if hc.SubManifest == nil {
		return nil
	}
	mainChartName := hc.Manifest.Application.Identifie
	for _, subManifest := range hc.SubManifest {
		if hc.Manifest.Application.Type == logic2.Tradition_App &&
			subManifest.Application.Identifie == hc.Manifest.Platform.Tradition.EnvironmentName {
			continue
		}
		sharedStorageTargetApp := ""
		if hasSharedPersistentStorage(hc.Manifest.Platform.Volumes, subManifest.Platform.Volumes) {
			sharedStorageTargetApp = mainChartName
		}
		err := PackManifestToHelm(subManifest, nil, rootDir, true, sharedStorageTargetApp)
		if err != nil {
			return err
		}
	}

	return nil
}

// generateChartYaml 生成 Chart.yaml
func (hc *HelmPack) generateChartYaml(rootDir string) error {
	dependencies := make([]Dependency, 0)
	parseChartFromDir := func(chartDir string) (*Dependency, error) {
		chartYamlPath := filepath.Join(chartDir, "Chart.yaml")
		if !function.FileExists(chartYamlPath) {
			return nil, fmt.Errorf("chart.yaml不存在: %s", chartDir)
		}

		data, err := os.ReadFile(chartYamlPath)
		if err != nil {
			return nil, fmt.Errorf("读取Chart.yaml失败: %v", err)
		}
		var chart ChartYAML
		if err := yaml.Unmarshal(data, &chart); err != nil {
			return nil, fmt.Errorf("解析Chart.yaml失败: %v", err)
		}

		return &Dependency{
			Name:       chart.Name,
			Version:    chart.Version,
			Repository: "", // 本地目录没有repository
		}, nil
	}

	chartsDir := filepath.Join(rootDir, "charts")
	if !function.IsDirEmpty(chartsDir) {
		entries, err := os.ReadDir(chartsDir)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if entry.IsDir() {
				dep, err := parseChartFromDir(filepath.Join(chartsDir, entry.Name()))
				if err == nil && dep.Name != "" {
					dep.Repository = "file://./charts/" + entry.Name()
					dependencies = append(dependencies, *dep)
				}
			}
		}
	}

	chartYaml := ChartYAML{}
	chartYamlPath := filepath.Join(rootDir, "Chart.yaml")
	if function.FileExists(chartYamlPath) {
		data, err := os.ReadFile(chartYamlPath)
		if err != nil {
			return err
		}
		if err := yaml.Unmarshal(data, &chartYaml); err != nil {
			return err
		}
	} else {
		chartYaml = ChartYAML{
			APIVersion:  "v2",
			Name:        hc.Manifest.Application.Identifie,
			Version:     hc.ChartVersion,
			Description: hc.Manifest.Application.Description,
			Type:        "application",
			AppVersion:  "1.16.0",
		}
	}

	if len(dependencies) > 0 {
		mergedDependencies := append([]Dependency(nil), chartYaml.Dependencies...)
		indexByName := make(map[string]int, len(mergedDependencies))
		indexByRepo := make(map[string]int, len(mergedDependencies))

		for i, dep := range mergedDependencies {
			indexByName[dep.Name] = i
			if dep.Repository != "" {
				indexByRepo[dep.Repository] = i
			}
		}

		for _, dep := range dependencies {
			targetIndex := -1
			if idx, ok := indexByName[dep.Name]; ok {
				targetIndex = idx
			} else if idx, ok := indexByRepo[dep.Repository]; ok {
				targetIndex = idx
			}

			if targetIndex >= 0 {
				merged := mergedDependencies[targetIndex]
				merged.Name = dep.Name
				merged.Version = dep.Version
				merged.Repository = dep.Repository
				mergedDependencies[targetIndex] = merged
				indexByName[merged.Name] = targetIndex
				if merged.Repository != "" {
					indexByRepo[merged.Repository] = targetIndex
				}
				continue
			}

			mergedDependencies = append(mergedDependencies, dep)
			targetIndex = len(mergedDependencies) - 1
			indexByName[dep.Name] = targetIndex
			if dep.Repository != "" {
				indexByRepo[dep.Repository] = targetIndex
			}
		}

		chartYaml.Dependencies = mergedDependencies
	}

	filePath := filepath.Join(rootDir, "Chart.yaml")
	return writeYAMLFile(filePath, chartYaml)
}

// generateHelpersTpl 生成 _helpers.tpl
func (hc *HelmPack) generateHelpersTpl(rootDir string, identify string) error {
	helpersTemplate, err := loadHelmTemplate("_helpers.tpl")
	if err != nil {
		return err
	}
	helpersTemplate = renderHelmTemplatePlaceholders(helpersTemplate, map[string]string{
		"__IDENTIFY__": identify,
	})
	return writeFile(filepath.Join(rootDir, "_helpers.tpl"), helpersTemplate)
}

// generateValuesYaml 生成 values.yaml
func (hc *HelmPack) generateValuesYaml(rootDir string) error {
	platform := hc.Manifest.Platform
	if hc.Manifest.Application.Type == logic2.EnvironmentApp {
		platform = withEnvironmentAppStorage(platform)
	} else if hc.Manifest.Application.Type == logic2.Tradition_App {
		platform = withTraditionAppStorage(platform)
	}
	platform.Shells = hc.getHelmShells()
	if hc.Manifest.Application.Annotation == nil {
		hc.Manifest.Application.Annotation = make(map[string]interface{})
	}
	defaultPort := 0
	for _, item := range hc.Manifest.Platform.ContainerV2s {
		if defaultPort != 0 {
			break
		}
		if len(item.Ports) != 0 {
			defaultPort = int(item.Ports[0].ContainerPort)
		}
	}

	appName := hc.Manifest.Platform.BaseInfo.Name
	if appName == "" {
		appName = hc.Manifest.Platform.BaseInfo.Identifie
	}
	values := map[string]interface{}{
		"PVC_NAME":   "",
		"DOMAIN_URL": "",
		"app": map[string]interface{}{
			"title":    appName,
			"identify": hc.Manifest.Application.Identifie,
		},
		"annotations":          hc.Manifest.Application.Annotation,
		"global":               hc.getGlobalValues(),
		"replicas":             1,
		"workload":             hc.getWorkloadValues(platform),
		"service":              hc.getServiceValues(platform),
		"node_service":         hc.getNodePortServiceValues(platform),
		"ingress":              hc.getIngressValues(platform, hc.Manifest.Application),
		"serviceAccount":       hc.getServiceAccountValues(hc.Manifest.Application),
		"volumes":              platform.Volumes,
		"volumeClaimTemplates": hc.getVolumeClaimTemplateValues(hc.Manifest.Platform),
		"defaultPort":          defaultPort,
		"startParams":          hc.getStartParamsEnvValues(hc.Manifest.Platform),
		"runtimeClass":         hc.getRuntimeClassValues(hc.Manifest.Platform),
		"hostUsers":            hc.Manifest.Platform.HostUsers,
		"affinity":             hc.getWorkloadAffinityValues(),
		"jobAffinity":          hc.getJobAffinityValues(),
		"w7panelSidecars":      hc.Sidecars,
	}
	values["jobs"] = hc.buildJobValues(platform)
	if hc.Manifest.Application.Type == logic2.EnvironmentApp {
		if err := hc.addEnvironmentAppValues(values); err != nil {
			return err
		}
	}
	if hc.Manifest.Application.Type == logic2.Tradition_App {
		if err := hc.addTraditionAppValues(values); err != nil {
			return err
		}
	}
	helmContainers := make([]map[string]interface{}, 0, len(platform.ContainerV2s))
	for _, container := range platform.ContainerV2s {
		helmContainers = append(helmContainers, hc.generateContainerV2Values(container, hc.Manifest.Application))
	}
	values["containers"] = helmContainers

	for _, item := range hc.Manifest.Platform.StartParams {
		if len(strings.Split(item.Name, ".")) > 1 {
			continue
		}
		values[item.Name] = item.ValuesText
	}

	filePath := filepath.Join(rootDir, "values.yaml")
	return writeYAMLFile(filePath, values)
}

func (hc *HelmPack) getWorkloadAffinityValues() map[string]interface{} {
	if hc.Manifest.Application.Type == logic2.EnvironmentApp {
		return environmentAppPodAffinityValues()
	}
	if hc.SharedStorageTargetApp == "" {
		return nil
	}
	return podAffinityByIdentify(hc.SharedStorageTargetApp)
}

func (hc *HelmPack) getJobAffinityValues() map[string]interface{} {
	if hc.Manifest.Application.Type == logic2.EnvironmentApp {
		return environmentAppPodAffinityValues()
	}
	return podAffinityByIdentify(hc.Manifest.Application.Identifie)
}

func podAffinityByIdentify(identify string) map[string]interface{} {
	return map[string]interface{}{"podAffinity": map[string]interface{}{
		"requiredDuringSchedulingIgnoredDuringExecution": []interface{}{map[string]interface{}{
			"labelSelector": map[string]interface{}{"matchExpressions": []interface{}{map[string]interface{}{
				"key": "w7.cc/identifie", "operator": "In", "values": []string{identify},
			}}},
			"topologyKey": "kubernetes.io/hostname",
		}},
	}}
}

func (hc *HelmPack) getWorkloadValues(platform logic2.Platform) map[string]interface{} {
	return map[string]interface{}{
		"kind":           platform.Workload.Type,
		"isDeployment":   platform.Workload.Type == logic2.K8sWorkloadTypeDeployment,
		"isStatefulSet":  platform.Workload.Type == logic2.K8sWorkloadTypeStatefulSet,
		"isDaemonSet":    platform.Workload.Type == logic2.K8sWorkloadTypeDaemonSet,
		"updateStrategy": platform.Workload.UpdateStrategy,
	}
}

func (hc *HelmPack) getVolumeClaimTemplateValues(platform logic2.Platform) []map[string]interface{} {
	if platform.Workload.Type != logic2.K8sWorkloadTypeStatefulSet {
		return []map[string]interface{}{}
	}

	templates := make([]map[string]interface{}, 0)
	for _, item := range platform.VolumeClaimTemplates {
		if item.Name == "" {
			continue
		}
		item.Spec.AccessModes = []v1.PersistentVolumeAccessMode{"{{ .Values.global.cluster.accessModes }}"}
		if item.Spec.Resources.Requests == nil {
			item.Spec.Resources.Requests = v1.ResourceList{}
		}
		storageClass := "{{ .Values.global.cluster.storageClassName }}"
		item.Spec.StorageClassName = &storageClass

		rawItem, err := json.Marshal(item)
		if err != nil {
			continue
		}

		templateItem := make(map[string]interface{})
		if err = json.Unmarshal(rawItem, &templateItem); err != nil {
			continue
		}
		spec, _ := templateItem["spec"].(map[string]interface{})
		if spec == nil {
			spec = make(map[string]interface{})
			templateItem["spec"] = spec
		}
		resources, _ := spec["resources"].(map[string]interface{})
		if resources == nil {
			resources = make(map[string]interface{})
			spec["resources"] = resources
		}
		requests, _ := resources["requests"].(map[string]interface{})
		if requests == nil {
			requests = make(map[string]interface{})
			resources["requests"] = requests
		}
		requests[string(v1.ResourceStorage)] = "{{ .Values.global.cluster.storageSize }}"
		templates = append(templates, templateItem)
	}
	return templates
}

func hasSharedPersistentStorage(mainVolumes []v1.Volume, subVolumes []v1.Volume) bool {
	mainExistsClaim := false
	for _, volume := range mainVolumes {
		if volume.PersistentVolumeClaim != nil {
			mainExistsClaim = true
			break
		}
	}

	if !mainExistsClaim {
		return false
	}

	for _, volume := range subVolumes {
		if volume.PersistentVolumeClaim != nil {
			return true
		}
	}

	return false
}

func buildStableSubPathTemplate(containerName string, volumeMount v1.VolumeMount) string {
	return fmt.Sprintf(`{{ printf "%%s|%%s|%%s|%s|%s|%s" .Release.Name .Release.Namespace .Chart.Name | sha256sum | trunc 12 }}`,
		containerName,
		volumeMount.Name,
		volumeMount.MountPath,
	)
}

func (hc *HelmPack) generateContainerV2Values(container logic2.ContainerV2, application logic2.Application) map[string]interface{} {
	ports := make([]v1.ContainerPort, 0)
	for _, item := range container.Ports {
		item.Name = logic2.GetPortName(item)
		item.HostPort = 0
		ports = append(ports, item)
	}

	for index, volume := range container.VolumeMounts {
		if volume.SubPath == "%RANDOM_DIR%" || volume.SubPath == "RANDOM_DIR" {
			container.VolumeMounts[index].SubPath = buildStableSubPathTemplate(container.Name, volume)
		}
	}

	return map[string]interface{}{
		"name":    strings.ReplaceAll(container.Name, "_", "-"),
		"image":   hc.getImageValues(container),
		"command": container.Command,
		"args":    container.Args,
		"ports":   ports,
		"env":     container.Env,
		// 制品中配置的资源限制暂不写入 Helm，由安装/调度侧统一设置。
		"resources":       v1.ResourceRequirements{},
		"volumeMounts":    container.VolumeMounts,
		"livenessProbe":   container.LivenessProbe,
		"startupProbe":    container.StartupProbe,
		"readinessProb":   container.ReadinessProb,
		"lifecycle":       container.Lifecycle,
		"securityContext": hc.getSecurityContext(container),
		"isInitContainer": container.IsInitContainer,
		"buildImageJobs":  hc.getBuildImageValues(container, application),
	}
}

// generateWorkloadYaml 生成 templates/deployment.yaml
func (hc *HelmPack) generateWorkloadYaml(rootDir string) error {
	if hc.Manifest.Platform.ContainerV2s == nil || len(hc.Manifest.Platform.ContainerV2s) == 0 {
		return nil
	}
	return writeHelmTemplateFile(rootDir, "workload.yaml", "workload.yaml.tpl")
}

// generateServiceYaml 生成 templates/service.yaml
func (hc *HelmPack) generateServiceYaml(rootDir string) error {
	return writeHelmTemplateFile(rootDir, "service.yaml", "service.yaml.tpl")
}

func (hc *HelmPack) generateNodePortServiceYaml(rootDir string) error {
	return writeHelmTemplateFile(rootDir, "node_service.yaml", "node_service.yaml.tpl")
}

func (hc *HelmPack) generateServiceAccountYaml(rootDir string) error {
	return writeHelmTemplateFile(rootDir, "serviceaccount.yaml", "serviceaccount.yaml.tpl")
}

func (hc *HelmPack) generateClusterRoleYaml(rootDir string) error {
	return writeHelmTemplateFile(rootDir, "clusterrole.yaml", "clusterrole.yaml.tpl")
}

func (hc *HelmPack) generateClusterRoleBindingYaml(rootDir string) error {
	return writeHelmTemplateFile(rootDir, "clusterrolebinding.yaml", "clusterrolebinding.yaml.tpl")
}

func (hc *HelmPack) generateSecretYaml(rootDir string) error {
	return writeHelmTemplateFile(rootDir, "secret.yaml", "secret.yaml.tpl")
}

func (hc *HelmPack) generateIngressesYaml(rootDir string) error {
	// Environment applications use a dedicated Ingress which routes through
	// site-manager nginx rather than their own Service.
	if hc.Manifest.Application.Type == logic2.EnvironmentApp {
		return nil
	}
	domain := hc.getIngressDomain(hc.Manifest.Platform)
	if domain == "" {
		return nil
	}

	for name, parent := range hc.IngressNames {
		err := hc.generateIngressYaml(rootDir, name, parent)
		if err != nil {
			return err
		}
	}

	return nil
}

// generateIngressYaml 生成 templates/ingress.yaml
func (hc *HelmPack) generateIngressYaml(rootDir string, ingressName string, parentIngressName string) error {
	parentIngressLabel := ""
	if parentIngressName != "" {
		parentIngressLabel = "parents: {{ $fullName }}-" + parentIngressName
	}
	ingressTemplate, err := loadHelmTemplate("ingress.yaml.tpl")
	if err != nil {
		return err
	}
	ingressTemplate = renderHelmTemplatePlaceholders(ingressTemplate, map[string]string{
		"__INGRESS_NAME__":         ingressName,
		"__PARENT_INGRESS_LABEL__": parentIngressLabel,
	})
	return writeFile(filepath.Join(rootDir, ingressName+"-ingress.yaml"), ingressTemplate)
}

func (hc *HelmPack) generateShellsTemplates(rootDir string) error {
	return writeHelmTemplateFile(rootDir, "shell-job.yaml", "shell-job.yaml.tpl")
}

func (hc *HelmPack) getHelmShells() []logic2.Shell {
	if hc.Manifest.Application.Type == logic2.Tradition_App {
		return hc.getTraditionAppShells()
	}
	return append([]logic2.Shell(nil), hc.Manifest.Platform.Shells...)
}

func (hc *HelmPack) generateBuildImageJobTemplate(rootDir string) error {
	if len(hc.Manifest.Platform.ContainerV2s) == 0 {
		return nil
	}
	return writeHelmTemplateFile(rootDir, "container-build-image.yaml", "container-build-image.yaml.tpl")
}

func getStartParamsEnvJSONTemplate() string {
	return `{{- $startParamsEnv := dict -}}{{- range $qkey, $qvalue := .Values.startParams }}{{- $_ := set $startParamsEnv $qkey (tpl $qvalue $) -}}{{- end }}{{ $startParamsEnv | toJson | b64enc }}`
}

var helmValuesPlaceholderRegexp = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_.-]*)\}`)

func renderHelmValuesPlaceholders(value string) string {
	return helmValuesPlaceholderRegexp.ReplaceAllStringFunc(value, func(match string) string {
		parts := helmValuesPlaceholderRegexp.FindStringSubmatch(match)
		if len(parts) != 2 || strings.HasPrefix(parts[1], "system.") {
			return match
		}
		return "{{ .Values." + parts[1] + " }}"
	})
}

func renderHelmValuesPlaceholdersMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return values
	}
	rendered := make(map[string]string, len(values))
	for key, value := range values {
		rendered[key] = renderHelmValuesPlaceholders(value)
	}
	return rendered
}

func (hc *HelmPack) generateRegisterSiteJobTemplate(rootDir string, manifest logic2.Manifest) error {
	if !manifest.Application.RegisterSite {
		return nil
	}

	siteFilePath := filepath.Join(rootDir, "site-register-site.yaml")
	if function.FileExists(siteFilePath) || function.FileExists(filepath.Join(rootDir, "job-register-site.yaml")) {
		return nil
	}

	siteTemplate, err := loadHelmTemplate("site-register-site.yaml.tpl")
	if err != nil {
		return err
	}
	siteName := manifest.Application.Name
	if siteName == "" {
		siteName = manifest.Application.Identifie
	}
	siteTemplate = renderHelmTemplatePlaceholders(siteTemplate, map[string]string{
		"__APPLICATION_IDENTIFIER__": manifest.Application.Identifie,
		"__SITE_NAME__":              strconv.Quote(siteName),
	})

	return writeFile(siteFilePath, siteTemplate)
}

func (hc *HelmPack) generateGatewayPluginValuesYaml(rootDir string, renderer gatewayPluginRenderer) error {
	plugin := hc.Manifest.Platform.GatewayPlugin.Normalize()
	defaultConfig := plugin.DefaultConfig
	if defaultConfig == nil {
		defaultConfig = make(map[string]interface{})
	}
	appName := hc.Manifest.Application.Name
	if appName == "" {
		appName = hc.Manifest.Application.Identifie
	}
	values := map[string]interface{}{
		"app": map[string]interface{}{
			"title":    appName,
			"identify": hc.Manifest.Application.Identifie,
		},
		"gatewayPlugin": map[string]interface{}{
			"defaultEnabled": plugin.IsEnabledByDefault(),
			"supportGlobal":  plugin.IsSupportGlobal(),
			"supportRule":    plugin.Supports.Rule,
			"defaultConfig":  defaultConfig,
			"configSchema":   plugin.ConfigSchema,
			"hasFrontend":    strings.TrimSpace(hc.Manifest.Web.Url) != "",
			"runtime":        renderer.RuntimeValues(plugin),
		},
	}
	return writeYAMLFile(filepath.Join(rootDir, "values.yaml"), values)
}

func (hc *HelmPack) generateGatewayPluginTemplate(rootDir string, renderer gatewayPluginRenderer) error {
	template, err := loadHelmTemplate(renderer.TemplateName())
	if err != nil {
		return err
	}
	template = renderHelmTemplatePlaceholders(template, map[string]string{
		"__APPLICATION_DESCRIPTION__": strconv.Quote(hc.Manifest.Application.Description),
		"__APPLICATION_IDENTIFY__":    hc.Manifest.Application.Identifie,
		"__APPLICATION_VERSION__":     hc.Manifest.Application.Version,
	})
	return writeFile(filepath.Join(rootDir, renderer.OutputName()), template)
}

func shouldPackageMicroApp(manifest logic2.Manifest) bool {
	if strings.TrimSpace(manifest.Web.Url) != "" {
		return true
	}
	for _, binding := range manifest.Bindings {
		if len(binding.Menu) > 0 {
			return true
		}
	}
	return false
}

func (hc *HelmPack) generateMicroAppTemplate(rootDir string, manifest logic2.Manifest) error {
	microAppFilePath := filepath.Join(rootDir, "microapp.yaml")
	if function.FileExists(microAppFilePath) {
		return nil
	}

	menuConfigValues, backendConfigValues := buildMicroAppValues(manifest.Bindings)
	configMap := map[string]interface{}{
		"backend_config": backendConfigValues,
		"bindings":       menuConfigValues,
	}
	yamlBytes, err := yaml.Marshal(configMap)
	if err != nil {
		return fmt.Errorf("序列化 YAML 失败: %w", err)
	}

	yamlBytes = bytes.Join([][]byte{[]byte("\n"), yamlBytes}, []byte{'\n'})
	err = function.AppendBytesToFile(filepath.Join(filepath.Dir(rootDir), "values.yaml"), yamlBytes)
	if err != nil {
		return err
	}

	appName := manifest.Application.Name
	if appName == "" {
		appName = manifest.Application.Identifie
	}
	manifestType := manifest.Application.Type
	if manifestType == logic2.Docker_App {
		manifestType = "native"
	}

	microAppTemplate, err := loadHelmTemplate("microapp.yaml.tpl")
	if err != nil {
		return err
	}
	microAppTemplate = renderHelmTemplatePlaceholders(microAppTemplate, map[string]string{
		"__APPLICATION_TYPE__":     manifest.Application.Type,
		"__APPLICATION_IDENTIFY__": manifest.Application.Identifie,
		"__APPLICATION_VERSION__":  manifest.Application.Version,
		"__MANIFEST_TYPE__":        manifestType,
		"__APP_TITLE__":            strconv.Quote(appName),
	})

	return writeFile(microAppFilePath, microAppTemplate)
}

// 获取全局配置
func (hc *HelmPack) getGlobalValues() map[string]interface{} {
	return map[string]interface{}{
		"name":                 hc.Manifest.Application.Identifie,
		"namespace":            hc.Manifest.Application.Identifie,
		"siteManagerNamespace": "",
		"clusterDomain":        "cluster.local",
		"imagePullSecrets":     []string{},
		"cluster": map[string]interface{}{
			"storageClassName": "",
			"storageSize":      "1G",
			"accessModes":      "ReadWriteOnce",
		},
	}
}

// 获取 Image 配置
func (hc *HelmPack) getImageValues(container logic2.ContainerV2) map[string]interface{} {
	imageName := container.Image
	if imageName == "" {
		imageName = GetBuildImageName(hc.Manifest.Application)
	}
	if hc.Manifest.Application.Type == logic2.SystemImageApp || hc.Manifest.Application.Type == logic2.EnvironmentApp {
		imageName = strings.ReplaceAll(imageName, "{version}", "{{ .Values.IMAGE_VERSION }}")
	}
	return imageValues(imageName, container.ImagePullPolicy)
}

// 获取 Pod 安全上下文
func (hc *HelmPack) getSecurityContext(container logic2.ContainerV2) map[string]interface{} {
	conf := make(map[string]interface{})

	if container.SecurityContext != nil && container.SecurityContext.RunAsGroup != nil && *container.SecurityContext.RunAsGroup > 0 {
		conf["runAsGroup"] = container.SecurityContext.RunAsGroup
	}
	if container.SecurityContext != nil && container.SecurityContext.RunAsUser != nil && *container.SecurityContext.RunAsUser > 0 {
		conf["runAsUser"] = container.SecurityContext.RunAsUser
	}
	if container.SecurityContext != nil && container.SecurityContext.RunAsNonRoot != nil && *container.SecurityContext.RunAsNonRoot {
		conf["runAsNonRoot"] = true
	}
	if container.SecurityContext != nil && container.SecurityContext.Privileged != nil && *container.SecurityContext.Privileged {
		conf["privileged"] = true
	}

	return conf
}

func (hc *HelmPack) buildShellJobValues(items []logic2.Shell) []map[string]interface{} {
	jobs := make([]map[string]interface{}, 0)
	for _, item := range items {
		shellWeight := 0
		hookName := ""
		switch item.Type {
		case "requireinstall":
			shellWeight = -5
			hookName = "pre-install"
		case "pre-install":
			shellWeight = -4
			hookName = "pre-install"
		case "pre-upgrade":
			shellWeight = -4
			hookName = "pre-upgrade"
		case "install":
			shellWeight = -3
			hookName = "post-install"
		case "post-install":
			shellWeight = -3
			hookName = "post-install"
		case "upgrade":
			shellWeight = -2
			hookName = "post-upgrade"
		case "post-upgrade":
			shellWeight = -2
			hookName = "post-upgrade"
		case "uninstall":
			shellWeight = -1
			hookName = "post-delete"
		case "custom":
			shellWeight = 0
			hookName = "custom"
		}

		jobs = append(jobs, map[string]interface{}{
			"enabled":       true,
			"title":         item.Title,
			"image":         item.Image,
			"shell":         item.Shell,
			"type":          hookName,
			"weight":        shellWeight,
			"containerName": item.Container,
			"name":          strings.ToLower(function.GetRandomStringNotContainerNumber(12)),
		})
	}

	return jobs
}

func (hc *HelmPack) buildJobValues(platform logic2.Platform) []map[string]interface{} {
	jobs := make([]map[string]interface{}, 0)
	for _, job := range hc.buildShellJobValues(platform.Shells) {
		containerName, _ := job["containerName"].(string)
		job["container"] = hc.getShellJobContainerValues(platform, containerName)
		jobs = append(jobs, job)
	}
	return jobs
}

func (hc *HelmPack) getShellJobContainerValues(platform logic2.Platform, containerName string) map[string]interface{} {
	if hc.Manifest.Application.Type == logic2.Tradition_App {
		return hc.getTraditionShellJobContainerValues()
	}
	if len(platform.ContainerV2s) == 0 {
		return map[string]interface{}{
			"name":            strings.ReplaceAll(hc.Manifest.Application.Identifie, "_", "-"),
			"image":           map[string]interface{}{"repository": GetBuildImageName(hc.Manifest.Application), "tag": "latest", "pullPolicy": "IfNotPresent"},
			"env":             []v1.EnvVar{},
			"resources":       v1.ResourceRequirements{},
			"volumeMounts":    []v1.VolumeMount{},
			"securityContext": map[string]interface{}{},
		}
	}

	container := platform.ContainerV2s[0]
	for _, item := range platform.ContainerV2s {
		if !item.IsInitContainer {
			container = item
			break
		}
	}
	if containerName != "" {
		for _, item := range platform.ContainerV2s {
			if item.Name == containerName {
				container = item
				break
			}
		}
	}
	for index, volume := range container.VolumeMounts {
		if volume.SubPath == "%RANDOM_DIR%" || volume.SubPath == "RANDOM_DIR" {
			container.VolumeMounts[index].SubPath = buildStableSubPathTemplate(container.Name, volume)
		}
	}

	return map[string]interface{}{
		"name":            strings.ReplaceAll(container.Name, "_", "-"),
		"image":           hc.getImageValues(container),
		"env":             container.Env,
		"resources":       container.Resources,
		"volumeMounts":    container.VolumeMounts,
		"securityContext": hc.getSecurityContext(container),
	}
}

func (hc *HelmPack) getBuildImageValues(container logic2.ContainerV2, application logic2.Application) []map[string]interface{} {
	if container.Image != "" || container.CodeAttachUrl == "" {
		return make([]map[string]interface{}, 0)
	}

	depot, _ := NewDepot()
	zipUrl, _ := depot.GetFormulaBackendZipDownloadUrlByApplication(
		application, strings.TrimPrefix(hc.Manifest.Source.Url, "file://"), false,
	)

	dockerFilePath := "Dockerfile"
	if container.Build.Context != "" {
		dockerFilePath = strings.Trim(container.Build.Context, "/") + "/" + dockerFilePath
	}

	return []map[string]interface{}{
		{
			"identifie":      application.Identifie,
			"zipUrl":         zipUrl,
			"dockerFilePath": dockerFilePath,
			"buildImageName": GetBuildImageName(hc.Manifest.Application),
		},
	}
}

// 获取 Service 配置
func (hc *HelmPack) getServiceValues(platform logic2.Platform) map[string]interface{} {
	allPorts := make([]interface{}, 0)
	for _, c := range platform.ContainerV2s {
		for _, port := range c.Ports {
			allPorts = append(allPorts, map[string]interface{}{
				"name":       logic2.GetPortName(port),
				"port":       port.ContainerPort,
				"targetPort": port.ContainerPort,
				"protocol":   string(port.Protocol),
			})
		}
	}

	return map[string]interface{}{
		"enabled":     len(allPorts) > 0, // 有端口才启用 Service
		"type":        "ClusterIP",
		"annotations": map[string]interface{}{},
		"name":        "",
		"ports":       allPorts, // 自动填充
	}
}

// 获取 Service 配置
func (hc *HelmPack) getNodePortServiceValues(platform logic2.Platform) map[string]interface{} {
	allPorts := make([]interface{}, 0)
	for _, c := range platform.ContainerV2s {
		for _, port := range c.Ports {
			if port.HostPort > 0 {
				allPorts = append(allPorts, map[string]interface{}{
					"name":       logic2.GetPortName(port),
					"port":       port.HostPort,
					"targetPort": port.ContainerPort,
					"protocol":   string(port.Protocol),
				})
			}
		}
	}

	return map[string]interface{}{
		"enabled":     len(allPorts) > 0, // 有端口才启用 Service
		"type":        "LoadBalancer",
		"annotations": map[string]interface{}{},
		"name":        "",
		"ports":       allPorts, // 自动填充
	}
}

func (hc *HelmPack) getServiceAccountValues(application logic2.Application) map[string]interface{} {
	return map[string]interface{}{
		"create": application.ClusterPrivileged,
		"annotations": struct {
		}{},
		"name": "",
	}
}

func (hc *HelmPack) getIngressDomain(platform logic2.Platform) string {
	domain := ""
	for _, item := range platform.StartParams {
		if item.ValuesText == "%DOMAIN_SSL_URL%" || item.ValuesText == "%DOMAIN_URL%" || item.ValuesText == "%DOMAIN_HOST%" {
			domain = "%DOMAIN_SSL_URL%"
			break
		}
	}

	return domain
}

// 获取 Ingress 配置
func (hc *HelmPack) getIngressValues(platform logic2.Platform, application logic2.Application) map[string]interface{} {
	domain := hc.getIngressDomain(platform)
	if len(platform.Ingress) == 0 && domain == "" {
		return map[string]interface{}{}
	}
	if len(platform.Ingress) == 0 && len(platform.ContainerV2s) > 0 && len(platform.ContainerV2s[0].Ports) > 0 {
		platform.Ingress = []logic2.Ingress{
			logic2.Ingress{
				Name: "/",
				Routes: []logic2.IngressRoute{
					logic2.IngressRoute{
						Path: "/",
						Backend: logic2.IngressBackend{
							Name:  application.Identifie,
							Match: "Prefix",
							Port:  int(platform.ContainerV2s[0].Ports[0].ContainerPort),
						},
					},
				},
			},
		}
	}

	ingressMap := make(map[string]interface{})
	for _, ingress := range platform.Ingress {
		parent := ""
		for _, route := range ingress.Routes {
			serviceKey := function.ConvertDigitsToLetters(function.GetMd5(ingress.Name + "-" + route.Path))
			routeName := "self"
			if route.Backend.Name != application.Identifie {
				routeName = fmt.Sprintf(`{{ index .Values %q "fullnameOverride" }}`, route.Backend.Name)
			}
			rule := map[string]interface{}{
				"host": "",
				"paths": []map[string]interface{}{
					{
						"path":     route.Path,
						"pathType": route.Backend.Match,
						"backend": map[string]interface{}{
							"service": map[string]interface{}{
								"name": routeName,
								"port": map[string]interface{}{
									"number": func() int {
										return route.Backend.Port
									}(),
								},
							},
						},
					},
				},
			}

			var ingressAnnotations = make(map[string]interface{})
			// 添加更多匹配条件（header、method、query）
			hc.addIngressMatchAnnotations(ingressAnnotations, route)
			// 添加重写规则
			hc.addIngressRewriteAnnotations(ingressAnnotations, route)

			if route.Backend.Strategy != nil {
				for key, val := range route.Backend.Strategy {
					ingressAnnotations[key] = val
				}
			}

			hc.IngressNames[serviceKey] = parent
			if parent == "" {
				parent = serviceKey
			}
			ingressMap[serviceKey] = map[string]interface{}{
				"enabled":           true,
				"annotations":       ingressAnnotations,
				"ingressClassName":  "higress",
				"hosts":             []map[string]interface{}{rule},
				"ingressForceHttps": false,
			}
		}
	}

	return ingressMap
}

func (hc *HelmPack) addIngressMatchAnnotations(annotations map[string]interface{}, route logic2.IngressRoute) {
	backend := route.Backend

	// Header 匹配
	if backend.MoreMatch != nil && backend.MoreMatch.Header != nil {
		for _, header := range backend.MoreMatch.Header {
			annotations[fmt.Sprintf("higress.io/%s-match-header-%s", header.Type, header.Key)] = header.Value
		}
	}

	// Query 匹配
	if backend.MoreMatch != nil && backend.MoreMatch.Query != nil {
		for _, query := range backend.MoreMatch.Query {
			annotations[fmt.Sprintf("higress.io/%s-match-query-%s", query.Type, query.Key)] = query.Value
		}
	}

	// Method 匹配
	if backend.MoreMatch != nil && backend.MoreMatch.Method != nil {
		annotations["higress.io/match-method"] = strings.Join(backend.MoreMatch.Method, " ")
	}
}

// addIngressRewriteAnnotations 添加 Ingress 重写注解
func (hc *HelmPack) addIngressRewriteAnnotations(annotations map[string]interface{}, route logic2.IngressRoute) {
	backend := route.Backend

	// Host 重写
	if backend.Rewrite != nil && backend.Rewrite.Host != "" {
		annotations["higress.io/enable-rewrite"] = "true"
		annotations["higress.io/upstream-vhost"] = backend.Rewrite.Host
	}

	// Path 重写
	if backend.Rewrite != nil && backend.Rewrite.Path != "" {
		annotations["higress.io/enable-rewrite"] = "true"
		annotations["higress.io/rewrite-target"] = backend.Rewrite.Path
	}
}

func (hc *HelmPack) getRuntimeClassValues(platform logic2.Platform) map[string]interface{} {
	if platform.RuntimeClassName != "" {
		return map[string]interface{}{
			"enable": true,
			"name":   platform.RuntimeClassName,
		}
	}

	return map[string]interface{}{
		"enable": false,
	}
}

func (hc *HelmPack) getStartParamsEnvValues(platform logic2.Platform) map[string]interface{} {
	values := make(map[string]interface{})
	for _, item := range platform.StartParams {
		values[item.Name] = "{{ " + ".Values." + item.Name + " }}"
	}

	return values
}

func GetBuildImageName(application logic2.Application) string {
	return "registry.local.w7.cc/default/" + application.Identifie + ":" + application.Version
}

func imageValues(imageName string, pullPolicy v1.PullPolicy) map[string]interface{} {
	repository, tag := imageName, "latest"
	if index := strings.LastIndex(imageName, ":"); index > strings.LastIndex(imageName, "/") {
		repository, tag = imageName[:index], imageName[index+1:]
	}
	return map[string]interface{}{
		"repository": repository,
		"tag":        tag,
		"pullPolicy": pullPolicy,
	}
}

// ==================== 文件工具函数 ====================

// writeYAMLFile 将数据写入 YAML 文件
func writeYAMLFile(filePath string, data interface{}) error {
	yamlBytes, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("序列化 YAML 失败: %w", err)
	}

	yamlStr := strings.ReplaceAll(string(yamlBytes), "%", "")

	return writeFile(filePath, yamlStr)
}

// writeFile 写入文件
func writeFile(filePath, content string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer file.Close()

	_, err = io.WriteString(file, content)
	if err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	return nil
}
