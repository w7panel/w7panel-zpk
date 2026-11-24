package logic

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	const operation = "app_update"
	startedAt := time.Now()
	logInfo("tradition plugin operation started",
		"operation", operation,
		"site_dir", siteDir,
		"package_dir", packageDir,
		"state_dir", service.dataDir,
		"policy_plugin_count", len(policy.Plugins),
	)
	result, err := service.change(operation, siteDir, policy, func(state *State, _ string, previous map[string]struct{}) (OperationResult, error) {
		logInfo("capturing application base files",
			"operation", operation,
			"tracked_file_count", len(previous),
		)
		base, err := service.captureBase(nil, packageDir, previous, nil)
		if err != nil {
			return OperationResult{}, err
		}
		state.Base = base
		return OperationResult{Managed: true}, nil
	})
	service.logOperationResult(operation, startedAt, result, err)
	return result, err
}

// InstallPlugin 安装显式或自动受管的插件，并在首次覆盖某路径前保存该路径的原应用文件。
func (service *Installer) InstallPlugin(
	siteDir, packageDir, plugin string,
	policy Policy,
) (OperationResult, error) {
	const operation = "plugin_install"
	if plugin == "" {
		return OperationResult{}, fmt.Errorf("plugin identifier is required")
	}
	startedAt := time.Now()
	logInfo("tradition plugin operation started",
		"operation", operation,
		"plugin", plugin,
		"site_dir", siteDir,
		"package_dir", packageDir,
		"state_dir", service.dataDir,
		"policy_plugin_count", len(policy.Plugins),
	)
	result, err := service.change(operation, siteDir, policy, func(
		state *State,
		siteDir string,
		previous map[string]struct{},
	) (OperationResult, error) {
		_, managed := policy.Plugins[plugin]
		if !managed {
			packageRoot, err := sourceRoot(packageDir)
			if err != nil {
				return OperationResult{}, err
			}
			pluginFiles, err := treeFiles(packageRoot)
			if err != nil {
				return OperationResult{}, err
			}
			managed, err = hasSharedFiles(siteDir, pluginFiles)
			if err != nil {
				return OperationResult{}, err
			}
		}
		if managed {
			if _, configured := policy.Plugins[plugin]; !configured {
				logInfo("plugin is automatically managed because it shares files with the base",
					"operation", operation,
					"plugin", plugin,
				)
			}
		}
		if !managed {
			logInfo("plugin does not require file priority management",
				"operation", operation,
				"plugin", plugin,
			)
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
	service.logOperationResult(operation, startedAt, result, err, "plugin", plugin)
	return result, err
}

// UninstallPlugin 删除指定插件层，并使用剩余 OCI 层重建受管文件。
func (service *Installer) UninstallPlugin(
	siteDir, plugin string,
	policy Policy,
) (OperationResult, error) {
	const operation = "plugin_uninstall"
	if plugin == "" {
		return OperationResult{}, fmt.Errorf("plugin identifier is required")
	}
	startedAt := time.Now()
	logInfo("tradition plugin operation started",
		"operation", operation,
		"plugin", plugin,
		"site_dir", siteDir,
		"state_dir", service.dataDir,
		"policy_plugin_count", len(policy.Plugins),
	)
	result, err := service.change(operation, siteDir, policy, func(state *State, _ string, _ map[string]struct{}) (OperationResult, error) {
		if _, installed := state.Plugins[plugin]; !installed {
			_, managed := policy.Plugins[plugin]
			logInfo("plugin layer is not installed",
				"operation", operation,
				"plugin", plugin,
				"managed", managed,
			)
			return OperationResult{Managed: managed}, nil
		}
		delete(state.Plugins, plugin)
		delete(state.PluginFiles, plugin)
		return OperationResult{Managed: true}, nil
	})
	service.logOperationResult(operation, startedAt, result, err, "plugin", plugin)
	return result, err
}

// Restore 清理站点中的受管路径，再按 current manifest 的层顺序恢复文件。
func (service *Installer) Restore(siteDir string) (RecoveryResult, error) {
	const operation = "restore"
	startedAt := time.Now()
	logInfo("tradition plugin operation started",
		"operation", operation,
		"site_dir", siteDir,
		"state_dir", service.dataDir,
	)
	if err := service.ensureDataDir(); err != nil {
		logError("tradition plugin operation failed", "operation", operation, "error", err)
		return RecoveryResult{}, err
	}
	unlock, err := service.lock(true)
	if err != nil {
		logError("tradition plugin operation failed", "operation", operation, "error", err)
		return RecoveryResult{}, err
	}
	defer unlock()
	state, manifest, err := service.loadTag(currentTag)
	if err != nil {
		logError("tradition plugin operation failed", "operation", operation, "error", err)
		return RecoveryResult{}, err
	}
	logInfo("tradition plugin state loaded",
		"operation", operation,
		"has_base", state.Base != nil,
		"managed_plugin_count", len(state.Plugins),
		"tracked_file_count", len(trackedFiles(state.PluginFiles)),
		"layer_count", len(manifest.Layers),
	)
	if state.Base == nil && len(state.Plugins) == 0 {
		logInfo("tradition plugin restore skipped because state is empty",
			"operation", operation,
			"elapsed", time.Since(startedAt),
		)
		return RecoveryResult{Recovered: false}, nil
	}
	siteDir, err = targetRoot(siteDir)
	if err != nil {
		logError("tradition plugin operation failed", "operation", operation, "error", err)
		return RecoveryResult{}, err
	}
	materialized, err := service.materializeLayers(manifest.Layers)
	if err != nil {
		logError("tradition plugin operation failed", "operation", operation, "error", err)
		return RecoveryResult{}, err
	}
	defer os.RemoveAll(materialized)
	_, cleanup, err := service.replaceSiteFromTree(siteDir, materialized, trackedFiles(state.PluginFiles))
	if err != nil {
		logError("tradition plugin operation failed", "operation", operation, "error", err)
		return RecoveryResult{}, err
	}
	defer cleanup()
	logInfo("tradition plugin operation completed",
		"operation", operation,
		"recovered", true,
		"elapsed", time.Since(startedAt),
	)
	return RecoveryResult{Recovered: true}, nil
}

// change 统一完成加锁、状态变更、OCI 保存以及站点受管文件的重新应用。
func (service *Installer) change(
	operation string,
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
	logInfo("tradition plugin state loaded",
		"operation", operation,
		"has_base", state.Base != nil,
		"managed_plugin_count", len(state.Plugins),
		"tracked_file_count", len(previous),
	)
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
	if lastPluginRemoved {
		state.Base = nil
		storedLayers = nil
	}
	logInfo("tradition plugin site update planned",
		"operation", operation,
		"site_layer_count", len(siteLayers),
		"stored_layer_count", len(storedLayers),
		"previous_file_count", len(previous),
		"desired_file_count", len(desired),
		"removed_files", removedFiles,
		"base_pruned", basePruned,
		"last_plugin_removed", lastPluginRemoved,
	)
	materialized, err := service.materializeLayers(layerDescriptors(siteLayers))
	if err != nil {
		return OperationResult{}, err
	}
	defer os.RemoveAll(materialized)
	rollback, cleanup, err := service.replaceSiteFromTree(siteDir, materialized, previous)
	if err != nil {
		return OperationResult{}, err
	}
	defer cleanup()
	previousManifest, currentManifest, err := service.saveState(state, storedLayers)
	if err != nil {
		logWarn("OCI state save failed; rolling back site files",
			"operation", operation,
			"error", err,
		)
		if rollbackErr := rollback(); rollbackErr != nil {
			logError("site file rollback failed",
				"operation", operation,
				"error", rollbackErr,
			)
			return OperationResult{}, errors.Join(err, fmt.Errorf("restore site after OCI save failure: %w", rollbackErr))
		}
		logInfo("site files rolled back after OCI save failure", "operation", operation)
		return OperationResult{}, err
	}
	logInfo("tradition plugin state committed",
		"operation", operation,
		"manifest_digest", currentManifest.Digest.String(),
		"managed_plugin_count", len(state.Plugins),
		"stored_layer_count", len(storedLayers),
	)
	service.cleanupPreviousState(previousManifest, currentManifest)
	return result, nil
}

func (service *Installer) logOperationResult(
	operation string,
	startedAt time.Time,
	result OperationResult,
	err error,
	args ...any,
) {
	fields := []any{
		"operation", operation,
		"managed", result.Managed,
		"elapsed", time.Since(startedAt),
	}
	fields = append(fields, args...)
	if err != nil {
		fields = append(fields, "error", err)
		logError("tradition plugin operation failed", fields...)
		return
	}
	logInfo("tradition plugin operation completed", fields...)
}

// materializeLayers 先在临时目录应用并校验全部 OCI 层，避免损坏层写入实际站点。
func (service *Installer) materializeLayers(layers []v1.Descriptor) (string, error) {
	directory, err := os.MkdirTemp(filepath.Join(service.dataDir, "tmp"), "materialized-*")
	if err != nil {
		return "", err
	}
	if err := service.applyLayers(directory, layers); err != nil {
		os.RemoveAll(directory)
		return "", err
	}
	return directory, nil
}

// replaceSiteFromTree 备份本次涉及的站点路径，再以具象化目录替换；返回失败时使用的回滚函数。
func (service *Installer) replaceSiteFromTree(
	siteDir, materialized string,
	previous map[string]struct{},
) (rollback func() error, cleanup func(), err error) {
	materializedFiles, err := treeFiles(materialized)
	if err != nil {
		return nil, nil, err
	}
	affected := unionSets(previous, stringSet(materializedFiles))
	logInfo("replacing managed site files",
		"site_dir", siteDir,
		"previous_file_count", len(previous),
		"materialized_file_count", len(materializedFiles),
		"affected_file_count", len(affected),
	)
	backup, err := os.MkdirTemp(filepath.Join(service.dataDir, "tmp"), "backup-*")
	if err != nil {
		return nil, nil, err
	}
	cleanup = func() { _ = os.RemoveAll(backup) }
	if err := copySelected(siteDir, backup, affected, nil); err != nil {
		cleanup()
		return nil, nil, err
	}
	backupFiles, err := treeFiles(backup)
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	rollback = func() error {
		logWarn("rolling back managed site files",
			"site_dir", siteDir,
			"affected_file_count", len(affected),
			"backup_file_count", len(backupFiles),
		)
		if err := removePaths(siteDir, sortedPaths(affected, false)); err != nil {
			return err
		}
		return copySelected(backup, siteDir, stringSet(backupFiles), nil)
	}
	if err := removePaths(siteDir, sortedPaths(affected, false)); err != nil {
		cleanup()
		return nil, nil, err
	}
	if err := copySelected(materialized, siteDir, stringSet(materializedFiles), nil); err != nil {
		if rollbackErr := rollback(); rollbackErr != nil {
			err = errors.Join(err, fmt.Errorf("restore site after apply failure: %w", rollbackErr))
		}
		cleanup()
		return nil, nil, err
	}
	logInfo("managed site files replaced",
		"site_dir", siteDir,
		"materialized_file_count", len(materializedFiles),
	)
	return rollback, cleanup, nil
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

// hasSharedFiles 判断目录中是否存在给定列表中的任一文件。
func hasSharedFiles(directory string, files []string) (bool, error) {
	directoryFiles, err := treeFiles(directory)
	if err != nil {
		return false, err
	}
	directoryFileSet := stringSet(directoryFiles)
	for _, name := range files {
		if _, exists := directoryFileSet[name]; exists {
			return true, nil
		}
	}
	return false, nil
}

// unionSets 返回两个路径集合的并集。
func unionSets(left, right map[string]struct{}) map[string]struct{} {
	result := make(map[string]struct{}, len(left)+len(right))
	for name := range left {
		result[name] = struct{}{}
	}
	for name := range right {
		result[name] = struct{}{}
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
