package logic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	containerdarchive "github.com/containerd/containerd/archive"
	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	ociStore "oras.land/oras-go/v2/content/oci"
	"oras.land/oras-go/v2/errdef"
)

const (
	artifactType        = "application/vnd.w7.tradition-plugin.v1"
	currentTag          = "current"
	layerKindAnnotation = "io.w7.tradition-plugin.kind"
)

// store 打开插件数据目录中的 OCI Layout，并由业务流程显式控制索引保存和垃圾回收。
func (service *Installer) store() (*ociStore.Store, error) {
	store, err := ociStore.New(service.dataDir)
	if err != nil {
		return nil, err
	}
	store.AutoSaveIndex = false
	store.AutoGC = false
	return store, nil
}

// loadState 从 current tag 加载当前插件安装状态。
func (service *Installer) loadState() (State, error) {
	state, _, err := service.loadTag(currentTag)
	return state, err
}

// loadTag 加载指定 tag 的 manifest 和 config，并根据层注解重建 Base 与插件状态。
func (service *Installer) loadTag(tag string) (State, v1.Manifest, error) {
	state := State{
		PluginFiles: map[string][]string{},
		Plugins:     map[string]Layer{},
	}
	var manifest v1.Manifest
	store, err := service.store()
	if err != nil {
		return state, manifest, err
	}
	manifestDescriptor, err := store.Resolve(context.Background(), tag)
	if errors.Is(err, errdef.ErrNotFound) {
		return state, manifest, nil
	}
	if err != nil {
		return state, manifest, err
	}
	manifestContent, err := content.FetchAll(context.Background(), store, manifestDescriptor)
	if err != nil {
		return state, manifest, err
	}
	if err := json.Unmarshal(manifestContent, &manifest); err != nil {
		return state, manifest, err
	}
	configContent, err := content.FetchAll(context.Background(), store, manifest.Config)
	if err != nil {
		return state, manifest, err
	}
	if err := json.Unmarshal(configContent, &state); err != nil {
		return state, manifest, err
	}
	if state.PluginFiles == nil {
		state.PluginFiles = map[string][]string{}
	}
	for _, descriptor := range manifest.Layers {
		if descriptor.MediaType == v1.MediaTypeEmptyJSON {
			continue
		}
		name := descriptor.Annotations[v1.AnnotationTitle]
		switch descriptor.Annotations[layerKindAnnotation] {
		case "base":
			state.Base = &Layer{Name: "base", Blob: descriptor}
		case "plugin":
			files, exists := state.PluginFiles[name]
			if name == "" || !exists {
				return state, manifest, fmt.Errorf("plugin layer %q has no file list", name)
			}
			state.Plugins[name] = Layer{Name: name, Blob: descriptor, Files: files}
		default:
			return state, manifest, fmt.Errorf("layer %q has no valid kind", name)
		}
	}
	return state, manifest, nil
}

// packLayer 将源目录打包为可复现的 tar 层，写入 OCI Layout 并返回文件列表。
func (service *Installer) packLayer(sourceDir, layerName string) (Layer, error) {
	sourceDir, err := sourceRoot(sourceDir)
	if err != nil {
		return Layer{}, err
	}
	files, err := treeFiles(sourceDir)
	if err != nil {
		return Layer{}, err
	}
	temp, err := os.CreateTemp(filepath.Join(service.dataDir, "tmp"), "layer-*.tar")
	if err != nil {
		return Layer{}, err
	}
	defer os.Remove(temp.Name())
	empty, err := os.MkdirTemp(filepath.Join(service.dataDir, "tmp"), "empty-*")
	if err != nil {
		temp.Close()
		return Layer{}, err
	}
	defer os.RemoveAll(empty)
	epoch := time.Unix(0, 0).UTC()
	if err := containerdarchive.WriteDiff(
		context.Background(), temp, empty, sourceDir,
		containerdarchive.WithSourceDateEpoch(&epoch),
	); err != nil {
		temp.Close()
		return Layer{}, err
	}
	if err := temp.Close(); err != nil {
		return Layer{}, err
	}
	descriptor, err := descriptorFromPath(temp.Name(), v1.MediaTypeImageLayer)
	if err != nil {
		return Layer{}, err
	}
	store, err := service.store()
	if err != nil {
		return Layer{}, err
	}
	exists, err := store.Exists(context.Background(), *descriptor)
	if err != nil {
		return Layer{}, err
	}
	if !exists {
		file, err := os.Open(temp.Name())
		if err != nil {
			return Layer{}, err
		}
		pushErr := store.Push(context.Background(), *descriptor, file)
		closeErr := file.Close()
		if pushErr != nil {
			return Layer{}, pushErr
		}
		if closeErr != nil {
			return Layer{}, closeErr
		}
	}
	return Layer{
		Name:  layerName,
		Blob:  *descriptor,
		Files: files,
	}, nil
}

