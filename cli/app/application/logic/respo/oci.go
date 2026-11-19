package respo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	clicommon "github.com/w7panel/w7panel-zpk/cli/app/application/logic/common"
	"github.com/w7panel/w7panel-zpk/common/function"
	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/w7panel/w7panel-zpk/common/service/oci"
	"oras.land/oras-go/v2/content"
	"sigs.k8s.io/yaml"
)

// TransferResult describes OCI transport only. Helm building and market
// publication remain the server's existing, separate workflow.
type TransferResult struct {
	Reference string
	Version   string
	Digest    string
}

type appPackage struct {
	Identifie string
	Resources []commonlogic.FileOciDescriptor
}

type appManifest struct {
	name     string
	raw      map[string]interface{}
	manifest commonlogic.Manifest
}

func Push(ctx context.Context, target clicommon.Target, session clicommon.Session, directory string, output io.Writer) (*TransferResult, error) {
	workDir, err := os.MkdirTemp("", "zpk-pack-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(workDir)
	pkg, err := pack(directory, workDir)
	if err != nil {
		return nil, err
	}
	if pkg.Identifie != target.Identifie {
		return nil, fmt.Errorf("本地应用标识 %s 与目标 %s 不一致", pkg.Identifie, target.Identifie)
	}
	repository, err := clicommon.NewRepository(target, session)
	if err != nil {
		return nil, err
	}
	tags, err := clicommon.ListVersions(ctx, repository)
	if err != nil {
		return nil, err
	}
	version := target.Version
	if version == "" || version == "latest" {
		version, err = clicommon.NextVersion(tags)
		if err != nil {
			return nil, err
		}
	}
	result := &TransferResult{Reference: target.Reference(), Version: version}
	if err = commonlogic.PushOciToRemote(ctx, repository, version, pkg.Resources, nil); err != nil {
		return nil, err
	}
	descriptor, err := repository.Resolve(ctx, version)
	if err != nil {
		return nil, fmt.Errorf("版本 %s 已推送，但查询推送结果失败: %w", version, err)
	}
	latestDescriptor := descriptor
	versions := append(tags, version)
	function.SortTagsDesc(versions)
	if versions[0] != version {
		latestDescriptor, err = repository.Resolve(ctx, versions[0])
		if err != nil {
			return nil, fmt.Errorf("版本 %s 已推送，但解析最新版失败: %w", version, err)
		}
	}
	if err = repository.Tag(ctx, latestDescriptor, "latest"); err != nil {
		return nil, fmt.Errorf("版本 %s 已推送，但 latest 更新失败: %w", version, err)
	}
	result.Digest = descriptor.Digest.String()
	if output != nil {
		fmt.Fprintf(output, "OCI 已推送 %s:%s\n%s\n", result.Reference, result.Version, result.Digest)
	}
	return result, nil
}

func Pull(ctx context.Context, target clicommon.Target, session clicommon.Session, directory string) (*TransferResult, error) {
	destination, err := filepath.Abs(directory)
	if err != nil {
		return nil, err
	}
	if _, err = os.Lstat(destination); err == nil {
		return nil, fmt.Errorf("目标目录已存在，请使用新目录: %s", destination)
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	repository, err := clicommon.NewRepository(target, session)
	if err != nil {
		return nil, err
	}
	version, descriptor, err := clicommon.ResolveVersion(ctx, repository, target.Version)
	if err != nil {
		return nil, fmt.Errorf("拉取应用失败（非公开应用请先 login）: %w", err)
	}
	manifest, err := commonlogic.GetOciManifest(ctx, repository, descriptor.Digest.String())
	if err != nil {
		return nil, err
	}
	if manifest.ArtifactType != "application/vnd.w7.files.v1+tar" {
		return nil, fmt.Errorf("目标不是 ZPK 应用包")
	}
	if err = os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return nil, err
	}
	stage, err := os.MkdirTemp(filepath.Dir(destination), ".zpk-pull-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(stage)
	workDir, err := os.MkdirTemp("", "zpk-pull-blobs-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(workDir)
	blobs := map[string]string{}
	download := v1.Manifest{}
	mediaTypes := []string{}
	seen := map[string]bool{}
	for _, layer := range manifest.Layers {
		if seen[layer.Digest.String()] {
			continue
		}
		seen[layer.Digest.String()] = true
		download.Layers = append(download.Layers, layer)
		mediaTypes = append(mediaTypes, layer.MediaType)
	}
	index := 0
	// Exact media types select every download layer, preserving callback order.
	err = commonlogic.UnPackOciToLocal(ctx, repository, &download, mediaTypes, func(_, _ string, reader io.Reader) error {
		layer := download.Layers[index]
		filename := filepath.Join(workDir, fmt.Sprint(index))
		index++
		maxSize := maxAttachmentSize
		if layer.MediaType == commonlogic.MediaTypeFilesJson {
			maxSize = 8 << 20
		}
		if layer.Size < 0 || layer.Size > maxSize {
			return fmt.Errorf("blob 大小无效")
		}
		file, err := os.Create(filename)
		if err != nil {
			return err
		}
		verified := content.NewVerifyReader(reader, layer)
		_, copyErr := io.Copy(file, verified)
		if err = errors.Join(copyErr, file.Close()); err != nil {
			return err
		}
		if err = verified.Verify(); err != nil {
			return err
		}
		blobs[layer.Digest.String()] = filename
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err = unpack(*manifest, blobs, target.Identifie, stage); err != nil {
		return nil, err
	}

	// Recheck to avoid replacing a directory created while the transfer ran.
	if _, err = os.Lstat(destination); !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("目标目录已存在或不可访问: %s", destination)
	}
	if err = os.Rename(stage, destination); err != nil {
		return nil, err
	}
	return &TransferResult{Reference: target.Reference(), Version: version, Digest: descriptor.Digest.String()}, nil
}

func ReadAppIdentifier(directory string) (string, error) {
	filename, err := function.ResolveLocalPath(directory, "manifest.yaml")
	if err != nil {
		return "", err
	}
	content, err := function.ReadFile(filename, 4<<20)
	if err != nil {
		return "", err
	}
	_, manifest, err := commonlogic.ReadManifest(content)
	return manifest.Application.Identifie, err
}

// pack compiles the editable directory into existing application OCI layers.
// Only push calls this function; pull and update operate on unpacked files.
func pack(directory, workDir string) (*appPackage, error) {
	root, err := filepath.Abs(directory)
	if err != nil {
		return nil, err
	}
	manifests, err := readAppManifests(root)
	if err != nil {
		return nil, err
	}
	pkg := &appPackage{Identifie: manifests[0].manifest.Application.Identifie}
	runtimeFiles := map[string]string{}
	manifestPaths := map[string]bool{}
	attachmentDirs := map[string]bool{}
	for _, entry := range manifests {
		resources, directories, packErr := packAttachments(root, workDir, entry)
		if packErr != nil {
			return nil, packErr
		}
		pkg.Resources = append(pkg.Resources, resources...)
		for _, name := range directories {
			attachmentDirs[name] = true
		}
		manifestPaths[entry.name] = true
		content, marshalErr := yaml.Marshal(entry.raw)
		if marshalErr != nil {
			return nil, marshalErr
		}
		name := "manifest.yaml"
		if entry.name != name {
			name = entry.manifest.Application.Identifie + "/manifest.yaml"
		}
		runtimeFiles[name] = string(content)
	}
	packed, err := commonlogic.PackFileListToOci(runtimeFiles)
	if err != nil {
		return nil, err
	}
	pkg.Resources = append(pkg.Resources, packed...)
	err = filepath.WalkDir(root, func(filename string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(root, filename)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(relative)
		if entry.IsDir() {
			if attachmentDirs[name] {
				return filepath.SkipDir
			}
			return nil
		}
		if manifestPaths[name] {
			return nil
		}
		if !entry.Type().IsRegular() || isAppManifest(name) {
			return fmt.Errorf("不支持的应用包文件: %s", name)
		}
		if _, err = function.RelativePath(name); err != nil {
			return err
		}
		media := "application/octet-stream"
		if name == "icon.png" {
			media = commonlogic.MediaTypeIcon
		}
		descriptor, err := oci.GetOciDescriptorByPath(filename, media)
		if err != nil {
			return err
		}
		descriptor.Annotations = map[string]string{v1.AnnotationTitle: name}
		pkg.Resources = append(pkg.Resources, commonlogic.FileOciDescriptor{Path: filename, Descriptor: *descriptor})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return pkg, nil
}

func readAppManifests(root string) ([]appManifest, error) {
	read := func(name string) (appManifest, error) {
		filename, err := function.ResolveLocalPath(root, name)
		if err != nil {
			return appManifest{}, err
		}
		content, err := function.ReadFile(filename, 4<<20)
		if err != nil {
			return appManifest{}, err
		}
		raw, manifest, err := commonlogic.ReadManifest(content)
		return appManifest{name, raw, manifest}, err
	}
	main, err := read("manifest.yaml")
	if err != nil {
		return nil, err
	}
	entries := []appManifest{main}
	seen := map[string]bool{main.manifest.Application.Identifie: true}
	for _, dependency := range main.manifest.Platform.Depends {
		if dependency.Type == "out" {
			continue
		}
		if seen[dependency.Identifie] {
			return nil, fmt.Errorf("重复应用标识: %s", dependency.Identifie)
		}
		entry, err := read("apps/" + dependency.Identifie + "/manifest.yaml")
		if err != nil {
			return nil, err
		}
		if entry.manifest.Application.Identifie != dependency.Identifie {
			return nil, fmt.Errorf("子应用目录与标识不一致: %s", dependency.Identifie)
		}
		seen[dependency.Identifie] = true
		entries = append(entries, entry)
	}
	return entries, nil
}

func packAttachments(root, workDir string, entry appManifest) ([]commonlogic.FileOciDescriptor, []string, error) {
	resources := []commonlogic.FileOciDescriptor{}
	directories := []string{}
	for _, field := range attachmentFields {
		relative, err := localAttachmentPath(entry.raw, entry.name, entry.manifest.Application.Identifie, field)
		if err != nil {
			return nil, nil, err
		}
		if relative == "" {
			continue
		}
		name := path.Join(path.Dir(entry.name), relative)
		directory, err := function.ResolveLocalPath(root, name)
		if err != nil {
			return nil, nil, err
		}
		info, err := os.Stat(directory)
		if err != nil {
			return nil, nil, err
		}
		if !info.IsDir() {
			return nil, nil, fmt.Errorf("本地附件必须是解压目录: %s", name)
		}
		filename := filepath.Join(workDir, entry.manifest.Application.Identifie+"-"+field.kind+field.extension)
		if err = packAttachment(directory, filename, field); err != nil {
			return nil, nil, err
		}
		descriptor, err := oci.GetOciDescriptorByPath(filename, field.media)
		if err != nil {
			return nil, nil, err
		}
		storage := "/Storage/sha256/" + descriptor.Digest.Encoded() + field.extension
		if field.kind != "backend" {
			descriptor.MediaType += storage
		}
		descriptor.Annotations = map[string]string{v1.AnnotationTitle: name}
		resources = append(resources, commonlogic.FileOciDescriptor{Path: filename, Descriptor: *descriptor})
		commonlogic.SetManifestString(entry.raw, field.section, field.field, "file://"+storage)
		directories = append(directories, name)
	}
	return resources, directories, nil
}

func appAssetsDir(manifestPath, identifier string) string {
	if manifestPath == "manifest.yaml" {
		return "apps/" + identifier + "/assets"
	}
	return "assets"
}

func localAttachmentPath(raw map[string]interface{}, manifestPath, identifier string, field attachmentField) (string, error) {
	value := commonlogic.ManifestString(raw, field.section, field.field)
	if value == "" || strings.HasPrefix(value, "https://") || strings.HasPrefix(value, "http://") {
		return "", nil
	}
	if field.kind == "helm" && commonlogic.ManifestString(raw, field.section, "repository") != "" && !strings.HasPrefix(value, "file://") {
		return "", nil
	}
	if field.kind == "backend" && manifestPath != "manifest.yaml" {
		return "", fmt.Errorf("%s: 原有 OCI 格式仅支持主应用本地后端 ZIP，子应用请使用镜像或远程源码引用", manifestPath)
	}
	relative := strings.TrimPrefix(strings.TrimPrefix(value, "file://"), "./")
	if _, err := function.RelativePath(relative); err != nil {
		return "", fmt.Errorf("%s 中的附件引用 %q: %w", manifestPath, value, err)
	}
	assetsDir := appAssetsDir(manifestPath, identifier)
	if !strings.HasPrefix(relative, assetsDir+"/") {
		return "", fmt.Errorf("%s 的本地附件必须位于 %s/: %s", manifestPath, assetsDir, value)
	}
	return relative, nil
}

func isAppManifest(name string) bool {
	return name == "manifest.yaml" || (strings.HasPrefix(name, "apps/") && strings.Count(name, "/") == 2 && path.Base(name) == "manifest.yaml")
}

// unpack restores downloaded OCI blobs into the editable directory layout.
// Transport and temporary blob files belong to Pull.
func unpack(manifest v1.Manifest, blobs map[string]string, identifier, directory string) error {
	files := map[string]string{}
	attachments := map[string][]commonlogic.FileOciDescriptor{}
	for _, layer := range manifest.Layers {
		filename := blobs[layer.Digest.String()]
		if layer.MediaType == commonlogic.MediaTypeFilesJson {
			data, err := function.ReadFile(filename, 8<<20)
			if err != nil {
				return err
			}
			if err = json.Unmarshal(data, &files); err != nil {
				return err
			}
			continue
		}
		isAttachment := false
		for _, field := range attachmentFields {
			if layer.MediaType == field.media || strings.HasPrefix(layer.MediaType, field.media+"/") {
				attachments[layer.MediaType] = append(attachments[layer.MediaType], commonlogic.FileOciDescriptor{Path: filename, Descriptor: layer})
				isAttachment = true
				break
			}
		}
		if isAttachment {
			continue
		}
		name := layer.Annotations[v1.AnnotationTitle]
		if name == "" && layer.MediaType == commonlogic.MediaTypeIcon {
			name = "icon.png"
		}
		if name == "" {
			continue
		}
		if isAppManifest(name) {
			return fmt.Errorf("应用包资源路径无效: %s", name)
		}
		destination, err := function.ResolveLocalPath(directory, name)
		if err != nil {
			return err
		}
		if err = function.CopyFile(filename, destination, 0644, maxAttachmentSize); err != nil {
			return err
		}
	}
	if files["manifest.yaml"] == "" {
		return fmt.Errorf("应用包缺少 manifest.yaml")
	}
	budget := &extractionBudget{}
	for filename, source := range files {
		raw, app, err := commonlogic.ReadManifest([]byte(source))
		if err != nil {
			return err
		}
		localName := "manifest.yaml"
		if filename == localName {
			if app.Application.Identifie != identifier {
				return fmt.Errorf("根应用标识不一致")
			}
		} else {
			if filename != app.Application.Identifie+"/manifest.yaml" {
				return fmt.Errorf("子应用 manifest 路径无效: %s", filename)
			}
			localName = "apps/" + filename
		}
		entry := appManifest{localName, raw, app}
		if err = unpackAttachments(directory, entry, attachments, budget); err != nil {
			return err
		}
		content, err := yaml.Marshal(raw)
		if err != nil {
			return err
		}
		if err = function.WriteFile(filepath.Join(directory, filepath.FromSlash(localName)), content, 0644); err != nil {
			return err
		}
	}
	return nil
}

func unpackAttachments(directory string, entry appManifest, attachments map[string][]commonlogic.FileOciDescriptor, budget *extractionBudget) error {
	base := path.Dir(entry.name)
	assets := appAssetsDir(entry.name, entry.manifest.Application.Identifie)
	prefix := path.Join(base, assets) + "/"
	for _, field := range attachmentFields {
		reference := commonlogic.ManifestString(entry.raw, field.section, field.field)
		if !strings.HasPrefix(reference, "file://") {
			continue
		}
		media := field.media
		if field.kind != "backend" {
			media += strings.TrimPrefix(reference, "file://")
		}
		var attachment commonlogic.FileOciDescriptor
		for _, candidate := range attachments[media] {
			title := candidate.Descriptor.Annotations[v1.AnnotationTitle]
			if title == "" || strings.HasPrefix(title, prefix) {
				attachment = candidate
				break
			}
		}
		if attachment.Path == "" {
			return fmt.Errorf("应用包缺少附件: %s", reference)
		}
		relative := path.Join(assets, field.kind)
		if title := attachment.Descriptor.Annotations[v1.AnnotationTitle]; title != "" {
			relative = title
			if base != "." {
				relative = strings.TrimPrefix(title, base+"/")
			}
		}
		destination, err := function.ResolveLocalPath(directory, path.Join(base, relative))
		if err != nil {
			return err
		}
		if err = os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
			return err
		}
		if err = unpackAttachment(attachment.Path, destination, field, budget); err != nil {
			return err
		}
		commonlogic.SetManifestString(entry.raw, field.section, field.field, "./"+relative)
	}
	return nil
}
