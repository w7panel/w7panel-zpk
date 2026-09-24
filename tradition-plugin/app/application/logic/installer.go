package logic

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

type Layer struct {
	Name  string
	Blob  v1.Descriptor
	Files []string
}

type Policy struct {
	Plugins map[string]int
}

type State struct {
	PluginFiles map[string][]string `json:"pluginFiles"`
	Base        *Layer              `json:"-"`
	Plugins     map[string]Layer    `json:"-"`
}

type OperationResult struct {
	Managed bool `json:"managed"`
}

type RecoveryResult struct {
	Recovered bool `json:"recovered"`
}

type Installer struct {
	dataDir string
}

// NewInstaller 创建使用指定 OCI 数据目录的传统应用插件安装器。
func NewInstaller(dataDir string) *Installer {
	return &Installer{dataDir: filepath.Clean(strings.TrimSpace(dataDir))}
}

// UpdateApplication 从新应用包重新提取所有受管路径的原文件，并按当前优先级重放插件层。
func (service *Installer) UpdateApplication(
	siteDir, packageDir string,
	policy Policy,
) (OperationResult, error) {
	return service.change(siteDir, policy, func(state *State, _ string, previous map[string]struct{}) (OperationResult, error) {
		base, err := service.captureBase(nil, packageDir, previous, nil)
		if err != nil {
			return OperationResult{}, err
		}
		state.Base = base
		return OperationResult{Managed: true}, nil
	})
}

// InstallPlugin 安装受策略管理的插件，并在首次覆盖某路径前保存该路径的原应用文件。
func (service *Installer) InstallPlugin(
	siteDir, packageDir, plugin string,
	policy Policy,
) (OperationResult, error) {
	if plugin == "" {
		return OperationResult{}, fmt.Errorf("plugin identifier is required")
	}
	return service.change(siteDir, policy, func(
		state *State,
		siteDir string,
		previous map[string]struct{},
	) (OperationResult, error) {
		_, managed := policy.Plugins[plugin]
		if !managed {
			return OperationResult{Managed: false}, nil
		}
		layer, err := service.packLayer(packageDir, plugin)
		if err != nil {
			return OperationResult{}, err
		}
		state.Base, err = service.captureBase(
			state.Base,
			siteDir,
			stringSet(layer.Files),
			previous,
		)
		if err != nil {
			return OperationResult{}, err
		}
		state.Plugins[plugin] = layer
		state.PluginFiles[plugin] = layer.Files
		return OperationResult{Managed: true}, nil
	})
}

// UninstallPlugin 删除指定插件层，并使用剩余 OCI 层重建受管文件。
func (service *Installer) UninstallPlugin(
	siteDir, plugin string,
	policy Policy,
) (OperationResult, error) {
	if plugin == "" {
		return OperationResult{}, fmt.Errorf("plugin identifier is required")
	}
	return service.change(siteDir, policy, func(state *State, _ string, _ map[string]struct{}) (OperationResult, error) {
		if _, installed := state.Plugins[plugin]; !installed {
			_, managed := policy.Plugins[plugin]
			return OperationResult{Managed: managed}, nil
		}
		delete(state.Plugins, plugin)
		delete(state.PluginFiles, plugin)
		return OperationResult{Managed: true}, nil
	})
}

// Restore 清理站点中的受管路径，再按 current manifest 的层顺序恢复文件。
func (service *Installer) Restore(siteDir string) (RecoveryResult, error) {
	if err := service.ensureDataDir(); err != nil {
		return RecoveryResult{}, err
	}
	unlock, err := service.lock(true)
	if err != nil {
		return RecoveryResult{}, err
	}
	defer unlock()
	state, manifest, err := service.loadTag(currentTag)
	if err != nil {
		return RecoveryResult{}, err
	}
	if state.Base == nil && len(state.Plugins) == 0 {
		return RecoveryResult{Recovered: false}, nil
	}
	siteDir, err = targetRoot(siteDir)
	if err != nil {
		return RecoveryResult{}, err
	}
	if err := removePaths(siteDir, sortedPaths(trackedFiles(state.PluginFiles), false)); err != nil {
		return RecoveryResult{}, err
	}
	if err := service.applyLayers(siteDir, manifest.Layers); err != nil {
		return RecoveryResult{}, err
	}
	return RecoveryResult{Recovered: true}, nil
}