// descriptorFromPath 根据文件内容计算摘要和大小，生成 OCI descriptor。
func descriptorFromPath(path, mediaType string) (*v1.Descriptor, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	fileDigest, err := digest.FromReader(file)
	if err != nil {
		return nil, err
	}
	return &v1.Descriptor{
		MediaType: mediaType,
		Digest:    fileDigest,
		Size:      info.Size(),
	}, nil
}

// applyLayer 校验 OCI Blob 摘要，并将单个 tar 层应用到目标目录。
func (service *Installer) applyLayer(targetDir string, layer v1.Descriptor) error {
	store, err := service.store()
	if err != nil {
		return err
	}
	blob, err := store.Fetch(context.Background(), layer)
	if err != nil {
		return err
	}
	defer blob.Close()
	verified := content.NewVerifyReader(blob, layer)
	_, err = containerdarchive.Apply(
		context.Background(), targetDir, verified,
		containerdarchive.WithNoSameOwner(),
	)
	if err != nil {
		return err
	}
	if _, err := io.Copy(io.Discard, verified); err != nil {
		return err
	}
	return verified.Verify()
}

// applyLayers 按 manifest 中的先后顺序将 OCI 文件层应用到站点目录。
func (service *Installer) applyLayers(
	siteDir string,
	layers []v1.Descriptor,
) error {
	for _, layer := range layers {
		if layer.MediaType == v1.MediaTypeEmptyJSON {
			continue
		}
		if layer.MediaType != v1.MediaTypeImageLayer {
			return fmt.Errorf("unsupported OCI layer media type %q", layer.MediaType)
		}
		if err := service.applyLayer(siteDir, layer); err != nil {
			return err
		}
	}
	return nil
}

// saveState 保存状态 config 和有序层 manifest，更新 current tag 并清理旧 manifest。
func (service *Installer) saveState(state State, ordered []Layer) error {
	store, err := service.store()
	if err != nil {
		return err
	}
	previous, resolveErr := store.Resolve(context.Background(), currentTag)
	if resolveErr != nil && !errors.Is(resolveErr, errdef.ErrNotFound) {
		return resolveErr
	}
	layers := make([]v1.Descriptor, 0, len(ordered))
	for _, layer := range ordered {
		descriptor := layer.Blob
		kind := "plugin"
		if layer.Name == "base" {
			kind = "base"
		}
		descriptor.Annotations = map[string]string{
			v1.AnnotationTitle:  layer.Name,
			layerKindAnnotation: kind,
		}
		layers = append(layers, descriptor)
	}
	configContent, err := json.Marshal(state)
	if err != nil {
		return err
	}
	configDescriptor := content.NewDescriptorFromBytes(
		"application/vnd.w7.tradition-plugin.config.v1+json", configContent,
	)
	exists, err := store.Exists(context.Background(), configDescriptor)
	if err != nil {
		return err
	}
	if !exists {
		if err := store.Push(context.Background(), configDescriptor, bytes.NewReader(configContent)); err != nil {
			return err
		}
	}
	manifest, err := oras.PackManifest(
		context.Background(), store, oras.PackManifestVersion1_1,
		artifactType,
		oras.PackManifestOptions{Layers: layers, ConfigDescriptor: &configDescriptor},
	)
	if err != nil {
		return err
	}
	if err := store.Tag(context.Background(), manifest, currentTag); err != nil {
		return err
	}
	if resolveErr == nil && previous.Digest != manifest.Digest {
		store.AutoGC = true
		store.AutoSaveIndex = true
		if err := store.Delete(context.Background(), previous); err != nil {
			return err
		}
		return nil
	}
	if err := store.SaveIndex(); err != nil {
		return err
	}
	return nil
}

// orderedLayers 将 Base 放在最底层，再按优先级从低到高排列插件层。
func orderedLayers(state State, policy Policy) ([]Layer, error) {
	plugins := make([]Layer, 0, len(state.Plugins))
	for name, plugin := range state.Plugins {
		if _, exists := policy.Plugins[name]; !exists {
			return nil, fmt.Errorf("managed plugin %q has no priority", name)
		}
		plugins = append(plugins, plugin)
	}
	sort.Slice(plugins, func(i, j int) bool {
		left := policy.Plugins[plugins[i].Name]
		right := policy.Plugins[plugins[j].Name]
		if left != right {
			return left < right
		}
		return plugins[i].Name < plugins[j].Name
	})
	result := make([]Layer, 0, len(plugins)+1)
	if state.Base != nil {
		result = append(result, *state.Base)
	}
	return append(result, plugins...), nil
}

// layerDescriptors 从业务层列表中提取供 manifest 应用的 OCI 描述符。
func layerDescriptors(layers []Layer) []v1.Descriptor {
	result := make([]v1.Descriptor, 0, len(layers))
	for _, layer := range layers {
		result = append(result, layer.Blob)
	}
	return result
}
