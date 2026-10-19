package formula

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	"github.com/w7panel/w7panel-zpk/common/function"
	"github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/w7panel/w7panel-zpk/common/types/registry"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry/remote"
	"sigs.k8s.io/yaml"
)

func (self *Depot) Copy(src *Formula, dest *Formula) error {
	srcRemoteOci, err := logic.GetDefaultRemoteOci(logic.GetFormulaOciName(src.Name))
	if err != nil {
		return err
	}
	destRemoteOci, err := logic.GetDefaultRemoteOci(logic.GetFormulaOciName(dest.Name))
	if err != nil {
		return err
	}

	_, _, err = self.getOciManifest(src)
	if err != nil {
		if errors.Is(err, logic.OciManifestNotFoundErr) {
			return nil
		}
	}

	_, err = oras.Copy(context.Background(), srcRemoteOci, self.GetFormulaOciTag(src), destRemoteOci, self.GetFormulaOciTag(dest), oras.DefaultCopyOptions)
	return err
}

func (self Depot) OnRepositoryPushed(payload registry.RegistryRepositoryWebHookPayLoad) {
	slog.Info("depot OnRepositoryPushed", "payload", payload)
	repositoryName, namespace := logic.ParseRepositoryNameAndNamespace(payload.Event.Target.Repository)
	if namespace != facade.GetConfig().GetString("setting.depot.oci_namespace") {
		return
	}
	tag := payload.Event.Target.Tag
	if tag == "" || tag == AssetsOCITag {
		return
	}

	depot, _ := NewDepot()
	formula := GetFormulaByName(repositoryName)
	if formula == nil {
		username := payload.Event.Actor.Name
		if username == "" {
			slog.Error("OnRepositoryPushed UnPackOciToLocal err", "payload", payload, "err", "empty username")
			return
		}
		user, err := logic.User{}.GetByUsername(username)
		if err != nil {
			slog.Error("OnRepositoryPushed UnPackOciToLocal GetByUsername err", "payload", payload, "err", err)
			return
		}
		err = depot.AddFormula(repositoryName, tag, user)
		if err != nil {
			slog.Error("OnRepositoryPushed Failed to add formula to repository", "payload", payload, "err", err)
			return
		}
		formula = GetFormulaByName(repositoryName)
	}
	if formula == nil {
		slog.Error("OnRepositoryPushed Failed to get formula from repository", "payload", payload)
		return
	}

	version, _ := dao.Q.Version.Where(dao.Q.Version.FormulaID.Eq(formula.ID)).Where(dao.Q.Version.Name.Eq(tag)).First()
	if version == nil {
		err := dao.Q.Version.Create(&entity.Version{
			FormulaID:     formula.ID,
			Name:          tag,
			Description:   "新版本",
			PublishStatus: -1,
		})
		if err != nil {
			slog.Error("OnRepositoryPushed Failed to create version", "payload", payload, "err", err)
		}
	} else {
		filesDir := filepath.Join(depot.GetBasePath(), filepath.Join(GetFormulaRelativeDir(formula.Name, version.ID), "files"))
		slog.Info("OnRepositoryPushed formula oci after push clear cache", "payload:", payload, "filesDir", filesDir)
		os.RemoveAll(filesDir)
	}
}

