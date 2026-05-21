package logic

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"os/exec"
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
	"github.com/w7panel/w7panel-zpk/common/service/oci"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"github.com/we7coreteam/w7-rangine-go/v2/src/core/err_handler"
	"oras.land/oras-go/v2"
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

const (
	MediaTypeIcon       = "application/vnd.w7.formula.icon+png"
	MediaTypeManifest   = "application/vnd.w7.formula.manifest+yml"
	MediaTypeFilesJson  = "application/vnd.w7.formula.files.json+json"
	MediaTypeCodeZip    = "application/vnd.w7.formula.code.zip+zip"
	MediaTypeWebCodeZip = "application/vnd.w7.formula.code.web.zip+zip"
	MediaTypeHelmZip    = "application/vnd.w7.formula.helm.zip+zip"
)

var ZipFileNotFoundErr = errors.New("zip file not found")
var OciManifestNotFoundErr = errors.New("oci manifest not found")

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

func (self *Depot) GetHelmLocalRelativePath(helm Helm) string {
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
			Application `json:"application"`
		}
		manifest := &DefaultManifest{
			Application: Application{
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
		ID:                         row.ID,
		UserId:                     row.UserID,
		Name:                       row.Name,
		Title:                      row.Title,
		VersionId:                  row.VersionLatestID,
		LatestVersionId:            row.VersionLatestID,
		Manifest:                   &Manifest{},
		ZipPath:                    "",
		WebZipPaths:                map[string]string{},
		HelmPaths:                  map[string]string{},
		Icon:                       "/zpk/zip/icon/" + row.Name,
		Tags:                       row.Tag,
		ConsoleUid:                 int64(row.RemoteUID),
		GoodsId:                    row.GoodsID,
		GoodsProductId:             row.GoodsProductID,
		InstallServiceFee:          row.InstallServiceFee,
		ServicePackages:            row.ServicePackages,
		VersionPrices:              row.VersionPrices,
		IsFreeUpgrade:              row.IsFreeUpgrade,
		ProductType:                row.ProductType,
		PublishOfficialStoreStatus: row.PublishOfficialStoreStatus,
		RemoteFormulaInfoUrl:       row.RemoteFormulaInfoURL,
		AuditStatus:                row.AuditStatus,
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
			return nil, err
		}
	}

	//load all manifest
	list, _ := self.GetFileList(result)
	for key, value := range list {
		if strings.Contains(key, "manifest.yaml") {
			manifestRow := &Manifest{}
			err := yaml.Unmarshal([]byte(value), manifestRow)
			if err != nil {
				return nil, err
			}

			//处理转换旧数据格式
			manifestRow.Application.Version = result.Version
			tmpManifest := ProcessManifestIdentify(*manifestRow)
			tmpManifest = GetManifestV2(tmpManifest)
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
			return nil, err
		}
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

func (self *Depot) GetFormulaBackendZipDownloadUrlByApplication(application Application, isTemporary bool) (string, string) {
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

func (self *Depot) GetZipFileList(formula *Formula) ([]string, error) {
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

func (self *Depot) GetZipFileContent(formula *Formula, path string) ([]byte, error) {
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

func (self *Depot) Copy(src *Formula, dest *Formula) error {
	srcRemoteOci, err := logic.GetDefaultRemoteOci(self.getFormulaOciName(src))
	if err != nil {
		return err
	}
	destRemoteOci, err := logic.GetDefaultRemoteOci(self.getFormulaOciName(dest))
	if err != nil {
		return err
	}

	_, _, err = self.getOciManifest(src)
	if err != nil {
		if errors.Is(err, OciManifestNotFoundErr) {
			return nil
		}
	}

	_, err = oras.Copy(context.Background(), srcRemoteOci, self.GetFormulaOciTag(src), destRemoteOci, self.GetFormulaOciTag(dest), oras.DefaultCopyOptions)
	return err
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
	remoteOci, err := logic.GetDefaultRemoteOci(self.getFormulaOciName(formula))
	if err != nil {
		return err
	}

	resourcesDescriptor := make([]FileOciDescriptor, 0)

	iconPath := filepath.Join(self.basePath, formula.GetIconRelativePath())
	slog.Info("打包 icon", "name", formula.Name, "path", iconPath)
	if function.FileExists(iconPath) {
		ociIconDescriptor, err := oci.GetOciDescriptorByPath(iconPath, MediaTypeIcon)
		if err != nil {
			return err
		}
		resourcesDescriptor = append(resourcesDescriptor, FileOciDescriptor{
			Path:       iconPath,
			Descriptor: *ociIconDescriptor,
		})
	}

	fileList, err := self.GetFileList(formula)
	if err != nil {
		return err
	}
	slog.Info("打包 filelist", "name", formula.Name, "filelist", fileList, "err", err)
	if len(fileList) > 0 {
		fileListContent, err := json.Marshal(fileList)
		if err != nil {
			return err
		}
		ociFileListDescriptor, err := oci.GetOciDescriptorByData(fileListContent, MediaTypeFilesJson)
		if err != nil {
			return err
		}
		resourcesDescriptor = append(resourcesDescriptor, FileOciDescriptor{
			Content:    fileListContent,
			Descriptor: *ociFileListDescriptor,
		})
	}

	if strings.HasSuffix(formula.Manifest.Source.Url, ".git") {
		slog.Info("打包 git 文件", "name", formula.Name)
		gitFileDescriptors, err := self.packSourceCodeByGitToOci(formula)
		if err != nil {
			return err
		}
		resourcesDescriptor = append(resourcesDescriptor, gitFileDescriptors...)
	}
	if strings.HasSuffix(formula.Manifest.Source.Url, ".zip") || strings.HasSuffix(formula.Manifest.Web.Url, ".zip") {
		slog.Info("打包 zip 文件", "name", formula.Name)
		zipFileDescriptors, err := self.packSourceCodeByZipToOci(formula)
		if err != nil {
			return err
		}
		resourcesDescriptor = append(resourcesDescriptor, zipFileDescriptors...)
	}
	helmFileDescriptors, err := self.packHelmToOci(formula)
	if err != nil {
		return err
	}
	resourcesDescriptor = append(resourcesDescriptor, helmFileDescriptors...)

	filesDescriptor := make([]v1.Descriptor, 0)
	ctx := context.Background()
	for _, item := range resourcesDescriptor {
		if item.Content != nil {
			err = remoteOci.Push(ctx, item.Descriptor, bytes.NewReader(item.Content))
			if err != nil {
				return err
			}
		} else {
			file, err := os.Open(item.Path)
			if err != nil {
				return err
			}
			err = remoteOci.Push(ctx, item.Descriptor, file)
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
	manifestDescriptor, err := oras.PackManifest(ctx, remoteOci, oras.PackManifestVersion1_1, artifactType, oras.PackManifestOptions{
		Layers: filesDescriptor,
	})
	if err != nil {
		return err
	}
	err = remoteOci.Tag(ctx, manifestDescriptor, self.GetFormulaOciTag(formula))
	if err != nil {
		return err
	}

	slog.Info("打包成功", "name", formula.Name, "tag", self.GetFormulaOciTag(formula))

	//暂时屏蔽
	//os.RemoveAll(filepath.Join(self.basePath, GetFormulaRelativeDir(formula.Name, formula.VersionId)))
	//if formula.ZipPath != "" {
	//	os.RemoveAll(filepath.Join(self.basePath, formula.ZipPath))
	//}
	//if formula.WebZipPath != nil {
	//	for _, path := range formula.WebZipPath {
	//		os.RemoveAll(filepath.Join(self.basePath, path))
	//	}
	//}

	return nil
}

func (self *Depot) packSourceCodeByGitToOci(formula *Formula) ([]FileOciDescriptor, error) {
	gitDir := filepath.Join(self.basePath, formula.GetZipRelativeDir())
	function.CreateDirIfNotExist(gitDir, os.ModePerm)
	defer os.RemoveAll(gitDir)

	gitLogic := git{
		depot: self,
	}
	_, err := gitLogic.runCmd("clone", formula.Manifest.Source.Url, gitDir)
	if err != nil {
		return nil, err
	}

	fileList, err := self.GetFileList(formula)
	if err != nil {
		return nil, err
	}
	if len(fileList) > 0 {
		for key, value := range fileList {
			file, _ := os.Create(filepath.Join(gitDir, key))
			_, err = file.Write([]byte(value))
			if err != nil {
				return nil, err
			}
			file.Close()
		}
	}

	cmd := exec.Command("zip", "-r", fmt.Sprintf("../%s.zip", formula.Name), ".")
	cmd.Dir = gitDir
	_, err = cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	err = cmd.Run()
	if err != nil {
		return nil, err
	}

	ociDescriptor, err := oci.GetOciDescriptorByPath(filepath.Join(self.basePath, "Storage", formula.Name+".zip"), MediaTypeCodeZip)
	if err != nil {
		return nil, err
	}
	return []FileOciDescriptor{
		{
			Path:       formula.ZipPath,
			Descriptor: *ociDescriptor,
		},
	}, nil
}

func (self *Depot) packSourceCodeByZipToOci(formula *Formula) ([]FileOciDescriptor, error) {
	fileDescriptors := make([]FileOciDescriptor, 0)

	zipPath := ""
	if formula.ZipPath != "" {
		zipPath = filepath.Join(self.basePath, formula.ZipPath)
	}

	if zipPath != "" && !function.IsEmptyFile(zipPath) {
		fileList, err := self.GetFileList(formula)
		if err != nil {
			return nil, err
		}

		if fileList != nil && len(fileList) > 0 {
			zipDir := filepath.Join(self.basePath, formula.GetZipRelativeDir())
			for name, value := range fileList {
				if value != "" {
					tempPath := filepath.Join(zipDir, name)
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

					slog.Info("执行命令", "cmd", "zip -u", zipPath, name, "dir", zipDir)
					cmd := exec.Command("zip", "-u", zipPath, name)
					cmd.Dir = zipDir
					message, err := cmd.CombinedOutput()
					if err != nil {
						return nil, err
					}
					slog.Info("执行命令结果", "cmd", "zip -u", zipPath, name, "message", string(message))
				}
			}
			os.RemoveAll(zipDir)
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
	// 放前端包
	for _, webZipPath := range formula.WebZipPaths {
		webAbsolutePath := filepath.Join(self.basePath, webZipPath)
		if webZipPath != "" && !function.IsEmptyFile(webAbsolutePath) {
			slog.Info("打包 web zip", "path", webZipPath)
			ociWebDescriptor, err := oci.GetOciDescriptorByPath(webAbsolutePath, MediaTypeWebCodeZip+webZipPath)
			if err != nil {
				return nil, err
			}

			fileDescriptors = append(fileDescriptors, FileOciDescriptor{
				Path:       webAbsolutePath,
				Descriptor: *ociWebDescriptor,
			})
		}
	}

	return fileDescriptors, nil
}

func (self *Depot) packHelmToOci(formula *Formula) ([]FileOciDescriptor, error) {
	fileDescriptors := make([]FileOciDescriptor, 0)
	for _, helmPath := range formula.HelmPaths {
		helmAbsolutePath := filepath.Join(self.basePath, helmPath)
		if helmPath != "" && !function.IsEmptyFile(helmAbsolutePath) {
			slog.Info("打包 helm", "path", helmPath)
			ociHelmDescriptor, err := oci.GetOciDescriptorByPath(helmAbsolutePath, MediaTypeHelmZip+helmPath)
			if err != nil {
				return nil, err
			}

			fileDescriptors = append(fileDescriptors, FileOciDescriptor{
				Path:       helmAbsolutePath,
				Descriptor: *ociHelmDescriptor,
			})
		}
	}

	return fileDescriptors, nil
}

func (self *Depot) unPackFilesFromOci(formula *Formula) error {
	filesDir := filepath.Join(self.basePath, formula.GetFilesRelativeDir())
	if !function.FileExists(filesDir) || function.IsDirEmpty(filesDir) {
		function.CreateDirIfNotExist(filesDir, os.ModePerm)

		remoteOci, manifest, err := self.getOciManifest(formula)
		if err != nil {
			if errors.Is(err, OciManifestNotFoundErr) {
				return nil
			}
			return err
		}
		for _, layer := range manifest.Layers {
			if layer.MediaType == MediaTypeFilesJson {
				reader, err := remoteOci.Fetch(context.Background(), layer)
				if err != nil {
					return err
				}

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

				break
			}
		}
	}

	return nil
}

func (self *Depot) unPackSourceCodeFromOCI(formula *Formula) error {
	unpackIcon := false
	unpackZipCode := false
	unpackWebCode := false
	unpackHelm := false
	iconPath := filepath.Join(self.basePath, formula.GetIconRelativePath())
	if !function.FileExists(iconPath) {
		function.CreateDirIfNotExist(filepath.Dir(iconPath), os.ModePerm)
		unpackIcon = true
	}
	zipPath := filepath.Join(self.basePath, formula.ZipPath)
	if formula.ZipPath != "" {
		if !function.FileExists(zipPath) {
			function.CreateDirIfNotExist(filepath.Dir(zipPath), os.ModePerm)
			unpackZipCode = true
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
	}
	if formula.HelmPaths != nil {
		for _, path := range formula.HelmPaths {
			tmpHelmPath := filepath.Join(self.basePath, path)
			if !function.FileExists(tmpHelmPath) {
				function.CreateDirIfNotExist(filepath.Dir(tmpHelmPath), os.ModePerm)
				unpackHelm = true
			}
		}
	}
	slog.Info("unpackfromoci", "formula", formula.Name, "helms", formula.HelmPaths, "needPack", unpackHelm)
	if !unpackIcon && !unpackZipCode && !unpackWebCode && !unpackHelm {
		return nil
	}

	remoteOci, manifest, err := self.getOciManifest(formula)
	if err != nil {
		if errors.Is(err, OciManifestNotFoundErr) {
			return nil
		}
		return err
	}
	for _, layer := range manifest.Layers {
		if layer.MediaType == MediaTypeIcon && unpackIcon {
			reader, err := remoteOci.Fetch(context.Background(), layer)
			if err != nil {
				return err
			}
			file, err := os.Create(iconPath)
			if err != nil {
				return err
			}
			defer file.Close()

			if _, err := io.Copy(file, reader); err != nil {
				return err
			}
		}

		if layer.MediaType == MediaTypeCodeZip && unpackZipCode {
			reader, err := remoteOci.Fetch(context.Background(), layer)
			if err != nil {
				return err
			}

			data, err := io.ReadAll(reader)
			if err != nil {
				return err
			}

			err = os.WriteFile(zipPath, data, os.ModePerm)
			if err != nil {
				return err
			}
		}

		if strings.Contains(layer.MediaType, MediaTypeWebCodeZip) && unpackWebCode {
			reader, err := remoteOci.Fetch(context.Background(), layer)
			if err != nil {
				return err
			}

			data, err := io.ReadAll(reader)
			if err != nil {
				return err
			}

			err = os.WriteFile(filepath.Join(self.basePath, layer.MediaType[len(MediaTypeWebCodeZip)+1:]), data, os.ModePerm)
			if err != nil {
				return err
			}
		}

		if strings.Contains(layer.MediaType, MediaTypeHelmZip) && unpackHelm {
			reader, err := remoteOci.Fetch(context.Background(), layer)
			if err != nil {
				return err
			}

			data, err := io.ReadAll(reader)
			if err != nil {
				return err
			}

			err = os.WriteFile(filepath.Join(self.basePath, layer.MediaType[len(MediaTypeHelmZip)+1:]), data, os.ModePerm)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (self *Depot) getOciManifest(formula *Formula) (oras.GraphTarget, *v1.Manifest, error) {
	ociTag := self.GetFormulaOciTag(formula)
	slog.Info("开始解包项目:", formula.Name, "tag", ociTag)
	remoteOci, err := logic.GetDefaultRemoteOci(self.getFormulaOciName(formula))
	if err != nil {
		return nil, nil, err
	}

	_, fetchedManifestContent, err := oras.FetchBytes(context.Background(), remoteOci, ociTag, oras.DefaultFetchBytesOptions)
	if err != nil {
		if strings.Contains(err.Error(), ociTag+": not found") {
			return remoteOci, nil, OciManifestNotFoundErr
		}
		return nil, nil, err
	}

	// 6. Parse the fetched manifest content and get the layers
	var manifest v1.Manifest
	if err := json.Unmarshal(fetchedManifestContent, &manifest); err != nil {

		return nil, nil, err
	}

	return remoteOci, &manifest, nil
}

func (self *Depot) getFormulaOciName(formula *Formula) string {
	return facade.GetConfig().GetString("setting.depot.oci_namespace") + "/" + strings.ToLower(strings.ReplaceAll(formula.Name, "-", "_"))
}

func (self *Depot) GetFormulaOciTag(formula *Formula) string {
	versionModel, err := dao.Q.Version.Where(dao.Q.Version.ID.Eq(formula.VersionId)).First()
	if err != nil {
		return "latest"
	}

	return versionModel.Name
}

type FileOciDescriptor struct {
	Path       string
	Content    []byte
	Descriptor v1.Descriptor
}
