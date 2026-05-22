package logic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/w7panel/w7panel-zpk/common/function"
	"github.com/w7panel/w7panel-zpk/common/service/oci"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry/remote"
)

var DefaultWorkDir = filepath.Join(os.TempDir(), "w7panel-zpk", "oci-pack")
var OciManifestNotFoundErr = errors.New("oci manifest not found")

const (
	MediaTypeIcon       = "application/vnd.w7.formula.icon+png"
	MediaTypeFilesJson  = "application/vnd.w7.formula.files.json+json"
	MediaTypeCodeZip    = "application/vnd.w7.formula.code.zip+zip"
	MediaTypeWebCodeZip = "application/vnd.w7.formula.code.web.zip+zip"
	MediaTypeHelmZip    = "application/vnd.w7.formula.helm.zip+zip"
)

type FileOciDescriptor struct {
	Path       string
	Content    []byte
	Descriptor v1.Descriptor
}

func PackIconToOci(iconPath string) ([]FileOciDescriptor, error) {
	fileDescriptors := make([]FileOciDescriptor, 0)
	if function.FileExists(iconPath) {
		ociIconDescriptor, err := oci.GetOciDescriptorByPath(iconPath, MediaTypeIcon)
		if err != nil {
			return nil, err
		}
		fileDescriptors = append(fileDescriptors, FileOciDescriptor{
			Path:       iconPath,
			Descriptor: *ociIconDescriptor,
		})
	}

	return fileDescriptors, nil
}

func PackHelmToOci(helmPaths []string) ([]FileOciDescriptor, error) {
	fileDescriptors := make([]FileOciDescriptor, 0)
	for _, helmPath := range helmPaths {
		if helmPath != "" && !function.IsEmptyFile(helmPath) {
			slog.Info("打包 helm", "path", helmPath)
			ociHelmDescriptor, err := oci.GetOciDescriptorByPath(helmPath, MediaTypeHelmZip+helmPath)
			if err != nil {
				return nil, err
			}

			fileDescriptors = append(fileDescriptors, FileOciDescriptor{
				Path:       helmPath,
				Descriptor: *ociHelmDescriptor,
			})
		}
	}

	return fileDescriptors, nil
}

func PackBackendCodeZipToOci(zipPath string, fileList map[string]string) ([]FileOciDescriptor, error) {
	fileDescriptors := make([]FileOciDescriptor, 0)

	if len(fileList) > 0 {
		fileListContent, err := json.Marshal(fileList)
		if err != nil {
			return nil, err
		}
		ociFileListDescriptor, err := oci.GetOciDescriptorByData(fileListContent, MediaTypeFilesJson)
		if err != nil {
			return nil, err
		}
		fileDescriptors = append(fileDescriptors, FileOciDescriptor{
			Content:    fileListContent,
			Descriptor: *ociFileListDescriptor,
		})
	}

	if zipPath != "" && !function.IsEmptyFile(zipPath) {
		packDir := filepath.Join(DefaultWorkDir, function.GetRandomString(8))
		if fileList != nil && len(fileList) > 0 {
			for name, value := range fileList {
				if value != "" {
					tempPath := filepath.Join(packDir, name)
					function.CreateDirIfNotExist(filepath.Dir(tempPath), os.ModePerm)

					file, err := os.Create(tempPath)
					if err != nil {
						return nil, err
					}
					_, err = file.WriteString(value)
					if err != nil {
						return nil, err
					}
					file.Close()

					slog.Info("执行命令", "cmd", "zip -u", zipPath, name, "dir", packDir)
					cmd := exec.Command("zip", "-u", zipPath, name)
					cmd.Dir = packDir
					message, err := cmd.CombinedOutput()
					if err != nil {
						return nil, err
					}
					slog.Info("执行命令结果", "cmd", "zip -u", zipPath, name, "message", string(message))
				}
			}
			os.RemoveAll(packDir)
		}

		ociCodeDescriptor, err := oci.GetOciDescriptorByPath(zipPath, MediaTypeCodeZip)
		if err != nil {
			return nil, err
		}
		fileDescriptors = append(fileDescriptors, FileOciDescriptor{
			Path:       zipPath,
			Descriptor: *ociCodeDescriptor,
		})
	}

	return fileDescriptors, nil
}

