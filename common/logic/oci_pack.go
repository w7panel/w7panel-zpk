package logic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"slices"
	"strings"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/w7panel/w7panel-zpk/common/function"
	"github.com/w7panel/w7panel-zpk/common/service/oci"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry/remote"
)

var OciManifestNotFoundErr = errors.New("oci manifest not found")

const (
	MediaTypeIcon           = "application/vnd.w7.formula.icon+png"
	MediaTypeFilesJson      = "application/vnd.w7.formula.files.json+json"
	MediaTypeBackendCodeZip = "application/vnd.w7.formula.code.zip+zip"
	MediaTypeFrontedCodeZip = "application/vnd.w7.formula.code.web.zip+zip"
	MediaTypeHelmZip        = "application/vnd.w7.formula.helm.zip+zip"
	MediaTypeAuth           = "application/vnd.w7.formula.auth"
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

func PackFileListToOci(fileList map[string]string) ([]FileOciDescriptor, error) {
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

	return fileDescriptors, nil
}

func PackHelmToOci(helmPaths map[string]string) ([]FileOciDescriptor, error) {
	fileDescriptors := make([]FileOciDescriptor, 0)
	for key, helmPath := range helmPaths {
		if helmPath != "" && !function.IsEmptyFile(helmPath) {
			slog.Info("打包 helm", "path", helmPath)
			ociHelmDescriptor, err := oci.GetOciDescriptorByPath(helmPath, MediaTypeHelmZip+key)
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

func PackBackendCodeZipToOci(zipPath string) ([]FileOciDescriptor, error) {
	fileDescriptors := make([]FileOciDescriptor, 0)

	if zipPath != "" && !function.IsEmptyFile(zipPath) {
		ociCodeDescriptor, err := oci.GetOciDescriptorByPath(zipPath, MediaTypeBackendCodeZip)
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

func PackFrontedCodeZipToOci(zipPaths map[string]string) ([]FileOciDescriptor, error) {
	fileDescriptors := make([]FileOciDescriptor, 0)
	// 放前端包
	for key, webZipPath := range zipPaths {
		if webZipPath != "" && !function.IsEmptyFile(webZipPath) {
			slog.Info("打包 web zip", "path", webZipPath)
			ociWebDescriptor, err := oci.GetOciDescriptorByPath(webZipPath, MediaTypeFrontedCodeZip+key)
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

func PushOciToRemote(ctx context.Context, remoteRepository *remote.Repository, tag string, resourcesDescriptor []FileOciDescriptor, baseLayers []v1.Descriptor, annotations map[string]string) error {
	filesDescriptor := make([]v1.Descriptor, 0)
	if baseLayers != nil && len(baseLayers) > 0 {
		filesDescriptor = append(filesDescriptor, baseLayers...)
	}
	for _, item := range resourcesDescriptor {
		filesDescriptor = append(filesDescriptor, item.Descriptor)
		err := remoteRepository.Mount(ctx, item.Descriptor, remoteRepository.Reference.Repository, func() (io.ReadCloser, error) {
			if item.Content != nil {
				return io.NopCloser(bytes.NewReader(item.Content)), nil
			}
			return os.Open(item.Path)
		})
		if err != nil {
			return err
		}
	}

	artifactType := "application/vnd.w7.files.v1+tar"
	v1.DescriptorEmptyJSON.Platform = &v1.Platform{
		Architecture: "amd64",
		OS:           "linux",
	}
	manifestDescriptor, err := oras.PackManifest(ctx, remoteRepository, oras.PackManifestVersion1_1, artifactType, oras.PackManifestOptions{
		Layers:              filesDescriptor,
		ManifestAnnotations: annotations,
	})
	if err != nil {
		return err
	}
	return remoteRepository.Tag(ctx, manifestDescriptor, tag)
}

func GetOciManifest(ctx context.Context, remoteRepository *remote.Repository, tag string) (*v1.Manifest, error) {
	_, fetchedManifestContent, err := oras.FetchBytes(ctx, remoteRepository, tag, oras.DefaultFetchBytesOptions)
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

func UnPackOciToLocal(ctx context.Context, remoteRepository *remote.Repository, manifest *v1.Manifest, mediaTypes []string, writeHandler func(mediaType string, savePath string, reader io.Reader) error) error {
	for _, layer := range manifest.Layers {
		if slices.Contains(mediaTypes, layer.MediaType) {
			reader, err := remoteRepository.Fetch(ctx, layer)
			if err != nil {
				return err
			}
			err = writeHandler(layer.MediaType, "", reader)
			closeErr := reader.Close()
			if err != nil {
				return err
			}
			if closeErr != nil {
				return closeErr
			}
		} else {
			for _, item := range mediaTypes {
				if strings.HasPrefix(layer.MediaType, item+"/") {
					reader, err := remoteRepository.Fetch(ctx, layer)
					if err != nil {
						return err
					}
					err = writeHandler(item, layer.MediaType[len(item)+1:], reader)
					closeErr := reader.Close()
					if err != nil {
						return err
					}
					if closeErr != nil {
						return closeErr
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
