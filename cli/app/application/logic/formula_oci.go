package logic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/w7panel/w7panel-zpk/common/function"
	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/w7panel/w7panel-zpk/common/service/oci"
	"github.com/w7panel/w7panel-zpk/common/service/w7/base"
	"github.com/w7panel/w7panel-zpk/common/service/w7/zpk"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"sigs.k8s.io/yaml"
)

func PackFormulaToOci(session Session) error {
	password, err := DecryptPassword(session.Password)
	if err != nil {
		return err
	}

	nextVersion := "1.0.0"
	formulaInfo, err := zpk.ZpkService{
		Base: base.Base{
			BaseUrl: "https://" + session.Host,
		},
	}.GetRemoteFormulaBaseInfo(session.Artifact)
	if err == nil && formulaInfo != nil && formulaInfo.LatestVersion != "" {
		nextVersion, err = nextPatchVersion(formulaInfo.LatestVersion)
		if err != nil {
			return err
		}
	}

	remoteRepository, err := oci.NewRepositoryOci(session.Host, commonlogic.GetFormulaOciName(session.Artifact), auth.Credential{
		Username: session.Username,
		Password: password,
	})
	if err != nil {
		return err
	}

	descriptors := make([]commonlogic.FileOciDescriptor, 0)
	layerDescriptors := make([]v1.Descriptor, 0)
	fileList := map[string]string{}
	replacedMediaTypes := make([]string, 0)
	if formulaInfo != nil && formulaInfo.LatestVersion != "" {
		manifest, err := commonlogic.GetOciManifest(remoteRepository, formulaInfo.LatestVersion)
		if err != nil && !errors.Is(err, commonlogic.OciManifestNotFoundErr) {
			return err
		}
		if manifest != nil {
			layerDescriptors = append(layerDescriptors, manifest.Layers...)
			fileList, err = getFormulaFileList(remoteRepository, manifest)
			if err != nil {
				return err
			}
		}
	}

	formulaManifests, err := loadFormulaManifests(session, nextVersion, fileList)
	if err != nil {
		return err
	}

	replacedMediaTypes = append(replacedMediaTypes, commonlogic.MediaTypeFilesJson)
	for _, item := range session.Attachments {
		if _, err := os.Stat(item.Path); err != nil {
			return fmt.Errorf("attachment %s is not readable: %w", item.Path, err)
		}
		fileMd5, err := function.GetFileMD5(item.Path)
		if err != nil {
			return err
		}

		pathInfo := function.GetPathInfo(item.Path)
		attachKey := fmt.Sprintf("/Storage/%s/%s%s", time.Now().Format("200601"), fileMd5, pathInfo.Extension)

		formulaManifest, err := formulaManifests.manifest(item.Artifact)
		if err != nil {
			return err
		}

		switch item.Type {
		case AttachTypeFrontend:
			packed, err := commonlogic.PackFrontedCodeZipToOci(map[string]string{attachKey: item.Path})
			if err != nil {
				return err
			}
			descriptors = append(descriptors, packed...)

			if strings.HasPrefix(formulaManifest.Web.Url, "file://") {
				realPath := strings.TrimPrefix(formulaManifest.Web.Url, "file://")
				if realPath != "" {
					replacedMediaTypes = append(replacedMediaTypes, commonlogic.MediaTypeWebCodeZip+realPath)
				}
			}
		case AttachTypeBackend:
			packed, err := commonlogic.PackBackendCodeZipToOci(item.Path)
			if err != nil {
				return err
			}
			descriptors = append(descriptors, packed...)
			replacedMediaTypes = append(replacedMediaTypes, commonlogic.MediaTypeCodeZip)
		case AttachTypeHelm:
			packed, err := commonlogic.PackHelmToOci(map[string]string{attachKey: item.Path})
			if err != nil {
				return err
			}
			descriptors = append(descriptors, packed...)

			if strings.HasPrefix(formulaManifest.Platform.Helm.ChartName, "file://") || strings.HasPrefix(formulaManifest.Platform.Helm.ChartName, "/Storage") {
				realPath := strings.TrimPrefix(formulaManifest.Platform.Helm.ChartName, "file://")
				if realPath != "" {
					replacedMediaTypes = append(replacedMediaTypes, commonlogic.MediaTypeHelmZip+realPath)
				}
			}
		default:
			return fmt.Errorf("unsupported attachment type %q", item.Type)
		}
		item.Path = attachKey
		updateManifestByAttachment(formulaManifest, item)
	}

	manifestDescriptors, err := packManifestFileList(formulaManifests, fileList)
	if err != nil {
		return err
	}
	descriptors = append(descriptors, manifestDescriptors...)

	layerDescriptors = dropFormulaLayers(layerDescriptors, replacedMediaTypes)
	return pushFormulaManifest(remoteRepository, nextVersion, layerDescriptors, descriptors)
}

func nextPatchVersion(latestVersion string) (string, error) {
	parts := strings.Split(strings.TrimSpace(latestVersion), ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid latest version %q", latestVersion)
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return "", fmt.Errorf("invalid latest version %q: %w", latestVersion, err)
	}
	parts[2] = strconv.Itoa(patch + 1)
	return strings.Join(parts, "."), nil
}

