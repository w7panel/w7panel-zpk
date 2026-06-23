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
}

func NewHelmPack(manifest logic2.Manifest, subManifests []*logic2.Manifest, outputDir, chartVersion string, isSubFormula bool, sharedStorageTargetApp string) *HelmPack {
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

	traditionReleaseName := ""
	traditionSiteName := ""
	if hc.Manifest.Application.Type == logic2.Tradition_App {
		traditionReleaseName = hc.Manifest.Platform.Tradition.EnvironmentName
		traditionSiteName = buildTraditionSiteName(hc.Manifest.Platform.Tradition)
	}
	if hc.Manifest.Platform.Helm.ChartName != "" || hc.Manifest.Platform.Helm.Repository != "" || (hc.Manifest.Platform.Helm.DependYamls != nil && len(hc.Manifest.Platform.Helm.DependYamls) > 0) {
		err := hc.processHelmPkg(helmDir)
		if err != nil {
			return err
		}
	} else if hc.Manifest.Application.Type == logic2.Tradition_App {
		if err := hc.generateValuesYaml(helmDir); err != nil {
			return err
		}
		if err := hc.generateCreateSiteJobTemplate(templatesDir, hc.Manifest.Application, hc.Manifest.Platform.Tradition, traditionSiteName); err != nil {
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

	if !hc.IsSubFormula {
		if err := hc.generateMicroAppTemplate(templatesDir, hc.Manifest); err != nil {
			return err
		}
		if err := hc.generateRegisterSiteJobTemplate(templatesDir, hc.Manifest, traditionReleaseName, traditionSiteName, traditionSiteName); err != nil {
			return err
		}
	}

	if err := hc.generateChartYaml(helmDir); err != nil {
		return err
	}

	return nil
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
				err = function.DownloadFile(context.Background(), helmDownloadUrl, localHelmZipPath, nil)
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
	helpersTemplate := fmt.Sprintf(`{{/*
Expand the name of the chart.
*/}}
{{- define "common.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "common.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%%s-%%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "common.chart" -}}
{{- printf "%%s-%%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "common.labels" -}}
helm.sh/chart: {{ include "common.chart" . }}
{{ include "common.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
w7.cc/identifie: %s
{{- end }}

{{/*
Selector labels
*/}}
{{- define "common.selectorLabels" -}}
app: {{ include "common.fullname" . }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "common.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "common.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the name of the service to use
*/}}
{{- define "common.serviceName" -}}
{{ include "common.fullname" . }}
{{- end }}

{{/*
Create pull secrets
*/}}
{{- define "common.pullSecrets" -}}
{{- if .Values.image.pullSecrets }}
imagePullSecrets:
{{ toYaml .Values.image.pullSecrets | indent 2 }}
{{- end }}
{{- end }}


{{/*
Renders a list of volumes with proper handling of PVC and global PVC_NAME.
Usage:
  {{ include "common.volumesToYaml" (dict "root" $ "volumes" .Values.volumes) }}
*/}}
{{- define "common.volumesToYaml" }}
{{- $root := .root }}
{{- $volumes := .volumes }}

{{- if $volumes }}
{{- range $vol := $volumes }}
- name: {{ $vol.name | quote }}
  {{- if $vol.persistentVolumeClaim }}
  persistentVolumeClaim:
    claimName: {{ coalesce $vol.persistentVolumeClaim.claimName $root.Values.PVC_NAME | quote }}
  {{- else }}
    {{- /* Reconstruct the volume spec without 'name' and 'persistentVolumeClaim' */}}
    {{- $spec := dict }}
    {{- range $key, $value := $vol }}
      {{- if and (ne $key "name") (ne $key "persistentVolumeClaim") }}
        {{- $_ := set $spec $key $value }}
      {{- end }}
    {{- end }}
    {{- if empty $spec }}
  emptyDir: {}
    {{- else }}
    {{- toYaml $spec | nindent 2 }}
    {{- end }}
  {{- end }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Renders only volumeMounts backed by pod volumes.
Used by Job templates, which cannot consume StatefulSet volumeClaimTemplates.
*/}}
{{- define "common.jobVolumeMountsToYaml" }}
{{- $root := .root }}
{{- $mounts := .mounts }}
{{- $allowed := dict }}
{{- range $vol := $root.Values.volumes }}
{{- if $vol.name }}
{{- $_ := set $allowed $vol.name true }}
{{- end }}
{{- end }}
{{- $validMounts := list }}
{{- range $mount := $mounts }}
{{- if hasKey $allowed $mount.name }}
{{- $validMounts = append $validMounts $mount }}
{{- end }}
{{- end }}
{{- if $validMounts }}
{{- tpl (toYaml $validMounts) $root }}
{{- end }}
{{- end }}
`, identify)

	filePath := filepath.Join(rootDir, "_helpers.tpl")
	return writeFile(filePath, helpersTemplate)
}

// generateValuesYaml 生成 values.yaml
func (hc *HelmPack) generateValuesYaml(rootDir string) error {
	platform := hc.Manifest.Platform
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
		"workload":             hc.getWorkloadValues(hc.Manifest.Platform),
		"service":              hc.getServiceValues(hc.Manifest.Platform),
		"node_service":         hc.getNodePortServiceValues(hc.Manifest.Platform),
		"ingress":              hc.getIngressValues(platform, hc.Manifest.Application),
		"serviceAccount":       hc.getServiceAccountValues(hc.Manifest.Application),
		"volumes":              hc.Manifest.Platform.Volumes,
		"volumeClaimTemplates": hc.getVolumeClaimTemplateValues(hc.Manifest.Platform),
		"defaultPort":          defaultPort,
		"startParams":          hc.getStartParamsEnvValues(hc.Manifest.Platform),
		"gpu":                  hc.getGpuValues(hc.Manifest.Platform),
		"sharedStorageAffinity": map[string]interface{}{
			"targetSelectorApp": hc.SharedStorageTargetApp,
		},
	}
	helmContainers := make([]map[string]interface{}, 0, len(hc.Manifest.Platform.ContainerV2s))
	for _, container := range hc.Manifest.Platform.ContainerV2s {
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

func (hc *HelmPack) generatePortName(port v1.ContainerPort) string {
	return "port-" + strconv.Itoa(int(port.ContainerPort))
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
		"name":            strings.ReplaceAll(container.Name, "_", "-"),
		"image":           hc.getImageValues(container),
		"command":         container.Command,
		"args":            container.Args,
		"ports":           ports,
		"env":             container.Env,
		"resources":       v1.ResourceRequirements{},
		"volumeMounts":    container.VolumeMounts,
		"livenessProbe":   container.LivenessProbe,
		"startupProbe":    container.StartupProbe,
		"readinessProb":   container.ReadinessProb,
		"lifecycle":       container.Lifecycle,
		"securityContext": hc.getSecurityContext(container),
		"isInitContainer": container.IsInitContainer,
		"jobs":            hc.getJobValues(container),
		"buildImageJobs":  hc.getBuildImageValues(container, application),
	}
}

// generateWorkloadYaml 生成 templates/deployment.yaml
func (hc *HelmPack) generateWorkloadYaml(rootDir string) error {
	if hc.Manifest.Platform.ContainerV2s == nil || len(hc.Manifest.Platform.ContainerV2s) == 0 {
		return nil
	}

	workloadTemplate := `apiVersion: apps/v1
kind: {{ .Values.workload.kind }}
metadata:
  name: {{ include "common.fullname" . }}
  labels:
    {{- include "common.labels" . | nindent 4 }}
  annotations:
    title: {{ .Values.app.title | quote }}
    w7.cc.app/title: {{ .Values.app.title | quote }}
    w7.cc/create-svc: 'false'
    w7.cc/group-name: {{ .Release.Name }}
spec:
  {{- if not .Values.workload.isDaemonSet }}
  replicas: {{ .Values.replicas }}
  {{- end }}
  {{- if and .Values.workload.isDeployment .Values.workload.updateStrategy }}
  strategy: {{- toYaml .Values.workload.updateStrategy | nindent 4 }}
  {{- end }}
  {{- if .Values.workload.isStatefulSet }}
  serviceName: {{ include "common.serviceName" . }}
  podManagementPolicy: OrderedReady
  {{- end }}
  {{- if and (or .Values.workload.isDaemonSet .Values.workload.isStatefulSet) .Values.workload.updateStrategy }}
  updateStrategy: {{- toYaml .Values.workload.updateStrategy | nindent 4 }}
  {{- end }}
  selector:
    matchLabels:
      {{- include "common.selectorLabels" . | nindent 6 }}
  template:
    metadata:
      annotations:
      {{- if .Values.podAnnotations }}
        {{- toYaml . | nindent 8 }}
      {{- end }}
        {{- if .Values.annotations }}
        {{- toYaml .Values.annotations | nindent 12 }}
        {{- end }}
      labels:
        {{- include "common.selectorLabels" . | nindent 8 }}
        w7.cc/identifie: {{ .Values.app.identify | quote }}
    spec:
      {{- if .Values.gpu.enable }}
      runtimeClassName: {{ .Values.gpu.driver }}
      {{- end }}

      {{- $podVolumes := .Values.volumes }}
      {{- if .Values.workload.isStatefulSet }}
      {{- $claimTemplateNames := dict }}
      {{- range .Values.volumeClaimTemplates }}
      {{- $_ := set $claimTemplateNames .metadata.name true }}
      {{- end }}
      {{- $filteredVolumes := list }}
      {{- range .Values.volumes }}
      {{- if not (and .persistentVolumeClaim (hasKey $claimTemplateNames .name)) }}
      {{- $filteredVolumes = append $filteredVolumes . }}
      {{- end }}
      {{- end }}
      {{- $podVolumes = $filteredVolumes }}
      {{- end }}
      {{- if $podVolumes }}
      volumes:
        {{- include "common.volumesToYaml" (dict "root" . "volumes" $podVolumes) | nindent 8 }}
      {{- end }}

      {{- with .Values.imagePullSecrets }}
      imagePullSecrets:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      serviceAccountName: {{ include "common.serviceAccountName" . }}
      {{- if .Values.sharedStorageAffinity.targetSelectorApp }}
      affinity:
        podAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            - labelSelector:
                matchExpressions:
                  - key: w7.cc/identifie
                    operator: In
                    values:
                      - {{ .Values.sharedStorageAffinity.targetSelectorApp | quote }}
              topologyKey: kubernetes.io/hostname
      {{- end }}

	    {{- $root := . }}
      {{- $rootCtx := $ }}
      containers:
      {{- range .Values.containers }}
      {{- if not .isInitContainer }}
        - name: {{ .name }}
          image: "{{ .image.repository }}:{{ .image.tag }}"
          imagePullPolicy: {{ .image.pullPolicy }}
          {{- with .command }}
          command: {{- toYaml . | nindent 12 }}
          {{- end }}
          {{- with .args }}
          args: {{- toYaml . | nindent 12 }}
          {{- end }}
          {{- with .ports }}
          ports: {{- toYaml . | nindent 12 }}
          {{- end }}
          env: 
            {{- with .env }}
            {{- toYaml . | nindent 12 }}
            {{- end }}
            {{- if $root.Values.startParams }}
            {{- range $qkey, $qvalue := $root.Values.startParams }}
            - name: {{ $qkey }}
              value: {{ tpl $qvalue $rootCtx | quote }}
            {{- end }}
            {{- end }}
          {{- with .resources }}
          resources: {{- toYaml . | nindent 12 }}
          {{- end }}
          {{- with .volumeMounts }}
          volumeMounts: {{- tpl (toYaml .) $rootCtx | nindent 12 }}
          {{- end }}
          {{- with .livenessProbe }}
          livenessProbe: {{- toYaml . | nindent 12 }}
          {{- end }}
          {{- with .startupProbe }}
          startupProbe: {{- toYaml . | nindent 12 }}
          {{- end }}
          {{- with .lifecycle }}
          lifecycle: {{- toYaml . | nindent 12 }}
          {{- end }}
          {{- with .securityContext }}
          securityContext: {{- toYaml . | nindent 12 }}
          {{- end }}
      {{- end }}
      {{- end }}

      {{- $hasInit := false }}
      {{- range .Values.containers }}
        {{- if .isInitContainer }}
          {{- $hasInit = true }}
        {{- end }}
      {{- end }}
      {{- if $hasInit }}
      initContainers:
        {{- range .Values.containers }}
        {{- if .isInitContainer }}
        - name: {{ .name }}
          image: "{{ .image.repository }}:{{ .image.tag }}"
          imagePullPolicy: {{ .image.pullPolicy }}
          {{- with .command }}
          command: {{- toYaml . | nindent 12 }}
          {{- end }}
          {{- with .args }}
          args: {{- toYaml . | nindent 12 }}
          {{- end }}
          env:
            {{- with .env }}
            {{- toYaml . | nindent 12 }}
            {{- end }}
            {{- if $root.Values.startParams }}
            {{- range $qkey, $qvalue := $root.Values.startParams }}
            - name: {{ $qkey }}
              value: {{ tpl $qvalue $rootCtx | quote }}
          {{- end }}
          {{- end }}
          {{- with .resources }}
          resources: {{- toYaml . | nindent 12 }}
          {{- end }}
          {{- with .volumeMounts }}
          volumeMounts: {{- tpl (toYaml .) $rootCtx | nindent 12 }}
          {{- end }}
          {{- with .securityContext }}
          securityContext: {{- toYaml . | nindent 12 }}
          {{- end }}
      {{- end }}
      {{- end }}
  {{- end }}
  {{- if and .Values.workload.isStatefulSet .Values.volumeClaimTemplates }}
  volumeClaimTemplates:
    {{- tpl (toYaml .Values.volumeClaimTemplates) . | nindent 4 }}
  {{- end }}
`
	filePath := filepath.Join(rootDir, "workload.yaml")
	return writeFile(filePath, workloadTemplate)
}

// generateServiceYaml 生成 templates/service.yaml
func (hc *HelmPack) generateServiceYaml(rootDir string) error {
	serviceTemplate := `{{- if .Values.service.ports }}
apiVersion: v1
kind: Service
metadata:
  name: {{ include "common.fullname" . }}
  labels:
    {{- include "common.labels" . | nindent 4 }}
spec:
  type: {{ .Values.service.type }}
  selector:
    {{- include "common.selectorLabels" . | nindent 4 }}
  ports:
    {{- range .Values.service.ports }}
    - port: {{ .port }}
      targetPort: {{ .targetPort }}
      protocol: {{ .protocol }}
      name: {{ .name }}
    {{- end }}
{{- end }}
`
	filePath := filepath.Join(rootDir, "service.yaml")
	return writeFile(filePath, serviceTemplate)
}

func (hc *HelmPack) generateNodePortServiceYaml(rootDir string) error {
	serviceTemplate := `{{- if .Values.node_service.ports }}
apiVersion: v1
kind: Service
metadata:
  name: {{ include "common.fullname" . }}-lb
  labels:
    {{- include "common.labels" . | nindent 4 }}
spec:
  type: {{ .Values.node_service.type }}
  selector:
    {{- include "common.selectorLabels" . | nindent 4 }}
  ports:
    {{- range .Values.node_service.ports }}
    - port: {{ .port }}
      targetPort: {{ .targetPort }}
      protocol: {{ .protocol }}
      name: {{ .name }}
    {{- end }}
{{- end }}
`
	filePath := filepath.Join(rootDir, "node_service.yaml")
	return writeFile(filePath, serviceTemplate)
}

func (hc *HelmPack) generateServiceAccountYaml(rootDir string) error {
	serviceAccountYamlTemplate := `apiVersion: v1
kind: ServiceAccount
metadata:
  name: {{ include "common.serviceAccountName" . }}
  labels:
    {{- include "common.labels" . | nindent 4 }}
  annotations:
    w7.cc/menu-name: k3k.permission.founder
`
	filePath := filepath.Join(rootDir, "serviceaccount.yaml")
	return writeFile(filePath, serviceAccountYamlTemplate)
}

func (hc *HelmPack) generateClusterRoleYaml(rootDir string) error {
	clusterRoleYamlTemplate := `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: {{ .Release.Name }}
  labels:
    {{- include "common.labels" . | nindent 4 }}
rules:
- apiGroups: ["*"]
  resources: ["*"]
  verbs: ["*"]
`
	filePath := filepath.Join(rootDir, "clusterrole.yaml")
	return writeFile(filePath, clusterRoleYamlTemplate)
}

func (hc *HelmPack) generateClusterRoleBindingYaml(rootDir string) error {
	clusterRoleBindingYamlTemplate := `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: {{ .Release.Name }}
  labels:
    {{- include "common.labels" . | nindent 4 }}
subjects:
- kind: ServiceAccount
  name: {{ .Release.Name }}
  namespace: default
roleRef:
  kind: ClusterRole
  name: {{ .Release.Name }}
  apiGroup: rbac.authorization.k8s.io
`
	filePath := filepath.Join(rootDir, "clusterrolebinding.yaml")
	return writeFile(filePath, clusterRoleBindingYamlTemplate)
}

func (hc *HelmPack) generateSecretYaml(rootDir string) error {
	secretYamlTemplate := `apiVersion: v1
kind: Secret
metadata:
  name: {{ .Release.Name }}
  annotations:
    kubernetes.io/service-account.name: {{ .Release.Name }}
  labels:
    {{- include "common.labels" . | nindent 4 }}
type: kubernetes.io/service-account-token
`
	filePath := filepath.Join(rootDir, "secret.yaml")
	return writeFile(filePath, secretYamlTemplate)
}

func (hc *HelmPack) generateIngressesYaml(rootDir string) error {
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
	if parentIngressName != "" {
		parentIngressName = "parents: {{ $fullName }}-" + parentIngressName
	}
	ingressTemplate := fmt.Sprintf(`{{- if .Values.ingress.%s.enabled -}}
{{- $fullName := include "common.fullname" . -}}
{{- $appName := .Values.global.name -}}
{{- $releaseName := .Release.Name -}}
{{- $domain := .Values.DOMAIN_URL -}}
{{- if and .Values.ingress.%s.ingressClassName (not (semverCompare ">=1.18-0" .Capabilities.KubeVersion.GitVersion)) }}
  {{- if not (hasKey .Values.ingress.%s.annotations "kubernetes.io/ingress.class") }}
  {{- $_ := set .Values.ingress.%s.annotations "kubernetes.io/ingress.class" .Values.ingress.%s.ingressClassName}}
  {{- end }}
{{- end }}
{{- if semverCompare ">=1.19-0" .Capabilities.KubeVersion.GitVersion -}}
apiVersion: networking.k8s.io/v1
{{- else if semverCompare ">=1.14-0" .Capabilities.KubeVersion.GitVersion -}}
apiVersion: networking.k8s.io/v1beta1
{{- else -}}
apiVersion: extensions/v1beta1
{{- end }}
kind: Ingress
metadata:
  name: {{ $fullName }}-%s
  labels:
    {{- include "common.labels" . | nindent 4 }}
    group: {{ $releaseName }}
    %s
  annotations:
  {{- with .Values.ingress.%s.annotations }}
    {{- toYaml . | nindent 4 }}
  {{- end }}
    {{- if .Values.ingressForceHttps }}
    cert-manager.io/cluster-issuer: w7-letsencrypt-prod
    cert-manager.io/renew-before: 30m
    {{- end }}
    kubernetes.io/ingress.class: higress
spec:
  {{- if .Values.ingressForceHttps }}
  tls:
    - hosts:
        {{- range .Values.ingress.%s.hosts }}
        - {{ coalesce .host $domain | quote }}
        {{- end }}
      secretName: {{ $fullName }}-%s-tls-secret
  {{- end }}
  rules:
    {{- range .Values.ingress.%s.hosts }}
    - host: {{ coalesce .host $domain | quote }}
      http:
        paths:
          {{- range .paths }}
          - path: {{ .path }}
            {{- if and .pathType (semverCompare ">=1.18-0" $.Capabilities.KubeVersion.GitVersion) }}
            pathType: {{ .pathType }}
            {{- end }}
            backend:
              {{- if semverCompare ">=1.19-0" $.Capabilities.KubeVersion.GitVersion }}
              service:
                {{- if eq .backend.service.name "self" }}
                name: {{ $fullName }}
                {{- else }}
                name: {{ tpl .backend.service.name $ }}
                {{- end }}
                port:
                  number: {{ .backend.service.port.number }}
              {{- else }}
              {{- if eq .backend.service.name "self" }}
              serviceName: {{ $fullName }}
              {{- else }}
              serviceName: {{ tpl .backend.service.name $ }}
              {{- end }}
              servicePort: {{ .backend.service.port.number }}
              {{- end }}
          {{- end }}
    {{- end }}
{{- end }}
`, ingressName, ingressName, ingressName, ingressName, ingressName, ingressName, parentIngressName, ingressName, ingressName, ingressName, ingressName)
	filePath := filepath.Join(rootDir, ingressName+"-ingress.yaml")
	return writeFile(filePath, ingressTemplate)
}

func (hc *HelmPack) generateShellsTemplates(rootDir string) error {
	if len(hc.Manifest.Platform.ContainerV2s) == 0 {
		return nil
	}

	jobTemplate := `{{- if or (hasKey .Values "containers") (gt (len .Values.containers) 0) }}
  {{- $hasJobs := false }}
  {{- range .Values.containers }}
    {{- if and (hasKey . "jobs") (gt (len .jobs) 0) }}
      {{- $hasJobs = true }}
    {{- end }}
  {{- end }}
  {{- if $hasJobs }}
	{{- $root := . }}
    {{/* 渲染所有 Job */}}
    {{- range $container := .Values.containers }}
      {{- range $job := $container.jobs }}
---
apiVersion: batch/v1
kind: Job
metadata:
  name: {{ include "common.fullname" $root }}-{{ $container.name }}-job-{{ $job.name }}
  labels:
    {{- include "common.labels" $root | nindent 4 }}
    group: {{ $root.Release.Name }}
    w7.cc/job-source: appgroup
  annotations:
	{{- if ne $job.type "custom" }}
    helm.sh/hook: {{ $job.type }}
    helm.sh/hook-weight: "{{ $job.weight }}"
    helm.sh/hook-delete-policy: before-hook-creation,hook-succeeded
	{{- else }}
    w7.cc/custom-hook: 'true'
    {{- end }}
spec:
  backoffLimit: 2
  ttlSecondsAfterFinished: 60
  template:
    metadata:
      annotations:
      {{- if $root.Values.podAnnotations }}
        {{- toYaml . | nindent 8 }}
      {{- end }}
        {{- if $root.Values.annotations }}
        {{- toYaml $root.Values.annotations | nindent 12 }}
        {{- end }}
    spec:
      restartPolicy: Never
      serviceAccountName: {{ include "common.serviceAccountName" $root }}
      affinity:
        podAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            - labelSelector:
                matchExpressions:
                  - key: w7.cc/identifie
                    operator: In
                    values:
                      - {{ $root.Values.app.identify | quote }}
              topologyKey: kubernetes.io/hostname
      {{- if $root.Values.volumes }}
      volumes:
        {{- include "common.volumesToYaml" (dict "root" $root "volumes" $root.Values.volumes) | nindent 8 }}
      {{- end }}
      containers:
        - name: {{ $container.name }}-{{ $job.name }}
          {{- if $job.image }}
          image: {{ $job.image | quote }}
          {{- else }}
          image: "{{ $container.image.repository }}:{{ $container.image.tag }}"
          {{- end }}
          imagePullPolicy: {{ $container.image.pullPolicy | default "IfNotPresent" }}
          command: ["/bin/sh", "-c"]
          args:
            - {{ $job.shell | quote }}
          env:
            {{- with $container.env }}
            {{- toYaml . | nindent 12 }}
            {{- end }}
            {{- if $root.Values.startParams }}
            {{- range $k, $v := $root.Values.startParams }}
            - name: {{ $k | quote }}
              value: {{ tpl $v $root | quote }}
            {{- end }}
            {{- end }}
          {{- with $container.resources }}
          resources: {{- toYaml . | nindent 12 }}
          {{- end }}
          {{- $renderedVolumeMounts := include "common.jobVolumeMountsToYaml" (dict "root" $root "mounts" $container.volumeMounts) }}
          {{- if $renderedVolumeMounts }}
          volumeMounts: {{- $renderedVolumeMounts | nindent 12 }}
          {{- end }}
          {{- with $container.securityContext }}
          securityContext: {{- toYaml . | nindent 12 }}
          {{- end }}
      {{- end }}
    {{- end }}
  {{- end }}
{{- end }}
`

	filePath := filepath.Join(rootDir, "container-job.yaml")
	return os.WriteFile(filePath, []byte(jobTemplate), 0644)
}

func (hc *HelmPack) generateBuildImageJobTemplate(rootDir string) error {
	if len(hc.Manifest.Platform.ContainerV2s) == 0 {
		return nil
	}

	jobTemplate := `{{- if or (hasKey .Values "containers") (gt (len .Values.containers) 0) }}
  {{- $hasJobs := false }}
  {{- range .Values.containers }}
    {{- if and (hasKey . "buildImageJobs") (gt (len .buildImageJobs) 0) }}
      {{- $hasJobs = true }}
    {{- end }}
  {{- end }}
  {{- if $hasJobs }}
	{{- $root := . }}
    {{/* 渲染所有 Job */}}
    {{- range $container := .Values.containers }}
      {{- range $job := $container.buildImageJobs }}
---
apiVersion: batch/v1
kind: Job
metadata:
  name: {{ include "common.fullname" $root }}-{{ $container.name }}-job-build-image
  labels:
    {{- include "common.labels" $root | nindent 4 }}
    group: {{ $root.Release.Name }}
    w7.cc/job-source: appgroup
spec:
  parallelism: 1
  completions: 1
  backoffLimit: 3
  ttlSecondsAfterFinished: 60
  template:
    metadata:
      {{- if $root.Values.podAnnotations }}
      annotations:
        {{- toYaml . | nindent 8 }}
      {{- end }}
    spec:
      restartPolicy: Never
      volumes:
        -
          name: my-host
          hostPath:
            path: /
            type: ''
      containers:
         -
            name: docker-build
            image: ccr.ccs.tencentyun.com/afan-public/kaniko:w7console-new5-19
            command:
               - /kaniko/start.sh
            workingDir: /workspace
            env:
              -
                name: NOTIFY_COMPLETION_URL
                value: /
              -
                name: KANIKO_REGISTRY_MAP
                value: "index.docker.io=mirror.ccs.tencentyun.com;index.docker.io=registry.cn-hangzhou.aliyuncs.com;\n\tindex.docker.io=docker.m.daocloud.io;index.docker.io=docker.1panel.live"
              -
                name: EMBED
                value: 'true'
              -
                name: USER_AGENT
                value: release
              -
                name: MODULE_NAME
                value: {{ $job.identifie }}
              -
                name: DOCKER_AUTH
                value: >-
                  {"auths":{"registry.local.w7.cc":{"auth":"YWRtaW46dzctc2VjcmV0"}}}
              -
                name: DOWNLOAD_URL
                value: {{ $job.zipUrl }}
              -
                name: NOTIFY_FAILED_URL
                value: /
              -
                name: CURL_CA_BUNDLE
                value: /kaniko/ssl/certs/ca-certificates.crt
              -
                name: CONTEXT
                value: /workspace
              -
                name: DOCKER_FILE
                value: /workspace/{{ $job.dockerFilePath }}
              -
                name: ATTACHMENT_TYPE
                value: zip
              -
                name: PUSH_IMAGE
                value: {{ $job.buildImageName }}
              -
                name: INSECURE
                value: '--insecure --insecure-pull'
            resources: {}
            volumeMounts:
              -
                name: my-host
                mountPath: /host
      {{- end }}
    {{- end }}
  {{- end }}
{{- end }}
`

	filePath := filepath.Join(rootDir, "container-build-image.yaml")
	return os.WriteFile(filePath, []byte(jobTemplate), 0644)
}

func getVersionIdentifie(appName, version string) string {
	if version != "" {
		cleanVersion := strings.ReplaceAll(version, ".", "")
		return appName + "_" + cleanVersion
	}
	return appName
}

func buildTraditionSiteName(tradition logic2.Tradition) string {
	appName := tradition.EnvironmentName
	version := tradition.EnvironmentVersion
	return "copy-" + strings.ToLower(function.GetRandomStringNotContainerNumber(6)) + "-" + strings.ReplaceAll(getVersionIdentifie(appName, version), "_", "-")
}

func getStartParamsEnvJSONTemplate() string {
	return `{{- $startParamsEnv := dict -}}{{- range $qkey, $qvalue := .Values.startParams }}{{- $_ := set $startParamsEnv $qkey (tpl $qvalue $) -}}{{- end }}{{ $startParamsEnv | toJson | b64enc }}`
}

func (hc *HelmPack) generateCreateSiteJobTemplate(rootDir string, application logic2.Application, tradition logic2.Tradition, k8sAppName string) error {
	depot, _ := NewDepot()
	zipUrl, _ := depot.GetFormulaBackendZipDownloadUrlByApplication(application, false)

	cmd := ""
	if len(tradition.Cmd) > 0 {
		val, err := json.Marshal(tradition.Cmd)
		if err != nil {
			return err
		}
		cmd = string(val)
	}

	jobTemplate := fmt.Sprintf(`{{- define "__cur__.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- .Release.Name }}-{{ $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{- $fullName := include "__cur__.fullname" . -}}

apiVersion: batch/v1
kind: Job
metadata:
  name: {{ $fullName }}-create-register-site
  labels:
    group: {{ .Release.Name }}
    w7.cc/job-source: appgroup
  annotations:
    helm.sh/hook: pre-install,pre-upgrade
    helm.sh/hook-weight: "1"
    helm.sh/hook-delete-policy: before-hook-creation,hook-succeeded
spec:
  backoffLimit: 2
  ttlSecondsAfterFinished: 60
  template:
    metadata:
      {{- if .Values.podAnnotations }}
      annotations:
        {{- toYaml . | nindent 8 }}
      {{- end }}
    spec:
      restartPolicy: Never
      containers:
        - name: create-site-job
          image: zpk.w7.cc/public/site-manager:v1.2.6
          command:
            - sh
            - -c
            - /home/rangine create:site --operation={{ ternary "upgrade" "install" .Release.IsUpgrade }} --w7panel-domain={{ .Values.global.panel.innerUrl }} --w7panel-token={{ .Values.global.panel.panelRealToken }} --title=%s --name=%s --language=%s --version=%s --domain={{ .Values.DOMAIN_URL }} --ssl={{ default false .Values.ingressForceHttps }} --cmd=%s --code-download-url=%s --app_name=%s --k8s-app-name={{ $fullName }} --k8s-env-app-name=%s --start-params-env-base64=%s -f /home/config.yaml
`, application.Identifie+"-"+tradition.EnvironmentVersion+"-副本", tradition.EnvironmentName, tradition.EnvironmentLanguage, tradition.EnvironmentVersion, cmd, zipUrl, application.Identifie, k8sAppName, getStartParamsEnvJSONTemplate())

	filePath := filepath.Join(rootDir, "create-site-job.yaml")
	return os.WriteFile(filePath, []byte(jobTemplate), 0644)
}

func (hc *HelmPack) generateRegisterSiteJobTemplate(rootDir string, manifest logic2.Manifest, releaseName, appName, containerName string) error {
	if !manifest.Application.RegisterSite {
		return nil
	}

	jobFilePath := filepath.Join(rootDir, "job-register-site.yaml")
	if function.FileExists(jobFilePath) {
		return nil
	}

	if releaseName == "" {
		releaseName = "{{ .Release.Name }}"
	}
	if appName == "" {
		appName = "{{ $fullName }}"
	}
	if containerName == "" {
		containerName = manifest.Application.Identifie
		if len(manifest.Platform.ContainerV2s) > 0 {
			containerName = manifest.Platform.ContainerV2s[0].Name
		}
	}

	jobTemplate := fmt.Sprintf(`{{- define "__cur__.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- .Release.Name }}-{{ $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{- $fullName := include "__cur__.fullname" . -}}

apiVersion: batch/v1
kind: Job
metadata:
  name: {{ $fullName }}-job-register-site
  labels:
    group: {{ .Release.Name }}
    w7.cc/job-source: appgroup
spec:
  backoffLimit: 2
  ttlSecondsAfterFinished: 60
  template:
    metadata:
      {{- if .Values.podAnnotations }}
      annotations:
        {{- toYaml . | nindent 8 }}
      {{- end }}
    spec:
      restartPolicy: Never
      containers:
        - name: register-site-job
          image: {{ .Values.global.panel.image }}
          command:
            - sh
            - -c
            - /ko-app/w7panel site:register-zpk-http --panelUrl={{ .Values.global.panel.innerUrl }} --installId={{ .Values.global.panel.installId }} --siteIdentifie=%s --host={{ .Values.DOMAIN_URL }} --releaseName=%s --appName=%s --namespace=default --containerName=%s
`, manifest.Application.Identifie, releaseName, appName, containerName)

	return writeFile(jobFilePath, jobTemplate)
}

func (hc *HelmPack) generateMicroAppTemplate(rootDir string, manifest logic2.Manifest) error {
	microAppFilePath := filepath.Join(rootDir, "microapp.yaml")
	if function.FileExists(microAppFilePath) {
		return nil
	}

	menuConfigValues := make([]map[string]interface{}, 0)
	backendConfigValues := make([]map[string]interface{}, 0)
	for _, item := range manifest.Bindings {
		conf := map[string]interface{}{
			"role":               item.Name,
			"load_mode":          item.LoadMode,
			"type":               item.BackendConfig.Type,
			"backend_identifier": item.BackendConfig.BackendIdentifie,
			"backend_port":       item.BackendConfig.BackendPort,
			"proxy_request":      item.BackendConfig.RequestProxy,
			"frontend_props":     item.BackendConfig.FrontendProps,
		}
		menuConfigValues = append(menuConfigValues, map[string]interface{}{
			"title":               item.Title,
			"name":                item.Name,
			"status":              item.Status,
			"support":             item.Support,
			"framework":           item.Framework,
			"is_default_register": item.IsDefaultRegister,
			"location":            item.Location,
			"menu":                item.Menu,
		})
		backendConfigValues = append(backendConfigValues, conf)
	}
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

	microAppTemplate := fmt.Sprintf(`{{- $releaseNamespace := .Release.Namespace -}}
{{- $defaultPort := .Values.defaultPort -}}
{{- if and (not $defaultPort) .Values.service (hasKey .Values.service "port") .Values.service.port -}}
{{- $defaultPort = .Values.service.port -}}
{{- end -}}
{{- $releaseName := .Release.Name -}}
{{- $applicationType := "%s" -}}

{{- define "__cur__.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- .Release.Name }}-{{ $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{- $fullName := include "__cur__.fullname" . -}}

apiVersion: microapp.w7.cc/v1alpha1
kind: MicroApp
metadata:
  name: {{ $releaseName }}
  labels:
    w7.cc/identifie: "%s"
    w7.cc/version: "%s"
    {{- range .Values.backend_config }}
    role.w7.cc/{{ .role }}: "true"
    {{- end }}
  annotations:
    w7.cc/version: "%s"
spec:
  title: %s
  frontendUrl: /ui/microapp/%s/%s/index.html
  
  config-v2:
    props:
      roleConfig:
        {{- range .Values.backend_config }}
        {{ .role }}:
          {{- if eq .load_mode "iframe" }}
          serverUrl: {{ tpl .backend_identifier $ | quote }}
          {{- else if eq .type "internal" }}
          {{- if eq $applicationType "tradition" }}
          serverUrl: {{ tpl .backend_identifier $ | quote }}
          {{- else }}
          serverUrl: "http://{{ $fullName }}.{{ $releaseNamespace }}.svc.cluster.local:{{ default $defaultPort .backend_port }}"
          {{- end }}
          {{- else if eq .type "external" }}
          serverUrl: {{ .backend_identifier }}
          {{- end }}
          load_mode: {{ .load_mode }}
          proxy_request: 
            {{- if .proxy_request.headers }}
            headers:
              {{- range $hkey, $hvalue := .proxy_request.headers }}
              {{ $hkey }}: {{ tpl $hvalue $ }}
              {{- end }}
            {{- end }}
            {{- if .proxy_request.query }}
            query:
              {{- range $qkey, $qvalue := .proxy_request.query }}
              {{ $qkey }}: {{ tpl $qvalue $ }}
              {{- end }}
            {{- end }}
          frontend_props:
            {{- range $fkey, $fvalue := .frontend_props }}
            {{ $fkey }}: {{ tpl $fvalue $ }}
            {{- end }}
            group: {{ $releaseName }}
            url: /panel-api/v1/microapp/{{ $releaseName }}/proxy
        {{- end }}
  bindings:
    {{- range .Values.bindings }}
    - name: {{ .name }}
      title: {{ .title }}
      status: {{ .status }}
      support: {{ .support }}
      framework: {{ .framework }}
      is_default_register: {{ .is_default_register }}
      location: {{ .location }}
      menu:
        {{- range .menu }}
        - displayorder: {{ .displayorder }}
          do: "{{ .do }}"
          title: {{ .title }}
          icon: {{ .icon }}
          {{- if .icon_svg }}
          icon_svg:
          {{- toYaml .icon_svg | nindent 12 }}
          {{- else }}
          icon_svg: null
          {{- end }}
          location: {{ .location }}
          is_default: {{ .is_default }}
          parent: "{{ .parent }}"
          {{- end }}
      {{- end }}
`, manifest.Application.Type, manifest.Application.Identifie, manifest.Application.Version, manifest.Application.Version, appName, manifest.Application.Identifie, manifest.Application.Version)

	return writeFile(microAppFilePath, microAppTemplate)
}

// 获取全局配置
func (hc *HelmPack) getGlobalValues() map[string]interface{} {
	return map[string]interface{}{
		"name":             hc.Manifest.Application.Identifie,
		"namespace":        hc.Manifest.Application.Identifie,
		"clusterDomain":    "cluster.local",
		"imagePullSecrets": []string{},
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
	imageParts := strings.Split(imageName, ":")
	repository := imageName
	tag := "latest"
	if len(imageParts) > 1 {
		tag = imageParts[1]
		repository = imageParts[0]
	}

	return map[string]interface{}{
		"repository": repository,
		"tag":        tag,
		"pullPolicy": container.ImagePullPolicy,
	}
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

func (hc *HelmPack) getJobValues(container logic2.ContainerV2) []map[string]interface{} {
	shells := make([]map[string]interface{}, 0)
	for _, item := range container.Shells {
		shellWeight := 0
		hookName := ""
		switch item.Type {
		case "requireinstall":
			shellWeight = -5
			hookName = "pre-install"
		case "pre-install":
			shellWeight = -4
			hookName = "pre-install"
		case "install":
			shellWeight = -3
			hookName = "post-install"
		case "upgrade":
			shellWeight = -2
			hookName = "post-upgrade"
		case "uninstall":
			shellWeight = -1
			hookName = "post-delete"
		case "custom":
			shellWeight = 0
			hookName = "custom"
		}

		shells = append(shells, map[string]interface{}{
			"enabled": true,
			"title":   item.Title,
			"image":   item.Image,
			"shell":   item.Shell,
			"type":    hookName,
			"weight":  shellWeight,
			"name":    strings.ToLower(function.GetRandomStringNotContainerNumber(12)),
		})
	}

	return shells
}

func (hc *HelmPack) getBuildImageValues(container logic2.ContainerV2, application logic2.Application) []map[string]interface{} {
	if container.Image != "" || container.CodeAttachUrl == "" {
		return make([]map[string]interface{}, 0)
	}

	depot, _ := NewDepot()
	zipUrl, _ := depot.GetFormulaBackendZipDownloadUrlByApplication(application, false)

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

func (hc *HelmPack) getGpuValues(platform logic2.Platform) map[string]interface{} {
	if platform.Gpu != "" {
		return map[string]interface{}{
			"enable": true,
			"driver": platform.Gpu,
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