func (self *Depot) packToOci(formula *Formula) error {
	slog.Info("开始打包项目:", "info", formula)
	remoteOci, err := logic.GetDefaultRemoteOci(logic.GetFormulaOciName(formula.Name))
	if err != nil {
		return err
	}

	resourcesDescriptor := make([]logic.FileOciDescriptor, 0)

	iconPath := filepath.Join(self.basePath, formula.GetIconRelativePath())
	slog.Info("打包 icon", "name", formula.Name, "path", iconPath)
	iconDescriptors, err := logic.PackIconToOci(iconPath)
	if err != nil {
		return err
	}
	resourcesDescriptor = append(resourcesDescriptor, iconDescriptors...)

	fileList, err := self.GetManifestFileList(formula)
	if err != nil {
		return err
	}
	if formula.Setting != nil && formula.Setting.BaseInfo != nil {
		if manifestContent, exists := fileList["manifest.yaml"]; exists {
			manifest := &logic.Manifest{}
			if err = yaml.Unmarshal([]byte(manifestContent), manifest); err != nil {
				return err
			}
			formula.ApplyBaseInfo(manifest)
			content, marshalErr := yaml.Marshal(manifest)
			if marshalErr != nil {
				return marshalErr
			}
			fileList["manifest.yaml"] = string(content)
		}
	}
	slog.Info("打包 filelist 文件", "name", formula.Name, "filelist", fileList, "err", err)
	fileListDescriptors, err := logic.PackFileListToOci(fileList)
	if err != nil {
		return err
	}
	resourcesDescriptor = append(resourcesDescriptor, fileListDescriptors...)

	if strings.HasSuffix(formula.Manifest.Source.Url, ".zip") {
		backendCodePath := filepath.Join(self.basePath, formula.ZipPath)
		slog.Info("打包 backend zip 文件", "name", formula.Name, "path", backendCodePath, "err", err)
		backendCodeDescriptors, err := logic.PackBackendCodeZipToOci(backendCodePath)
		if err != nil {
			return err
		}
		resourcesDescriptor = append(resourcesDescriptor, backendCodeDescriptors...)
	}

	frontedCodePaths := map[string]string{}
	for _, webZipPath := range formula.WebZipPaths {
		frontedCodePaths[webZipPath] = filepath.Join(self.basePath, webZipPath)
	}
	slog.Info("打包 frontend zip 文件", "name", formula.Name, "paths", frontedCodePaths, "err", err)
	frontedCodeDescriptors, err := logic.PackFrontedCodeZipToOci(frontedCodePaths)
	if err != nil {
		return err
	}
	resourcesDescriptor = append(resourcesDescriptor, frontedCodeDescriptors...)

	helmPaths := map[string]string{}
	for _, path := range formula.HelmPaths {
		helmPaths[path] = filepath.Join(self.basePath, path)
	}
	slog.Info("打包 helm 文件", "name", formula.Name, "paths", helmPaths, "err", err)
	helmFileDescriptors, err := logic.PackHelmToOci(helmPaths)
	if err != nil {
		return err
	}
	resourcesDescriptor = append(resourcesDescriptor, helmFileDescriptors...)

	tag := self.GetFormulaOciTag(formula)
	err = logic.PushOciToRemote(remoteOci, tag, resourcesDescriptor, nil)
	slog.Info("打包完成", "name", formula.Name, "tag", tag, "err", err)
	return err
}

func (self *Depot) unPackManifestFilesFromOci(formula *Formula) error {
	filesDir := filepath.Join(self.basePath, formula.GetFilesRelativeDir())
	if function.FileExists(filesDir) && !function.IsDirEmpty(filesDir) {
		return nil
	}
	remoteOci, manifest, err := self.getOciManifest(formula)
	if err != nil {
		if errors.Is(err, logic.OciManifestNotFoundErr) {
			return nil
		}
		return err
	}

	return logic.UnPackOciToLocal(remoteOci, manifest, []string{logic.MediaTypeFilesJson}, func(mediaType string, savePath string, reader io.Reader) error {
		files := map[string]string{}
		if err := json.NewDecoder(reader).Decode(&files); err != nil {
			return err
		}
		for name, content := range files {
			if !IsFormulaManifestPath(name) {
				continue
			}
			targetPath, err := resolveFormulaFilePath(filesDir, name)
			if err != nil {
				return err
			}
			if err = os.MkdirAll(filepath.Dir(targetPath), os.ModePerm); err != nil {
				return err
			}
			if err = os.WriteFile(targetPath, []byte(content), 0o644); err != nil {
				return err
			}
		}
		return nil
	})
}