// change 统一完成加锁、状态变更、OCI 保存以及站点受管文件的重新应用。
func (service *Installer) change(
	siteDir string,
	policy Policy,
	update func(*State, string, map[string]struct{}) (OperationResult, error),
) (OperationResult, error) {
	if err := service.ensureDataDir(); err != nil {
		return OperationResult{}, err
	}
	unlock, err := service.lock(false)
	if err != nil {
		return OperationResult{}, err
	}
	defer unlock()
	state, err := service.loadState()
	if err != nil {
		return OperationResult{}, err
	}
	previous := trackedFiles(state.PluginFiles)
	siteDir, err = targetRoot(siteDir)
	if err != nil {
		return OperationResult{}, err
	}
	result, err := update(&state, siteDir, previous)
	if err != nil {
		return OperationResult{}, err
	}
	if !result.Managed {
		return result, nil
	}
	lastPluginRemoved := state.Base != nil && len(state.Plugins) == 0
	desired := trackedFiles(state.PluginFiles)
	// 站点使用裁剪前的 Base 恢复退出管理的原文件，OCI 仅保存裁剪后的 Base。
	siteLayers, err := orderedLayers(state, policy)
	if err != nil {
		return OperationResult{}, err
	}
	removedFiles := hasRemovedFiles(previous, desired)
	basePruned := state.Base != nil && !lastPluginRemoved && removedFiles
	if basePruned {
		if err := service.cleanupBase(&state, desired); err != nil {
			return OperationResult{}, err
		}
	}
	storedLayers := siteLayers
	if basePruned {
		storedLayers, err = orderedLayers(state, policy)
		if err != nil {
			return OperationResult{}, err
		}
	}
	if err := removePaths(siteDir, sortedPaths(previous, false)); err != nil {
		return OperationResult{}, err
	}
	if err := service.applyLayers(siteDir, layerDescriptors(siteLayers)); err != nil {
		return OperationResult{}, err
	}
	if lastPluginRemoved {
		state.Base = nil
		storedLayers = nil
	}
	if err := service.saveState(state, storedLayers); err != nil {
		return OperationResult{}, err
	}
	return result, nil
}

// cleanupBase 删除 Base 中已不再被任何受管插件使用的原应用文件。
func (service *Installer) cleanupBase(state *State, desired map[string]struct{}) error {
	base, err := service.pruneBase(state.Base, desired)
	if err != nil {
		return err
	}
	state.Base = base
	return nil
}

// trackedFiles 返回所有插件文件路径的并集。
func trackedFiles(plugins map[string][]string) map[string]struct{} {
	files := map[string]struct{}{}
	for _, pluginFiles := range plugins {
		for _, name := range pluginFiles {
			files[name] = struct{}{}
		}
	}
	return files
}

// stringSet 将文件路径列表转换为便于查找和去重的集合。
func stringSet(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

// hasRemovedFiles 判断新的受管路径集合是否移除了原有路径。
func hasRemovedFiles(previous, desired map[string]struct{}) bool {
	for name := range previous {
		if _, exists := desired[name]; !exists {
			return true
		}
	}
	return false
}

// ensureDataDir 检查数据目录并初始化 OCI Layout 及临时目录。
func (service *Installer) ensureDataDir() error {
	if service.dataDir == "" || service.dataDir == "." {
		return fmt.Errorf("data directory is required")
	}
	if err := os.MkdirAll(filepath.Join(service.dataDir, "tmp"), 0o700); err != nil {
		return err
	}
	_, err := service.store()
	return err
}

// lock 创建进程间操作锁；force 为 true 时会先移除现有锁文件。
func (service *Installer) lock(force bool) (func(), error) {
	path := filepath.Join(service.dataDir, "lock")
	if force {
		_ = os.Remove(path)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("another file layer operation is running")
	}
	_ = file.Close()
	return func() { _ = os.Remove(path) }, nil
}