func getFormulaFileList(remoteRepository *remote.Repository, manifest *v1.Manifest) (map[string]string, error) {
	fileList := map[string]string{}
	err := commonlogic.UnPackOciToLocal(remoteRepository, manifest, []string{commonlogic.MediaTypeFilesJson}, func(mediaType string, savePath string, reader io.Reader) error {
		readAll, err := io.ReadAll(reader)
		if err != nil {
			return err
		}
		return json.Unmarshal(readAll, &fileList)
	})

	return fileList, err
}

type formulaManifestSet struct {
	manifests map[string]*commonlogic.Manifest
	paths     map[string]string
}

func (set *formulaManifestSet) manifest(artifact string) (*commonlogic.Manifest, error) {
	artifact = strings.Trim(strings.TrimSpace(artifact), "/")
	manifest, ok := set.manifests[artifact]
	if !ok {
		return nil, fmt.Errorf("manifest for artifact %q not found", artifact)
	}
	return manifest, nil
}

func loadFormulaManifests(session Session, version string, fileList map[string]string) (*formulaManifestSet, error) {
	if fileList == nil {
		fileList = map[string]string{}
	}

	result := &formulaManifestSet{
		manifests: map[string]*commonlogic.Manifest{},
		paths:     map[string]string{},
	}

	if len(fileList) == 0 {
		manifest := &commonlogic.Manifest{}
		manifest.Application.Name = session.Artifact
		manifest.Application.Identifie = strings.ReplaceAll(session.Artifact, "_", "-")
		manifest.Application.Type = commonlogic.Docker_App
		manifest.Application.Version = version
		result.manifests[session.Artifact] = manifest
		result.paths[session.Artifact] = "manifest.yaml"
		return result, nil
	}

	for path, manifestContent := range fileList {
		artifact, ok := artifactFromManifestPath(path)
		if !ok || strings.TrimSpace(manifestContent) == "" {
			continue
		}
		var manifest commonlogic.Manifest
		if err := yaml.Unmarshal([]byte(manifestContent), &manifest); err != nil {
			return nil, fmt.Errorf("unmarshal %s: %w", path, err)
		}

		if artifact == "" {
			artifact = session.Artifact
		}
		artifact = strings.ReplaceAll(artifact, "_", "-")

		manifest.Application.Version = version
		result.manifests[artifact] = &manifest
		result.paths[artifact] = path
	}

	return result, nil
}

func artifactFromManifestPath(path string) (string, bool) {
	path = strings.Trim(strings.TrimSpace(path), "/")
	if path == "manifest.yaml" {
		return "", true
	}
	if !strings.HasSuffix(path, "/manifest.yaml") {
		return "", false
	}
	return strings.TrimSuffix(path, "/manifest.yaml"), true
}

func packManifestFileList(set *formulaManifestSet, fileList map[string]string) ([]commonlogic.FileOciDescriptor, error) {
	for artifact, manifest := range set.manifests {
		rawManifestContent, err := yaml.Marshal(manifest)
		if err != nil {
			return nil, err
		}
		path := set.paths[artifact]
		fileList[path] = string(rawManifestContent)
	}

	descriptors, err := commonlogic.PackFileListToOci(fileList)
	if err != nil {
		return nil, err
	}

	return descriptors, nil
}

func updateManifestByAttachment(manifest *commonlogic.Manifest, attachment Attachment) {
	switch attachment.Type {
	case AttachTypeFrontend:
		manifest.Web.Url = "file://" + attachment.Path
		manifest.Web.Type = "zip"
	case AttachTypeBackend:
		manifest.Source.Url = "file://" + attachment.Path
		manifest.Source.Type = "zip"
	case AttachTypeHelm:
		manifest.Application.Type = commonlogic.Help_App
		manifest.Platform.Helm.ChartName = "file://" + attachment.Path
		manifest.Platform.Helm.Repository = ""
		manifest.Platform.Helm.Version = ""
	}
}

func dropFormulaLayers(layers []v1.Descriptor, mediaTypes []string) []v1.Descriptor {
	if len(layers) == 0 || len(mediaTypes) == 0 {
		return layers
	}

	result := make([]v1.Descriptor, 0, len(layers))
	for _, layer := range layers {
		keep := true
		for _, mediaType := range mediaTypes {
			if layer.MediaType == mediaType || strings.HasPrefix(layer.MediaType, mediaType) {
				keep = false
				break
			}
		}
		if keep {
			result = append(result, layer)
		}
	}

	return result
}

func pushFormulaManifest(remoteRepository *remote.Repository, tag string, baseLayers []v1.Descriptor, resources []commonlogic.FileOciDescriptor) error {
	ctx := context.Background()
	layers := append([]v1.Descriptor{}, baseLayers...)
	for _, item := range resources {
		if item.Content != nil {
			if err := remoteRepository.Push(ctx, item.Descriptor, bytes.NewReader(item.Content)); err != nil {
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
		layers = append(layers, item.Descriptor)
	}

	v1.DescriptorEmptyJSON.Platform = &v1.Platform{
		Architecture: "amd64",
		OS:           "linux",
	}
	manifestDescriptor, err := oras.PackManifest(ctx, remoteRepository, oras.PackManifestVersion1_1, "application/vnd.w7.files.v1+tar", oras.PackManifestOptions{
		Layers: layers,
	})
	if err != nil {
		return err
	}

	return remoteRepository.Tag(ctx, manifestDescriptor, tag)
}