func PackFrontedCodeZipToOci(zipPaths []string) ([]FileOciDescriptor, error) {
	fileDescriptors := make([]FileOciDescriptor, 0)
	// 放前端包
	for _, webZipPath := range zipPaths {
		if webZipPath != "" && !function.IsEmptyFile(webZipPath) {
			slog.Info("打包 web zip", "path", webZipPath)
			ociWebDescriptor, err := oci.GetOciDescriptorByPath(webZipPath, MediaTypeWebCodeZip+webZipPath)
			if err != nil {
				return nil, err
			}

			fileDescriptors = append(fileDescriptors, FileOciDescriptor{
				Path:       webZipPath,
				Descriptor: *ociWebDescriptor,
			})
		}
	}

	return fileDescriptors, nil
}

func PushOciToRemote(remoteRepository *remote.Repository, tag string, resourcesDescriptor []FileOciDescriptor) error {
	filesDescriptor := make([]v1.Descriptor, 0)
	ctx := context.Background()
	for _, item := range resourcesDescriptor {
		if item.Content != nil {
			err := remoteRepository.Push(ctx, item.Descriptor, bytes.NewReader(item.Content))
			if err != nil {
				return err
			}
		} else {
			file, err := os.Open(item.Path)
			if err != nil {
				return err
			}
			err = remoteRepository.Push(ctx, item.Descriptor, file)
			file.Close()
			if err != nil {
				return err
			}
		}
		filesDescriptor = append(filesDescriptor, item.Descriptor)
	}

	artifactType := "application/vnd.w7.files.v1+tar"
	v1.DescriptorEmptyJSON.Platform = &v1.Platform{
		Architecture: "amd64",
		OS:           "linux",
	}
	manifestDescriptor, err := oras.PackManifest(ctx, remoteRepository, oras.PackManifestVersion1_1, artifactType, oras.PackManifestOptions{
		Layers: filesDescriptor,
	})
	if err != nil {
		return err
	}
	return remoteRepository.Tag(ctx, manifestDescriptor, tag)
}

func GetOciManifest(remoteRepository *remote.Repository, tag string) (*v1.Manifest, error) {
	_, fetchedManifestContent, err := oras.FetchBytes(context.Background(), remoteRepository, tag, oras.DefaultFetchBytesOptions)
	if err != nil {
		if strings.Contains(err.Error(), tag+": not found") {
			return nil, OciManifestNotFoundErr
		}
		return nil, err
	}

	// 6. Parse the fetched manifest content and get the layers
	var manifest v1.Manifest
	if err := json.Unmarshal(fetchedManifestContent, &manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

func UnPackOciToLocal(remoteRepository *remote.Repository, manifest *v1.Manifest, mediaTypes []string, writeHandler func(mediaType string, savePath string, reader io.Reader) error) error {
	for _, layer := range manifest.Layers {
		if slices.Contains(mediaTypes, layer.MediaType) {
			reader, err := remoteRepository.Fetch(context.Background(), layer)
			if err != nil {
				return err
			}
			err = writeHandler(layer.MediaType, "", reader)
			if err != nil {
				return err
			}
		} else {
			for _, item := range mediaTypes {
				if strings.Contains(layer.MediaType, item) {
					reader, err := remoteRepository.Fetch(context.Background(), layer)
					if err != nil {
						return err
					}
					err = writeHandler(layer.MediaType, layer.MediaType[len(item)+1:], reader)
					if err != nil {
						return err
					}
					break
				}
			}
		}
	}

	return nil
}

func GetFormulaOciName(formulaName string) string {
	return facade.GetConfig().GetString("setting.depot.oci_namespace") + "/" + strings.ToLower(strings.ReplaceAll(formulaName, "-", "_"))
}
