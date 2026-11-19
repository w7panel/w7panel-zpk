package respo

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/w7panel/w7panel-zpk/common/function"
	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
	jsonpatch "gopkg.in/evanphx/json-patch.v4"
	"sigs.k8s.io/yaml"
)

type UpdateOptions struct {
	Directory, App, Kind        string
	Dir, File, Target           string
	Patch, PatchFile, PatchType string
	Create                      bool
}

// Update stages only the affected files or attachment directory before saving.
func Update(options UpdateOptions) ([]string, error) {
	root, err := filepath.Abs(options.Directory)
	if err != nil {
		return nil, err
	}
	identifier, err := ReadAppIdentifier(root)
	if err != nil {
		return nil, err
	}
	manifestPath := "manifest.yaml"
	if options.App != "" && options.App != identifier {
		manifestPath = "apps/" + options.App + "/manifest.yaml"
	}
	manifestFile, err := function.ResolveLocalPath(root, manifestPath)
	if err != nil {
		return nil, err
	}
	content, err := function.ReadFile(manifestFile, 4<<20)
	if err != nil {
		return nil, err
	}
	raw, manifest, err := commonlogic.ReadManifest(content)
	if err != nil {
		return nil, err
	}
	if options.Kind == "backend" && manifestPath != "manifest.yaml" {
		return nil, fmt.Errorf("原有 OCI 格式仅支持主应用本地后端包")
	}
	workDir, err := os.MkdirTemp(filepath.Dir(root), ".zpk-update-")
	if err != nil {
		return nil, err
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.RemoveAll(workDir)
		}
	}()
	stage := filepath.Join(workDir, "next")
	if err = os.Mkdir(stage, 0755); err != nil {
		return nil, err
	}
	changes := []string{manifestPath}
	if options.Kind == "manifest" {
		content, err = updateManifest(content, options)
		if err != nil {
			return nil, err
		}
		_, updated, err := commonlogic.ReadManifest(content)
		if err != nil {
			return nil, err
		}
		if updated.Application.Identifie != manifest.Application.Identifie {
			return nil, fmt.Errorf("update 不支持修改应用标识")
		}
	} else {
		for _, field := range attachmentFields {
			if field.kind != options.Kind {
				continue
			}
			paths, err := updateAttachment(root, stage, manifestPath, manifest.Application.Identifie, raw, field, options)
			if err != nil {
				return nil, err
			}
			changes = paths
		}
		if options.Dir != "" {
			changes = append(changes, manifestPath)
			content, err = yaml.Marshal(raw)
			if err != nil {
				return nil, err
			}
		}
	}
	if options.Kind == "manifest" || options.Dir != "" {
		mode, err := os.Stat(filepath.Join(root, filepath.FromSlash(manifestPath)))
		if err != nil {
			return nil, err
		}
		if err = function.WriteFile(filepath.Join(stage, filepath.FromSlash(manifestPath)), content, mode.Mode().Perm()); err != nil {
			return nil, err
		}
	}
	if err = commitUpdate(root, stage, filepath.Join(workDir, "backup"), changes); err != nil {
		// Keep backups even if rollback itself failed (disk/permission errors).
		cleanup = false
		return nil, fmt.Errorf("%w；恢复文件保留在 %s", err, workDir)
	}
	return changes, nil
}

func updateManifest(content []byte, options UpdateOptions) ([]byte, error) {
	if options.File != "" {
		return function.ReadFile(options.File, 4<<20)
	}
	patch := []byte(options.Patch)
	if options.PatchFile != "" {
		var err error
		patch, err = function.ReadFile(options.PatchFile, 4<<20)
		if err != nil {
			return nil, err
		}
	}
	document, err := yaml.YAMLToJSONStrict(content)
	if err != nil {
		return nil, err
	}
	patch, err = yaml.YAMLToJSONStrict(patch)
	if err != nil {
		return nil, err
	}
	if options.PatchType == "json" {
		operations, err := jsonpatch.DecodePatch(patch)
		if err != nil {
			return nil, err
		}
		document, err = operations.Apply(document)
		if err != nil {
			return nil, err
		}
	} else {
		var object map[string]json.RawMessage
		if err = json.Unmarshal(patch, &object); err != nil || object == nil {
			return nil, fmt.Errorf("merge patch 必须为 YAML/JSON 对象")
		}
		document, err = jsonpatch.MergePatch(document, patch)
		if err != nil {
			return nil, err
		}
	}
	return yaml.JSONToYAML(document)
}

