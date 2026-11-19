package function

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type PathInfoOut struct {
	DirName   string
	BaseName  string
	Extension string
	Filename  string
}

func GetPathInfo(path string) PathInfoOut {
	filename := filepath.Base(path)
	ext := filepath.Ext(filename)

	dirname, basename := filepath.Split(path)
	basename = basename[:len(basename)-len(ext)]
	result := PathInfoOut{}
	result.DirName = dirname
	result.BaseName = basename
	result.Extension = ext
	result.Filename = filename
	return result
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return true
}

func CreateDirIfNotExist(dirName string, perm os.FileMode) {
	if _, err := os.Stat(dirName); os.IsNotExist(err) {
		err := os.MkdirAll(dirName, perm)
		if err != nil {
			panic(err)
		}
	}
}

func IsDirEmpty(path string) bool {
	// 检查路径是否为目录
	fileInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return true
		}
		return true
	}
	if !fileInfo.IsDir() {
		return true
	}

	// 读取目录条目
	dir, err := os.Open(path)
	if err != nil {
		return true
	}
	defer dir.Close()

	entries, err := dir.Readdir(0)
	if err != nil {
		return true
	}

	return len(entries) == 0
}

func IsEmptyFile(filePath string) bool {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return true
	}

	// 检查文件大小
	if fileInfo.Size() == 0 {
		return true
	}
	return false
}

func AppendBytesToFile(filename string, data []byte) error {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(data)
	return err
}

func IsPathInDir(filePath string, dir string) bool {
	cleanFilePath := filepath.Clean(filePath)
	cleanDir := filepath.Clean(dir)
	return strings.HasPrefix(cleanFilePath, cleanDir+string(os.PathSeparator))
}

// RelativePath permits dotfiles but rejects absolute paths and traversal.
func RelativePath(name string) (string, error) {
	for strings.HasPrefix(name, "./") {
		name = strings.TrimPrefix(name, "./")
	}
	name = strings.TrimSuffix(name, "/")
	if name == "" || name == "." || path.Clean(name) != name || name == ".." || strings.HasPrefix(name, "../") || strings.HasPrefix(name, "/") || strings.ContainsAny(name, "\\:\x00") {
		return "", fmt.Errorf("相对路径无效: %q", name)
	}
	return name, nil
}

// ResolveLocalPath refuses symlink ancestors before reading or replacing files.
func ResolveLocalPath(root, name string) (string, error) {
	name, err := RelativePath(name)
	if err != nil {
		return "", err
	}
	current := root
	for _, part := range append([]string{""}, strings.Split(name, "/")...) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			break
		}
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("不支持符号链接: %s", current)
		}
	}
	return filepath.Join(root, filepath.FromSlash(name)), nil
}

func ReadFile(filename string, maxSize int64) ([]byte, error) {
	info, err := os.Lstat(filename)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > maxSize {
		return nil, fmt.Errorf("输入必须为大小符合限制的普通文件: %s", filename)
	}
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, maxSize+1))
	if err == nil && int64(len(content)) > maxSize {
		return nil, fmt.Errorf("输入文件超过大小限制: %s", filename)
	}
	return content, err
}

func WriteFile(filename string, content []byte, mode fs.FileMode) error {
	return writeStream(filename, bytes.NewReader(content), int64(len(content)), mode)
}

func CopyFile(source, destination string, mode fs.FileMode, maxSize int64) error {
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() > maxSize {
		return fmt.Errorf("输入必须为大小符合限制的普通文件: %s", source)
	}
	file, err := os.Open(source)
	if err != nil {
		return err
	}
	defer file.Close()
	return writeStream(destination, file, info.Size(), mode)
}

func writeStream(filename string, reader io.Reader, size int64, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(filename), "."+filepath.Base(filename)+".tmp-")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	n, copyErr := io.Copy(file, io.LimitReader(reader, size+1))
	if copyErr == nil && n != size {
		copyErr = fmt.Errorf("写入时文件大小发生变化")
	}
	if err = errors.Join(copyErr, file.Chmod(mode), file.Close()); err != nil {
		return err
	}
	return os.Rename(file.Name(), filename)
}
