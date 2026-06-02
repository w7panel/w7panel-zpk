package function

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/w7panel/w7panel-zpk/common/service"
)

func ExtractZip(zipPath string, targetDir string) error {
	zipReader, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer zipReader.Close()

	for _, file := range zipReader.File {
		if file.FileInfo().IsDir() || strings.HasPrefix(file.Name, "__MACOSX") {
			continue
		}

		cleanName, err := CleanZipFilePath(file.Name)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(targetDir, filepath.FromSlash(cleanName))
		if !IsPathInDir(targetPath, targetDir) {
			return fmt.Errorf("非法文件路径: %s", file.Name)
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), os.ModePerm); err != nil {
			return err
		}

		src, err := file.Open()
		if err != nil {
			return err
		}

		mode := file.Mode()
		if mode == 0 {
			mode = service.FileMode
		}
		dst, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
		if err != nil {
			src.Close()
			return err
		}

		_, copyErr := io.Copy(dst, src)
		closeDstErr := dst.Close()
		closeSrcErr := src.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeDstErr != nil {
			return closeDstErr
		}
		if closeSrcErr != nil {
			return closeSrcErr
		}
	}

	return nil
}

func CleanZipFilePath(filePath string) (string, error) {
	normalizedPath := strings.ReplaceAll(filePath, "\\", "/")
	cleanPath := path.Clean(strings.TrimLeft(normalizedPath, "/"))
	if cleanPath == "." || cleanPath == ".." || strings.HasPrefix(cleanPath, "../") {
		return "", fmt.Errorf("非法文件路径: %s", filePath)
	}
	return cleanPath, nil
}
