package logic

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	"github.com/w7panel/w7panel-zpk/common/function"
	"github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/w7panel/w7panel-zpk/common/service"
	"github.com/w7panel/w7panel-zpk/common/types/registry"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"github.com/we7coreteam/w7-rangine-go/v2/src/core/err_handler"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry/remote"
	"sigs.k8s.io/yaml"
)

const (
	SYNC_STATUS_PROCESS = 1
	SYNC_STATUS_FINISH  = 2
	SYNC_STATUS_FAILED  = 3
)

const (
	SETTING_TYPE_FILE    = 1
	SETTING_TYPE_SETTING = 2
	SETTING_TYPE_ICON    = 3
)

func RegisterDepot() error {
	err := facade.GetContainer().NamedSingleton("depot", func() *Depot {
		depot := &Depot{}
		depot.basePath = strings.TrimRight(facade.GetConfig().GetString("setting.depot.storage.local.path"), "/")
		depot.packChan = make(chan *Formula)
		depot.DownloadMapping = sync.Map{}
		depot.formulaMapping = sync.Map{}
		return depot
	})
	return err
}

func NewDepot() (*Depot, error) {
	var obj *Depot
	err := facade.GetContainer().NamedResolve(&obj, "depot")
	if err != nil {
		return nil, err
	}
	return obj, nil
}

type Depot struct {
	basePath        string
	formulaMapping  sync.Map
	DownloadMapping sync.Map
	packChan        chan *Formula
	PackProcessLock atomic.Bool
}

func (self *Depot) InitDepotEnv() error {
	mask := syscall.Umask(0)
	defer syscall.Umask(mask)

	if !function.FileExists(self.basePath) {
		err := os.MkdirAll(self.basePath, service.FileMode)
		if err != nil {
			return err_handler.Throw("资源仓库创建失败，请检查配置目录权限", err)
		}
	}

	return nil
}

func (self *Depot) GetBasePath() string {
	return self.basePath
}

func (self *Depot) GetHelmLocalRelativePath(helm logic.Helm) string {
	if helm.Repository == "" && helm.ChartName != "" {
		if strings.HasPrefix(helm.ChartName, "/Storage") || strings.HasPrefix(helm.ChartName, "file://") {
			return strings.TrimPrefix(helm.ChartName, "file://")
		}
	}

	return ""
}

func (self *Depot) AddFormula(name string, version string, user *entity.RegistryUser) error {
	if user == nil {
		return errors.New("user 不能为空")
	}

	name = strings.ReplaceAll(name, "_", "-")
	formulaData, _ := dao.Q.Formula.Where(dao.Q.Formula.Name.Eq(name)).First()
	if formulaData != nil {
		latestVersion, _ := dao.Q.Version.Where(dao.Q.Version.FormulaID.Eq(formulaData.ID)).Where(dao.Q.Version.Name.Eq(version)).First()
		if latestVersion != nil {
			function.CreateDirIfNotExist(filepath.Join(self.basePath, GetFormulaRelativeDir(name, latestVersion.ID)), os.ModePerm)
			return nil
		}
	}

	newCreate := false
	versionId := int32(0)
	err := dao.Q.Transaction(func(tx *dao.Query) error {
		versionData := &entity.Version{
			Name:          version,
			Description:   "",
			PublishStatus: -1,
		}
		err := dao.Q.Version.Create(versionData)
		if err != nil {
			return err
		}

		if formulaData == nil {
			formulaData = &entity.Formula{
				Name:            name,
				VersionLatestID: versionData.ID,
				UserID:          user.ID,
				Title:           name,
				AuditStatus:     FOEMULA_AUDIT_ING,
			}
			versionId = versionData.ID
			newCreate = true
			err = dao.Q.Formula.Create(formulaData)
			if err != nil {
				return err
			}
		}

		_, err = dao.Q.Version.Where(dao.Q.Version.ID.Eq(versionData.ID)).Update(dao.Version.FormulaID, formulaData.ID)
		if err != nil {
			return err
		}

		function.CreateDirIfNotExist(filepath.Join(self.basePath, GetFormulaRelativeDir(name, formulaData.VersionLatestID)), os.ModePerm)
		return nil
	})
	if err != nil {
		return err
	}

	if newCreate {
		type DefaultManifest struct {
			logic.Application `json:"application"`
		}
		manifest := &DefaultManifest{
			Application: logic.Application{
				Name:      name,
				Identifie: name,
			},
		}
		content, err := yaml.Marshal(manifest)
		if err == nil {
			_ = self.SaveFile(&Formula{
				Name:      name,
				VersionId: versionId,
			}, "manifest.yaml", string(content))
		}

		go facade.GetEvent().Publish(registry.AddUserPermissionEvent, registry.AddUserPermissionPayload{
			UserID:        user.ID,
			ResourceType:  "repository",
			ResourceValue: facade.GetConfig().GetString("setting.depot.oci_namespace") + "/" + strings.ReplaceAll(name, "-", "_"),
			Actions:       []string{"push", "pull"},
		})
	}

	return nil
}

