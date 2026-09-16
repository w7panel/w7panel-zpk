package function

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func UnzipHelmPackage(archivePath, targetDir string) error {
	// 打开压缩文件
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	// 创建 gzip 阅读器
	gzr, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("创建 gzip 阅读器失败: %w", err)
	}
	defer gzr.Close()

	var topLevelDir string
	isFirstItem := true

	// 创建 tar 阅读器
	tr := tar.NewReader(gzr)

	// 遍历 tar 文件中的每个条目
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break // 文件结束
		}
		if err != nil {
			return fmt.Errorf("读取 tar 条目失败: %w", err)
		}

		// 记录第一个条目的父目录（跳过可能的空路径）
		if isFirstItem && header.Name != "" {
			// 获取顶层目录（第一个路径组件）
			parts := strings.SplitN(strings.Trim(header.Name, "/"), "/", 2)
			if len(parts) > 0 && parts[0] != "" {
				topLevelDir = parts[0] + "/"
			}
			isFirstItem = false
		}
		// 如果找到了顶层目录前缀，则移除它
		var targetName string
		if topLevelDir != "" && strings.HasPrefix(header.Name, topLevelDir) {
			targetName = strings.TrimPrefix(header.Name, topLevelDir)
		} else {
			targetName = header.Name
		}

		// 如果目标名为空，表示是整个顶层目录（跳过它）
		if targetName == "" || targetName == "./" {
			continue
		}

		// 安全检查
		targetPath := filepath.Join(targetDir, targetName)
		if !strings.HasPrefix(targetPath, filepath.Clean(targetDir)+string(os.PathSeparator)) {
			return fmt.Errorf("非法文件路径: %s", targetName)
		}

		// 根据文件类型处理
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return fmt.Errorf("创建目录失败: %w", err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("创建父目录失败: %w", err)
			}

			//os.O_TRUNC：打开已存在文件时把文件长度截断为 0
			outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("创建文件失败: %w", err)
			}

			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return fmt.Errorf("复制文件内容失败: %w", err)
			}
			outFile.Close()
		case tar.TypeSymlink, tar.TypeLink:
			// Links can escape targetDir or redirect subsequent writes outside it.
			return fmt.Errorf("Helm 包不允许包含链接: %s", header.Name)
		}
	}

	return nil
}

// ZipHelmChart - 将 Chart 目录打包成 Helm Chart (.tgz)
func ZipHelmChart(sourceDir, outputFilePath string) error {
	// 验证源目录是否存在
	sourceInfo, err := os.Stat(sourceDir)
	if err != nil {
		return fmt.Errorf("读取源目录失败: %w", err)
	}
	if !sourceInfo.IsDir() {
		return fmt.Errorf("Helm Chart 源路径不是目录: %s", sourceDir)
	}

	// 确保输出文件路径以 .tgz 结尾
	if !strings.HasSuffix(outputFilePath, ".tgz") {
		outputFilePath += ".tgz"
	}

	// 获取目录名作为顶层目录名（用于 tar 包内路径）
	chartName := filepath.Base(sourceDir)

	// 先在目标目录内生成临时文件，完整写入后再通过 Rename 原子发布。
	// 下载请求因此只会看到旧的完整包或新的完整包，不会读到半成品。
	if err := os.MkdirAll(filepath.Dir(outputFilePath), 0o755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}
	outFile, err := os.CreateTemp(filepath.Dir(outputFilePath), "."+filepath.Base(outputFilePath)+".tmp-")
	if err != nil {
		return fmt.Errorf("创建临时输出文件失败: %w", err)
	}
	temporaryOutputPath := outFile.Name()
	defer os.Remove(temporaryOutputPath)

	// 创建 gzip 写入器
	gzw := gzip.NewWriter(outFile)

	// 创建 tar 写入器
	tw := tar.NewWriter(gzw)

	// 遍历源目录并添加文件到 tar 包
	err = filepath.Walk(sourceDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && !info.Mode().IsRegular() {
			return fmt.Errorf("Helm Chart 不允许包含特殊文件: %s", filePath)
		}

		// 创建 tar 头信息
		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return fmt.Errorf("创建 tar 头失败: %w", err)
		}

		// 计算 tar 包内的路径（添加顶层目录前缀）
		relPath, err := filepath.Rel(sourceDir, filePath)
		if err != nil {
			return fmt.Errorf("计算相对路径失败: %w", err)
		}

		// 在 tar 包内路径前加上顶层目录名
		header.Name = filepath.Join(chartName, relPath)

		// 确保目录路径以 "/" 结尾（tar 标准要求）
		if info.IsDir() && !strings.HasSuffix(header.Name, "/") {
			header.Name += "/"
		}

		// 设置修改时间为当前时间（可选的，Helm 包通常不关心这个）
		header.ModTime = time.Now()
		header.Format = tar.FormatPAX

		// 写入头信息
		if err := tw.WriteHeader(header); err != nil {
			return fmt.Errorf("写入 tar 头失败: %w", err)
		}

		// 如果是普通文件，写入文件内容
		if info.Mode().IsRegular() {
			file, err := os.Open(filePath)
			if err != nil {
				return fmt.Errorf("打开文件失败: %w", err)
			}
			_, copyErr := io.Copy(tw, file)
			closeErr := file.Close()
			if copyErr != nil {
				return fmt.Errorf("复制文件内容失败: %w", copyErr)
			}
			if closeErr != nil {
				return fmt.Errorf("关闭源文件失败: %w", closeErr)
			}
		}

		return nil
	})

	if err != nil {
		_ = tw.Close()
		_ = gzw.Close()
		_ = outFile.Close()
		return fmt.Errorf("遍历目录失败: %w", err)
	}
	if err := tw.Close(); err != nil {
		_ = gzw.Close()
		_ = outFile.Close()
		return fmt.Errorf("完成 tar 写入失败: %w", err)
	}
	if err := gzw.Close(); err != nil {
		_ = outFile.Close()
		return fmt.Errorf("完成 gzip 写入失败: %w", err)
	}
	if err := outFile.Sync(); err != nil {
		_ = outFile.Close()
		return fmt.Errorf("同步 Helm 包失败: %w", err)
	}
	if err := outFile.Close(); err != nil {
		return fmt.Errorf("关闭 Helm 包失败: %w", err)
	}
	if err := os.Chmod(temporaryOutputPath, 0o644); err != nil {
		return fmt.Errorf("设置 Helm 包权限失败: %w", err)
	}
	if err := os.Rename(temporaryOutputPath, outputFilePath); err != nil {
		return fmt.Errorf("发布 Helm 包失败: %w", err)
	}

	return nil
}