func updateAttachment(root, stage, manifestPath, identifier string, raw map[string]interface{}, field attachmentField, options UpdateOptions) ([]string, error) {
	relative, err := localAttachmentPath(raw, manifestPath, identifier, field)
	if err != nil {
		return nil, err
	}
	if relative == "" && options.Target != "" {
		return nil, fmt.Errorf("附件没有本地内容，请先通过 --dir 导入完整目录")
	}
	if relative == "" {
		relative = appAssetsDir(manifestPath, identifier) + "/" + field.kind
	}
	name := path.Join(path.Dir(manifestPath), relative)
	destination := filepath.Join(stage, filepath.FromSlash(name))
	currentPath, err := function.ResolveLocalPath(root, name)
	if err != nil {
		return nil, err
	}
	info, statErr := os.Stat(currentPath)
	if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return nil, statErr
	}
	if statErr == nil && !info.IsDir() {
		return nil, fmt.Errorf("本地附件必须是解压目录: %s", relative)
	}
	changes := []string{name}
	if err = os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return nil, err
	}
	if options.Dir != "" {
		if err = copyAttachmentDirectory(options.Dir, destination); err != nil {
			return nil, err
		}
	} else {
		if statErr != nil {
			return nil, fmt.Errorf("本地附件不存在: %s", relative)
		}
		target, err := function.RelativePath(options.Target)
		if err != nil {
			return nil, err
		}
		targetPath, err := function.ResolveLocalPath(root, path.Join(name, target))
		if err != nil {
			return nil, err
		}
		mode := fs.FileMode(0644)
		if info, err := os.Lstat(targetPath); err == nil {
			if !info.Mode().IsRegular() {
				return nil, fmt.Errorf("目标必须为普通文件: %s", target)
			}
			mode = info.Mode().Perm()
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		} else if !options.Create {
			return nil, fmt.Errorf("目标文件不存在，新增请使用 --create: %s", target)
		}
		if err = function.CopyFile(options.File, filepath.Join(destination, filepath.FromSlash(target)), mode, maxAttachmentSize); err != nil {
			return nil, err
		}
		return []string{path.Join(name, target)}, nil
	}
	commonlogic.SetManifestString(raw, field.section, field.field, "./"+relative)
	current := commonlogic.ManifestObject(raw, field.section)
	if field.kind == "helm" {
		delete(current, "repository")
	} else {
		current["type"] = "zip"
	}
	return changes, nil
}

func copyAttachmentDirectory(source, destination string) error {
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("--dir 必须为已解压的普通目录: %s", source)
	}
	if err = os.Mkdir(destination, 0755); err != nil {
		return err
	}
	type directoryMode struct {
		name string
		mode fs.FileMode
	}
	directories := []directoryMode{}
	err = walkAttachment(source, func(name string, info fs.FileInfo) error {
		filename := filepath.Join(destination, filepath.FromSlash(name))
		if info.IsDir() {
			directories = append(directories, directoryMode{filename, info.Mode().Perm()})
			return os.Mkdir(filename, 0755)
		}
		return function.CopyFile(filepath.Join(source, filepath.FromSlash(name)), filename, info.Mode().Perm(), maxAttachmentSize)
	})
	if err != nil {
		return err
	}
	for i := len(directories) - 1; i >= 0; i-- {
		if err = os.Chmod(directories[i].name, directories[i].mode); err != nil {
			return err
		}
	}
	return nil
}

func commitUpdate(root, stage, backup string, changes []string) error {
	if err := os.Mkdir(backup, 0700); err != nil {
		return err
	}
	type movedPath struct {
		destination, backup string
		original, updated   bool
	}
	moves := []movedPath{}
	rollback := func(cause error) error {
		for i := len(moves) - 1; i >= 0; i-- {
			move := moves[i]
			if move.updated {
				cause = errors.Join(cause, os.RemoveAll(move.destination))
			}
			if move.original {
				cause = errors.Join(cause, os.Rename(move.backup, move.destination))
			}
		}
		return cause
	}
	for index, name := range changes {
		destination := filepath.Join(root, filepath.FromSlash(name))
		source := filepath.Join(stage, filepath.FromSlash(name))
		move := movedPath{destination: destination, backup: filepath.Join(backup, fmt.Sprint(index))}
		if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
			return rollback(err)
		}
		if _, err := os.Lstat(destination); err == nil {
			if err = os.Rename(destination, move.backup); err != nil {
				return rollback(err)
			}
			move.original = true
		} else if !errors.Is(err, os.ErrNotExist) {
			return rollback(err)
		}
		moves = append(moves, move)
		if err := os.Rename(source, destination); err != nil {
			return rollback(err)
		}
		moves[len(moves)-1].updated = true
	}
	return nil
}