func (self *Depot) GetFormula(name string, version string, user *entity.RegistryUser) (*Formula, error) {
	name = strings.ReplaceAll(name, "_", "-")

	query := dao.Q.Formula.Preload(dao.Formula.Tag).Where(dao.Formula.Name.Value(name))
	if user != nil && !(logic.User{}.IsAdminUser(user)) {
		query = query.Where(dao.Formula.UserID.Eq(user.ID))
	}
	row, _ := query.First()
	if row == nil {
		return nil, err_handler.Throw("仓库不存在请先添加", nil)
	}

	if row.ProductType == 99 {
		row.ProductType = 0
	}
	result := &Formula{
		ID:                   row.ID,
		UserId:               row.UserID,
		Name:                 row.Name,
		Title:                row.Title,
		VersionId:            row.VersionLatestID,
		LatestVersionId:      row.VersionLatestID,
		Manifest:             &logic.Manifest{},
		ZipPath:              "",
		WebZipPaths:          map[string]string{},
		HelmPaths:            map[string]string{},
		Icon:                 "/zpk/zip/icon/" + row.Name,
		Tags:                 row.Tag,
		ConsoleUid:           row.RemoteUID,
		GoodsId:              row.GoodsID,
		GoodsProductId:       row.GoodsProductID,
		InstallServiceFee:    row.InstallServiceFee,
		ServicePackages:      row.ServicePackages,
		VersionPrices:        row.VersionPrices,
		IsFreeUpgrade:        row.IsFreeUpgrade,
		ProductType:          row.ProductType,
		RemoteFormulaInfoUrl: row.RemoteFormulaInfoURL,
		AuditStatus:          row.AuditStatus,
	}

	if version != "" {
		versionInfo := result.GetVersionByName(version)
		if versionInfo != nil {
			result.VersionId = versionInfo.ID
			result.Version = version
		}
	} else {
		latestVersion, err := dao.Q.Version.Where(dao.Q.Version.ID.Eq(row.VersionLatestID)).First()
		if err == nil {
			result.Version = latestVersion.Name
		}
	}

	if result.RemoteFormulaInfoUrl == "" {
		err := self.unPackFilesFromOci(result)
		if err != nil {
			slog.Error("unPackFilesFromOci err", "formula_name", result.Name, "version", result.Version, "err", err)
			return nil, err
		}
	}

	//load all manifest
	list, _ := self.GetFileList(result)
	for key, value := range list {
		if strings.Contains(key, "manifest.yaml") {
			manifestRow := &logic.Manifest{}
			err := yaml.Unmarshal([]byte(value), manifestRow)
			if err != nil {
				return nil, err
			}

			//处理转换旧数据格式
			manifestRow.Application.Version = result.Version
			tmpManifest := logic.ProcessManifestIdentify(*manifestRow)
			tmpManifest = logic.GetManifestV2(tmpManifest)
			manifestRow = &tmpManifest
			if key == "manifest.yaml" {
				result.Manifest = manifestRow
			}

			result.AllManifest = append(result.AllManifest, manifestRow)
		}
	}

	if strings.HasPrefix(result.Manifest.Source.Url, "file://") {
		result.ZipPath = strings.Split(result.Manifest.Source.Url, "file://")[1]
	}

	for _, webManifest := range result.AllManifest {
		if strings.HasPrefix(webManifest.Web.Url, "file://") {
			result.WebZipPaths[webManifest.Application.Identifie] = strings.Split(webManifest.Web.Url, "file://")[1]
		}

		if strings.HasPrefix(webManifest.Platform.Helm.ChartName, "file://") || strings.HasPrefix(webManifest.Platform.Helm.ChartName, "/Storage") {
			result.HelmPaths[webManifest.Application.Identifie] = strings.TrimPrefix(webManifest.Platform.Helm.ChartName, "file://")
		}
	}

	if result.RemoteFormulaInfoUrl == "" {
		err := self.unPackSourceCodeFromOCI(result)
		if err != nil {
			slog.Error("unPackSourceCodeFromOCI err", "formula_name", result.Name, "version", result.Version, "err", err)
			return nil, err
		}
		go func() {
			err = self.unPackIconFromOCI(result)
			if err != nil {
				slog.Error("unPackIconFromOCI faile", "formula_name", result.Name, "version", result.Version, "err", err)
			}
		}()
	}

	return result, nil
}

