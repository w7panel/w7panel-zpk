package function

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

func GetFileMD5(filePath string) (string, error) {
	// 1. 以只读方式打开文件
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close() // 确保函数结束时关闭文件

	// 2. 创建一个新的 MD5 哈希计算器
	hash := md5.New()

	// 3. 使用 io.Copy 将文件内容流式写入哈希计算器
	// 这种方式会自动分块读取，即使文件有几个 GB，内存占用也极低
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	// 4. 获取最终的哈希字节切片，并转换为十六进制字符串
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
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
