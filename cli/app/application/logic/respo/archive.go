package respo

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/w7panel/w7panel-zpk/common/function"
	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
)

type attachmentField struct {
	kind, section, field, media, extension string
}

var attachmentFields = []attachmentField{
	{"frontend", "web", "url", commonlogic.MediaTypeFrontedCodeZip, ".zip"},
	{"backend", "source", "url", commonlogic.MediaTypeBackendCodeZip, ".zip"},
	{"helm", "platform.helm", "chartName", commonlogic.MediaTypeHelmZip, ".tgz"},
}

const (
	maxAttachmentSize = int64(8 << 30)
	maxExpandedSize   = int64(32 << 30)
	maxArchiveEntries = 100000
)

type extractionBudget struct {
	size    int64
	entries int
}

// walkAttachment shares ordered traversal and safety checks for packing and copying.
func walkAttachment(directory string, visit func(string, fs.FileInfo) error) error {
	var total int64
	count := 0
	err := filepath.WalkDir(directory, func(filename string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if filename == directory {
			return nil
		}
		name, err := filepath.Rel(directory, filename)
		if err != nil {
			return err
		}
		name, err = function.RelativePath(filepath.ToSlash(name))
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.IsDir() && !info.Mode().IsRegular() {
			return fmt.Errorf("附件不支持链接或特殊文件: %s", name)
		}
		count++
		total += info.Size()
		if count > maxArchiveEntries || info.Size() > maxAttachmentSize || total > maxExpandedSize {
			return fmt.Errorf("附件文件数量或大小超过限制")
		}
		return visit(name, info)
	})
	if err == nil && count == 0 {
		return fmt.Errorf("附件不能为空")
	}
	return err
}

func packAttachment(directory, filename string, field attachmentField) (err error) {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	// Ignore local mtimes, owners and platform-specific metadata for stable digests.
	epoch := time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)
	var zw *zip.Writer
	var tw *tar.Writer
	var gz *gzip.Writer
	if field.kind == "helm" {
		gz = gzip.NewWriter(file)
		tw = tar.NewWriter(gz)
		defer func() { err = errors.Join(err, tw.Close(), gz.Close()) }()
	} else {
		zw = zip.NewWriter(file)
		defer func() { err = errors.Join(err, zw.Close()) }()
	}
	return walkAttachment(directory, func(name string, info fs.FileInfo) error {
		var writer io.Writer
		if info.IsDir() {
			name += "/"
		}
		if tw != nil {
			header, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return err
			}
			header.Name, header.ModTime = name, epoch
			header.AccessTime, header.ChangeTime = time.Time{}, time.Time{}
			header.Uid, header.Gid, header.Uname, header.Gname = 0, 0, "", ""
			header.Mode = int64(info.Mode().Perm())
			if err = tw.WriteHeader(header); err != nil {
				return err
			}
			writer = tw
		} else {
			header := &zip.FileHeader{Name: name, Method: zip.Deflate}
			header.SetMode(info.Mode() & (fs.ModeDir | fs.ModePerm))
			header.SetModTime(epoch)
			var err error
			writer, err = zw.CreateHeader(header)
			if err != nil {
				return err
			}
		}
		if info.IsDir() {
			return nil
		}
		source, err := os.Open(filepath.Join(directory, filepath.FromSlash(name)))
		if err != nil {
			return err
		}
		n, copyErr := io.Copy(writer, io.LimitReader(source, info.Size()+1))
		closeErr := source.Close()
		if copyErr == nil && n != info.Size() {
			copyErr = fmt.Errorf("打包时文件大小发生变化: %s", name)
		}
		return errors.Join(copyErr, closeErr)
	})
}

// unpackAttachment always writes to a new, disposable directory. No links are
// accepted, and limits apply to expanded content as well as compressed bytes.
func unpackAttachment(filename, directory string, field attachmentField, budget *extractionBudget) error {
	if err := os.Mkdir(directory, 0755); err != nil {
		return err
	}
	seen := map[string]bool{}
	directoryModes := map[string]fs.FileMode{}
	write := func(name string, mode fs.FileMode, size int64, reader io.Reader) error {
		name, err := function.RelativePath(name)
		if err != nil {
			return err
		}
		if seen[name] {
			return fmt.Errorf("附件内部路径重复: %s", name)
		}
		seen[name] = true
		budget.size += size
		budget.entries++
		if budget.entries > maxArchiveEntries || size < 0 || size > maxAttachmentSize || budget.size > maxExpandedSize {
			return fmt.Errorf("附件解压数量或大小超过限制")
		}
		destination := filepath.Join(directory, filepath.FromSlash(name))
		if mode.IsDir() {
			directoryModes[destination] = mode.Perm()
			return os.MkdirAll(destination, 0755)
		}
		if !mode.IsRegular() {
			return fmt.Errorf("附件不支持链接或特殊文件: %s", name)
		}
		if err = os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
			return err
		}
		file, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return err
		}
		n, copyErr := io.Copy(file, io.LimitReader(reader, size+1))
		if copyErr == nil && n != size {
			copyErr = fmt.Errorf("附件文件大小不匹配: %s", name)
		}
		return errors.Join(copyErr, file.Chmod(mode.Perm()), file.Close())
	}
	if field.kind == "helm" {
		file, err := os.Open(filename)
		if err != nil {
			return err
		}
		defer file.Close()
		gz, err := gzip.NewReader(file)
		if err != nil {
			return fmt.Errorf("Helm 附件必须为 tar.gz: %w", err)
		}
		defer gz.Close()
		limited := &io.LimitedReader{R: gz, N: maxExpandedSize + (128 << 20)}
		tr := tar.NewReader(limited)
		for {
			header, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			if header.Name == "./" && header.Typeflag == tar.TypeDir {
				continue
			}
			if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA && header.Typeflag != tar.TypeDir {
				return fmt.Errorf("Helm 附件不支持链接或特殊文件: %s", header.Name)
			}
			if err = write(header.Name, header.FileInfo().Mode(), header.Size, tr); err != nil {
				return err
			}
		}
		if _, err = io.Copy(io.Discard, limited); err != nil {
			return err
		}
		if limited.N == 0 {
			return fmt.Errorf("Helm 解压内容超过限制")
		}
	} else {
		reader, err := zip.OpenReader(filename)
		if err != nil {
			return fmt.Errorf("%s 附件必须为 ZIP: %w", field.kind, err)
		}
		defer reader.Close()
		for _, item := range reader.File {
			if item.UncompressedSize64 > uint64(maxAttachmentSize) {
				return fmt.Errorf("ZIP 文件大小超过限制")
			}
			source, err := item.Open()
			if err != nil {
				return err
			}
			err = write(item.Name, item.Mode(), int64(item.UncompressedSize64), source)
			if err = errors.Join(err, source.Close()); err != nil {
				return err
			}
		}
	}
	if len(seen) == 0 {
		return fmt.Errorf("附件不能为空")
	}
	// Children are already written, so read-only directory modes are safe now.
	for name, mode := range directoryModes {
		if err := os.Chmod(name, mode); err != nil {
			return err
		}
	}
	return nil
}