func (self *Depot) GetFormulaBackendZipDownloadUrl(formula *Formula, isTemporary bool) string {
	if formula.ZipPath != "" {
		zipUrl, token := self.GetFormulaBackendZipDownloadUrlByApplication(formula.Manifest.Application, isTemporary)
		self.DownloadMapping.Store(token, formula.ZipPath)

		return zipUrl
	}

	return ""
}

func (self *Depot) GetFormulaBackendZipDownloadUrlByApplication(application logic.Application, isTemporary bool) (string, string) {
	token := ""
	if isTemporary {
		token = function.GetRandomString(20)
	} else {
		token = function.GetMd5("backend" + application.Identifie + application.Version)
	}

	domain := facade.GetConfig().GetString("setting.depot.external_domain")
	zipUrl := fmt.Sprintf("https://%s/zpk/zip/download/%s", domain, token)

	return zipUrl, token
}

func (self *Depot) GetFormulaHelmDownloadUrl(formula *Formula) string {
	helmPath, err := PackFormulaToHelmAndPack(*formula, false)
	if err != nil {
		slog.Error("pack helm err", "formula", formula, "err", err)
	}
	if helmPath != "" {
		token := function.GetRandomString(20)
		domain := facade.GetConfig().GetString("setting.depot.external_domain")
		helmPackageUrl := fmt.Sprintf("https://%s/zpk/zip/download/%s", domain, token)
		self.DownloadMapping.Store(token, strings.TrimPrefix(helmPath, self.GetBasePath()))
		return helmPackageUrl
	}
	return ""
}

func (self *Depot) DeleteFormula(formula *Formula) error {
	_, err := dao.Q.Formula.Where(dao.Q.Formula.ID.Eq(formula.ID)).Delete()
	if err != nil {
		return err
	}

	os.RemoveAll(filepath.Join(self.basePath, GetFormulaRelativeDir(formula.Name, 0)))

	facade.GetEvent().Publish(registry.DeleteUserPermissionEvent, registry.DelUserPermissionPayload{
		UserID:        formula.UserId,
		ResourceType:  "repository",
		ResourceValue: facade.GetConfig().GetString("setting.depot.oci_namespace") + "/" + strings.ReplaceAll(formula.Name, "-", "_"),
	})

	return nil
}

func (self *Depot) SaveFile(formula *Formula, filename string, content string) error {
	filePath := filepath.Join(self.basePath, formula.GetFilesRelativeDir(), filename)
	if content == "" {
		os.Remove(filePath)
	} else {
		function.CreateDirIfNotExist(filepath.Dir(filePath), os.ModePerm)
		file, _ := os.Create(filePath)
		defer file.Close()
		_, err := file.Write([]byte(content))
		if err != nil {
			return err
		}
	}

	return nil
}

