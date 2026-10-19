package helm

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	copy2 "github.com/otiai10/copy"
	formulalogic "github.com/w7panel/w7panel-zpk/app/respo/logic/formula"
	"github.com/w7panel/w7panel-zpk/common/function"
	"sigs.k8s.io/yaml"
)

func (hc *HelmPack) packHelmApp(rootDir string) error {
	if err := hc.processHelmPkg(rootDir); err != nil {
		return err
	}
	return hc.configureHelmSidecarHost(rootDir)
}

func (hc *HelmPack) isHelmPackage() bool {
	return hc.Manifest.Platform.Helm.ChartName != "" ||
		hc.Manifest.Platform.Helm.Repository != "" ||
		len(hc.Manifest.Platform.Helm.DependYamls) > 0
}

func (hc *HelmPack) processHelmPkg(rootDir string) error {
	dependHelmYamlInHelm := false
	if hc.Manifest.Platform.Helm.ChartName != "" || hc.Manifest.Platform.Helm.Repository != "" {
		depot, _ := formulalogic.NewDepot()
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
		// charts/ 中可能已经包含 prepareRemoteHelmSidecars 提前准备的 Sidecar Chart。
		// 解包时只能合并目录，不可以删除整个 charts/，否则会丢失已准备好的 Sidecar。
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
				Name:        strings.ToLower(hc.Manifest.Application.Identifie),
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

	return hc.applyHelmPackageAffinity(rootDir)
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
	return mergeHelmValuesFile(rootDir, overrides)
}

func (hc *HelmPack) applyHelmPackageAffinity(rootDir string) error {
	affinity := hc.getSharedStorageWorkloadAffinityValues()
	if len(affinity) == 0 {
		return nil
	}
	return mergeHelmValuesFile(rootDir, map[string]interface{}{
		"affinity": affinity,
	})
}

func mergeHelmValuesFile(rootDir string, overrides map[string]interface{}) error {
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
	index, err := strconv.Atoi(part)
	return err == nil && index >= 0
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
