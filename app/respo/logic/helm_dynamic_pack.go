package logic

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/w7panel/w7panel-zpk/common/function"
	"golang.org/x/sync/singleflight"
)

var dynamicHelmPackageGroup singleflight.Group

type DynamicHelmPackageOption func(*dynamicHelmPackageOptions) error

type dynamicHelmPackageOptions struct {
	cacheParts [][]byte
	transforms []func(chartDir string) error
}

func (options *dynamicHelmPackageOptions) addTransform(cacheValue interface{}, transform func(chartDir string) error) error {
	cachePart, err := json.Marshal(cacheValue)
	if err != nil {
		return fmt.Errorf("序列化动态 Helm 配置失败: %w", err)
	}
	options.cacheParts = append(options.cacheParts, cachePart)
	options.transforms = append(options.transforms, transform)
	return nil
}

func resolveDynamicHelmPackageOptions(options ...DynamicHelmPackageOption) (*dynamicHelmPackageOptions, error) {
	resolved := &dynamicHelmPackageOptions{
		cacheParts: make([][]byte, 0, len(options)),
		transforms: make([]func(chartDir string) error, 0, len(options)),
	}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(resolved); err != nil {
			return nil, err
		}
	}
	return resolved, nil
}

func BuildDynamicHelmPackage(helmPath string, options ...DynamicHelmPackageOption) (string, error) {
	resolvedOptions, err := resolveDynamicHelmPackageOptions(options...)
	if err != nil {
		return "", err
	}
	cacheKey, err := dynamicHelmCacheKey(helmPath, resolvedOptions.cacheParts)
	if err != nil {
		return "", err
	}
	dynamicPackagePath := dynamicHelmPackagePath(helmPath, cacheKey)
	_, err, _ = dynamicHelmPackageGroup.Do(cacheKey, func() (interface{}, error) {
		if function.FileExists(dynamicPackagePath) {
			return dynamicPackagePath, nil
		}

		buildDir, err := os.MkdirTemp(filepath.Dir(helmPath), ".build-")
		if err != nil {
			return "", fmt.Errorf("创建动态 Helm 构建目录失败: %w", err)
		}
		defer os.RemoveAll(buildDir)

		workDir := filepath.Join(buildDir, "work")
		if err = function.UnzipHelmPackage(helmPath, workDir); err != nil {
			return "", fmt.Errorf("解压 Helm 包失败: %w", err)
		}
		chartName, err := readHelmChartName(workDir)
		if err != nil {
			return "", err
		}
		chartDir := filepath.Join(buildDir, chartName)
		if err = os.Rename(workDir, chartDir); err != nil {
			return "", fmt.Errorf("准备动态 Helm Chart 目录失败: %w", err)
		}
		for _, transform := range resolvedOptions.transforms {
			if err = transform(chartDir); err != nil {
				return "", err
			}
		}
		temporaryPackagePath := filepath.Join(buildDir, filepath.Base(dynamicPackagePath))
		if err = function.ZipHelmChart(chartDir, temporaryPackagePath); err != nil {
			return "", fmt.Errorf("打包动态 Helm 包失败: %w", err)
		}
		if err = os.Rename(temporaryPackagePath, dynamicPackagePath); err != nil {
			return "", fmt.Errorf("保存动态 Helm 缓存失败: %w", err)
		}
		return dynamicPackagePath, nil
	})
	if err != nil {
		return "", err
	}

	return dynamicPackagePath, nil
}

func dynamicHelmCacheKey(helmPath string, cacheParts [][]byte) (string, error) {
	fileInfo, err := os.Stat(helmPath)
	if err != nil {
		return "", fmt.Errorf("读取 Helm 包信息失败: %w", err)
	}
	hasher := sha256.New()
	_, _ = fmt.Fprintf(hasher, "%s\n%d\n%d\n", helmPath, fileInfo.Size(), fileInfo.ModTime().UnixNano())
	for _, cachePart := range cacheParts {
		_, _ = hasher.Write(cachePart)
		_, _ = hasher.Write([]byte{'\n'})
	}
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

func dynamicHelmPackagePath(helmPath, cacheKey string) string {
	extension := filepath.Ext(helmPath)
	return strings.TrimSuffix(helmPath, extension) + "-" + cacheKey + extension
}