func (self *Depot) GetFileList(formula *Formula) (map[string]string, error) {
	dir := filepath.Join(self.basePath, formula.GetFilesRelativeDir())
	function.CreateDirIfNotExist(dir, os.ModePerm)
	files := map[string]string{}
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			content, err := io.ReadAll(file)
			if err != nil {
				return err
			}
			files[path[len(dir)+1:]] = string(content)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}

func (self *Depot) GetBackendZipFileList(formula *Formula) ([]string, error) {
	result := make([]string, 0)

	if formula.ZipPath == "" {
		return result, nil
	}

	fileList, _ := self.GetFileList(formula)
	zipPath := filepath.Join(self.basePath, formula.ZipPath)
	zipReader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, err
	}
	for _, file := range zipReader.File {
		if strings.HasPrefix(file.Name, "__MACOSX") {
			continue
		}
		if _, exists := fileList[file.Name]; exists {
			continue
		}

		result = append(result, file.Name)
	}

	return result, nil
}

func (self *Depot) GetBackendZipFileContent(formula *Formula, path string) ([]byte, error) {
	zipPath := filepath.Join(self.basePath, formula.ZipPath)
	zipReader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, err
	}

	file, err := zipReader.Open(path)
	if err != nil {
		return nil, err
	}

	return io.ReadAll(file)
}

func (self *Depot) GetFrontendZipFileContent(formula *Formula, path string) ([]byte, error) {
	if _, exists := formula.WebZipPaths[formula.Name]; !exists {
		return nil, nil
	}

	zipPath := filepath.Join(self.basePath, formula.WebZipPaths[formula.Name])
	cacheRoot := filepath.Join(os.TempDir(), "w7panel-zpk", "zip_file_cache")
	return Attach{}.GetZipFileContent(cacheRoot, zipPath, path)
}

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
	if tag == "" {
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

func (self *Depot) Pack(formula *Formula, async bool) error {
	if async {
		self.packChan <- formula

		return nil
	}

	return self.packToOci(formula)
}

func (self *Depot) PackLoop() {
	for {
		select {
		case formula := <-self.packChan:
			self.PackProcessLock.Swap(true)

			err := self.packToOci(formula)
			if err != nil {
				slog.Error("打包项目失败", "name", formula.Name, "err", err)
			}

			self.PackProcessLock.Swap(false)
			log.Println("打包完成")
		}
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

	fileList, err := self.GetFileList(formula)
	if err != nil {
		return err
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

func (self *Depot) unPackFilesFromOci(formula *Formula) error {
	filesDir := filepath.Join(self.basePath, formula.GetFilesRelativeDir())
	if !function.FileExists(filesDir) || function.IsDirEmpty(filesDir) {
		function.CreateDirIfNotExist(filesDir, os.ModePerm)

		remoteOci, manifest, err := self.getOciManifest(formula)
		if err != nil {
			if errors.Is(err, logic.OciManifestNotFoundErr) {
				return nil
			}
			return err
		}

		return logic.UnPackOciToLocal(remoteOci, manifest, []string{logic.MediaTypeFilesJson}, func(mediaType string, savePath string, reader io.Reader) error {
			readAll, err := io.ReadAll(reader)
			if err != nil {
				return err
			}
			files := map[string]string{}
			err = json.Unmarshal(readAll, &files)
			if err != nil {
				return err
			}

			for name, content := range files {
				tempPath := filepath.Join(filesDir, name)
				function.CreateDirIfNotExist(filepath.Dir(tempPath), os.ModePerm)
				file, err := os.Create(tempPath)
				if err != nil {
					return err
				}
				_, err = file.WriteString(content)
				if err != nil {
					return err
				}
				file.Close()
			}

			return nil
		})
	}

	return nil
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
	slog.Info("开始解包项目:", formula.Name, "tag", ociTag)
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
