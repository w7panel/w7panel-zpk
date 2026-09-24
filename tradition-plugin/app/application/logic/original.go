package logic

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
)

// captureBase 从 sourceDir 中提取 selected 指定、且不在 excluded 中的原应用文件，
// 与现有 Base 合并后重新打包。这里只按文件路径提取，不比较文件内容；源目录中不存在的文件会被忽略。
func (service *Installer) captureBase(
	current *Layer,
	sourceDir string,
	selected, excluded map[string]struct{},
) (*Layer, error) {
	if len(selected) == 0 {
		return current, nil
	}
	tree, err := service.baseTree(current)
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tree)
	if err := copySelected(sourceDir, tree, selected, excluded); err != nil {
		return nil, err
	}
	return service.packBase(tree)
}

// pruneBase 仅保留剩余受管插件仍会使用的原应用文件。
// desired 是所有剩余受管插件文件路径的并集。
func (service *Installer) pruneBase(
	current *Layer,
	desired map[string]struct{},
) (*Layer, error) {
	if current == nil {
		return nil, nil
	}
	tree, err := service.baseTree(current)
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tree)
	files, err := treeFiles(tree)
	if err != nil {
		return nil, err
	}
	for index := len(files) - 1; index >= 0; index-- {
		name := files[index]
		if _, keep := desired[name]; keep {
			continue
		}
		if err := os.Remove(filepath.Join(tree, filepath.FromSlash(name))); err != nil && !os.IsNotExist(err) {
			return nil, err
		}
	}
	return service.packBase(tree)
}

// baseTree 创建 Base 临时目录，并在存在当前 Base 时将其内容展开到该目录。
func (service *Installer) baseTree(current *Layer) (string, error) {
	tempDir, err := os.MkdirTemp(filepath.Join(service.dataDir, "tmp"), "base-*")
	if err != nil {
		return "", err
	}
	if current != nil {
		if err := service.applyLayer(tempDir, current.Blob); err != nil {
			os.RemoveAll(tempDir)
			return "", err
		}
	}
	return tempDir, nil
}

// packBase 将临时 Base 目录打包为 OCI 层；目录为空时返回 nil。
func (service *Installer) packBase(baseDir string) (*Layer, error) {
	layer, err := service.packLayer(baseDir, "base")
	if err != nil {
		return nil, err
	}
	if len(layer.Files) == 0 {
		return nil, nil
	}
	return &layer, nil
}

// copySelected 将选中的源路径复制到目标目录，并跳过 excluded 中的路径。
func copySelected(
	sourceDir, targetDir string,
	selected, excluded map[string]struct{},
) error {
	sourceDir, err := sourceRoot(sourceDir)
	if err != nil {
		return err
	}
	for _, name := range sortedPaths(selected, false) {
		if _, exists := excluded[name]; exists {
			continue
		}
		if err := copyPath(sourceDir, targetDir, name); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// copyPath 按原类型和权限复制一个目录、普通文件或符号链接。
func copyPath(sourceDir, targetDir, name string) error {
	from := filepath.Join(sourceDir, filepath.FromSlash(name))
	info, err := os.Lstat(from)
	if err != nil {
		return err
	}
	to := filepath.Join(targetDir, filepath.FromSlash(name))
	switch {
	case info.IsDir():
		return replaceDirectory(to, info.Mode().Perm())
	case info.Mode().IsRegular():
		file, err := os.Open(from)
		if err != nil {
			return err
		}
		defer file.Close()
		return replaceFile(to, file, info.Mode().Perm())
	case info.Mode()&os.ModeSymlink != 0:
		link, err := os.Readlink(from)
		if err != nil {
			return err
		}
		return replaceSymlink(to, link)
	default:
		return fmt.Errorf("unsupported file %q", name)
	}
}

// treeFiles 返回目录树中所有条目的相对路径，并按名称稳定排序。
func treeFiles(directory string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(directory, func(name string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if name == directory {
			return nil
		}
		relative, err := filepath.Rel(directory, name)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(relative))
		return nil
	})
	sort.Strings(files)
	return files, err
}

// removePaths 按由深到浅的顺序删除受管路径，并保留仍包含其他内容的目录。
func removePaths(siteDir string, previous []string) error {
	paths := make(map[string]struct{}, len(previous))
	for _, name := range previous {
		paths[name] = struct{}{}
	}
	for _, name := range sortedPaths(paths, true) {
		if err := os.Remove(filepath.Join(siteDir, filepath.FromSlash(name))); err != nil &&
			!os.IsNotExist(err) && !errors.Is(err, syscall.ENOTEMPTY) {
			return err
		}
	}
	return nil
}

// replaceDirectory 确保目标路径为具有指定权限的目录。
func replaceDirectory(name string, mode os.FileMode) error {
	info, err := os.Lstat(name)
	if err == nil && !info.IsDir() {
		if err := os.Remove(name); err != nil {
			return err
		}
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(name, mode); err != nil {
		return err
	}
	return os.Chmod(name, mode)
}

// replaceFile 使用 reader 的内容和指定权限原子替换目标文件。
func replaceFile(name string, reader io.Reader, mode os.FileMode) error {
	if info, err := os.Lstat(name); err == nil && info.IsDir() {
		if err := os.Remove(name); err != nil {
			return err
		}
	}
	return atomicWrite(name, reader, mode)
}

// replaceSymlink 将目标路径替换为指向 target 的符号链接。
func replaceSymlink(name, target string) error {
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		return err
	}
	if err := os.RemoveAll(name); err != nil {
		return err
	}
	return os.Symlink(target, name)
}

// atomicWrite 先写入同目录临时文件，再通过重命名原子替换目标文件。
func atomicWrite(name string, reader io.Reader, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(name), ".w7-tradition-plugin-*")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	if err := temp.Chmod(mode); err != nil {
		temp.Close()
		return err
	}
	if _, err := io.Copy(temp, reader); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(temp.Name(), name)
}

// sourceRoot 返回已解析符号链接的源目录绝对路径，并校验目录必须已存在。
func sourceRoot(name string) (string, error) {
	absPath, err := filepath.Abs(strings.TrimSpace(name))
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("source must be a directory")
	}
	return resolved, nil
}

// targetRoot 创建目标目录并返回已解析符号链接的绝对路径。
func targetRoot(name string) (string, error) {
	absPath, err := filepath.Abs(strings.TrimSpace(name))
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(absPath, 0o755); err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(absPath)
}

// sortedPaths 按目录层级排序路径；reverse 为 true 时子路径排在父路径之前。
func sortedPaths(set map[string]struct{}, reverse bool) []string {
	paths := make([]string, 0, len(set))
	for name := range set {
		paths = append(paths, name)
	}
	sort.Slice(paths, func(i, j int) bool {
		left, right := strings.Count(paths[i], "/"), strings.Count(paths[j], "/")
		if left == right {
			return paths[i] < paths[j]
		}
		if reverse {
			return left > right
		}
		return left < right
	})
	return paths
}