func (self *Depot) unPackSourceCodeFromOCI(formula *Formula) error {
	unpackZipCode := false
	unpackWebCode := false
	unpackHelm := false
	mediaTypes := make([]string, 0)
	zipPath := filepath.Join(self.basePath, formula.ZipPath)
	if formula.ZipPath != "" {
		if !function.FileExists(zipPath) {
			function.CreateDirIfNotExist(filepath.Dir(zipPath), os.ModePerm)
			unpackZipCode = true
			mediaTypes = append(mediaTypes, logic.MediaTypeBackendCodeZip)
		}
	}
	if formula.WebZipPaths != nil {
		for _, path := range formula.WebZipPaths {
			tmpWebPath := filepath.Join(self.basePath, path)
			if !function.FileExists(tmpWebPath) {
				function.CreateDirIfNotExist(filepath.Dir(tmpWebPath), os.ModePerm)
				unpackWebCode = true
			}
		}
		if unpackWebCode {
			mediaTypes = append(mediaTypes, logic.MediaTypeFrontedCodeZip)
		}
	}
	if formula.HelmPaths != nil {
		for _, path := range formula.HelmPaths {
			tmpHelmPath := filepath.Join(self.basePath, path)
			if !function.FileExists(tmpHelmPath) {
				function.CreateDirIfNotExist(filepath.Dir(tmpHelmPath), os.ModePerm)
				unpackHelm = true
			}
		}
		if unpackHelm {
			mediaTypes = append(mediaTypes, logic.MediaTypeHelmZip)
		}
	}
	slog.Info("unpackfromoci", "formula", formula.Name, "helms", formula.HelmPaths, "needPack", unpackHelm, "mediaTypes", mediaTypes)
	if !unpackZipCode && !unpackWebCode && !unpackHelm {
		return nil
	}

	remoteOci, manifest, err := self.getOciManifest(formula)
	if err != nil {
		if errors.Is(err, logic.OciManifestNotFoundErr) {
			return nil
		}
		return err
	}

	return logic.UnPackOciToLocal(remoteOci, manifest, mediaTypes, func(mediaType string, savePath string, reader io.Reader) error {
		data, err := io.ReadAll(reader)
		if err != nil {
			return err
		}

		if mediaType == logic.MediaTypeBackendCodeZip {
			err = os.WriteFile(zipPath, data, os.ModePerm)
			if err != nil {
				return err
			}
		}

		if mediaType == logic.MediaTypeFrontedCodeZip || mediaType == logic.MediaTypeHelmZip {
			err = os.WriteFile(filepath.Join(self.basePath, savePath), data, os.ModePerm)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (self *Depot) unPackIconFromOCI(formula *Formula) error {
	unpackIcon := false
	mediaTypes := make([]string, 0)
	iconPath := filepath.Join(self.basePath, formula.GetIconRelativePath())
	if !function.FileExists(iconPath) {
		function.CreateDirIfNotExist(filepath.Dir(iconPath), os.ModePerm)
		unpackIcon = true
		mediaTypes = append(mediaTypes, logic.MediaTypeIcon)
	}

	if !unpackIcon {
		return nil
	}

	remoteOci, manifest, err := self.getOciManifest(formula)
	if err != nil {
		if errors.Is(err, logic.OciManifestNotFoundErr) {
			return nil
		}
		return err
	}

	return logic.UnPackOciToLocal(remoteOci, manifest, mediaTypes, func(mediaType string, savePath string, reader io.Reader) error {
		data, err := io.ReadAll(reader)
		if err != nil {
			return err
		}

		if mediaType == logic.MediaTypeIcon {
			file, err := os.Create(iconPath)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = file.Write(data)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (self *Depot) getOciManifest(formula *Formula) (*remote.Repository, *v1.Manifest, error) {
	ociTag := self.GetFormulaOciTag(formula)
	slog.Info("开始解包项目", "formula", formula.Name, "tag", ociTag)
	remoteOci, err := logic.GetDefaultRemoteOci(logic.GetFormulaOciName(formula.Name))
	if err != nil {
		return nil, nil, err
	}

	manifest, err := logic.GetOciManifest(remoteOci, ociTag)

	return remoteOci, manifest, err
}

func (self *Depot) GetFormulaOciTag(formula *Formula) string {
	versionModel, err := dao.Q.Version.Where(dao.Q.Version.ID.Eq(formula.VersionId)).First()
	if err != nil {
		return "latest"
	}

	return versionModel.Name
}
