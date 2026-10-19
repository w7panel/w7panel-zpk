package formula

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"

	attachlogic "github.com/w7panel/w7panel-zpk/app/respo/logic/attach"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	"github.com/w7panel/w7panel-zpk/common/function"
	"github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/w7panel/w7panel-zpk/common/service"
	"github.com/w7panel/w7panel-zpk/common/types/registry"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"github.com/we7coreteam/w7-rangine-go/v2/src/core/err_handler"
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

var (
	ErrFrontendPackageNotFound = errors.New("前端包不存在")
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
		err := tx.Version.Create(versionData)
		if err != nil {
			return err
		}

		if formulaData == nil {
			formulaData = &entity.Formula{
				Name:            name,
				VersionLatestID: versionData.ID,
				UserID:          user.ID,
				Title:           name,
			}
			versionId = versionData.ID
			newCreate = true
			err = tx.Formula.Create(formulaData)
			if err != nil {
				return err
			}
		}

		_, err = tx.Version.Where(tx.Version.ID.Eq(versionData.ID)).Update(tx.Version.FormulaID, formulaData.ID)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	function.CreateDirIfNotExist(filepath.Join(self.basePath, GetFormulaRelativeDir(name, versionId)), os.ModePerm)

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
			_ = self.SaveManifestFile(&Formula{
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
		CrossUpgradeFormulas: row.CrossUpgradeFormulas,
		Setting:              row.Setting,
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

	err := self.unPackManifestFilesFromOci(result)
	if err != nil {
		slog.Error("unPackManifestFilesFromOci err", "formula_name", result.Name, "version", result.Version, "err", err)
		return nil, err
	}
	sharedFilesDir := self.getSharedFilesDir(result.Name)
	if !function.FileExists(sharedFilesDir) || function.IsDirEmpty(sharedFilesDir) {
		if err = self.unPackSharedFilesFromOci(result); err != nil {
			slog.Error("unPackSharedFilesFromOci err", "formula_name", result.Name, "version", result.Version, "err", err)
			return nil, err
		}
	}
	//load all manifest
	list, _ := self.GetManifestFileList(result)
	rootManifest, err := self.parseFormulaManifestFile(list["manifest.yaml"])
	if err != nil {
		return nil, err
	}
	rootManifest.Application.Version = result.Version
	result.ApplyBaseInfo(rootManifest)
	result.Manifest = rootManifest
	result.AllManifest = append(result.AllManifest, rootManifest)

	for key, value := range list {
		if key == "manifest.yaml" {
			continue
		}
		manifestRow, err := self.parseFormulaManifestFile(value)
		if err != nil {
			return nil, err
		}
		self.applyChildManifestVersion(rootManifest, manifestRow, result.Version)
		result.AllManifest = append(result.AllManifest, manifestRow)
	}
	sortManifestsByApplicationOrder(result.AllManifest)

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

	err = self.unPackSourceCodeFromOCI(result)
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

	return result, nil
}

// FormulaInstallOnlyOnce resolves the field required by dependency selectors.
// Formula-level base info is stored in the database for newer records; legacy
// records fall back to their root manifest without restoring source archives,
// Helm charts, shared files, or icons from OCI.
func (self *Depot) FormulaInstallOnlyOnce(row *entity.Formula) (bool, error) {
	if row == nil {
		return false, errors.New("formula is nil")
	}
	formula := &Formula{
		ID:              row.ID,
		Name:            row.Name,
		Title:           row.Title,
		VersionId:       row.VersionLatestID,
		LatestVersionId: row.VersionLatestID,
		Setting:         row.Setting,
	}
	if row.Setting != nil && row.Setting.BaseInfo != nil {
		return formula.GetBaseInfo().InstallOnlyOnce, nil
	}
	if err := self.unPackManifestFilesFromOci(formula); err != nil {
		return false, err
	}
	manifestPath := filepath.Join(self.basePath, formula.GetFilesRelativeDir(), "manifest.yaml")
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		return false, err
	}
	manifest, err := self.parseFormulaManifestFile(string(content))
	if err != nil {
		return false, err
	}
	formula.Manifest = manifest
	return formula.GetBaseInfo().InstallOnlyOnce, nil
}

func (self *Depot) parseFormulaManifestFile(content string) (*logic.Manifest, error) {
	manifest := &logic.Manifest{}
	if err := yaml.Unmarshal([]byte(content), manifest); err != nil {
		return nil, err
	}
	result := logic.ProcessManifestIdentify(*manifest)
	result = logic.GetManifestV2(result)
	return &result, nil
}

func (self *Depot) applyChildManifestVersion(root, child *logic.Manifest, formulaVersion string) {
	if root == nil || child == nil {
		return
	}
	for _, dependency := range root.Platform.Depends {
		if dependency.Identifie == child.Application.Identifie && strings.TrimSpace(dependency.From) != "" {
			return
		}
	}
	child.Application.Version = formulaVersion
}

func (self *Depot) GetFormulaBackendZipDownloadUrl(formula *Formula, isTemporary bool) string {
	if formula.ZipPath != "" {
		zipUrl, token := self.GetFormulaBackendZipDownloadUrlByApplication(formula.Manifest.Application, formula.ZipPath, isTemporary)
		if isTemporary {
			self.DownloadMapping.Store(token, formula.ZipPath)
		}

		return zipUrl
	}

	return ""
}

func (self *Depot) GetFormulaBackendZipDownloadUrlByApplication(application logic.Application, zipPath string, isTemporary bool) (string, string) {
	token := ""
	if isTemporary {
		token = function.GetRandomString(20)
	} else {
		var err error
		token, err = attachlogic.CreatePermanentAttachmentDownloadToken(application, zipPath)
		if err != nil {
			slog.Error("create permanent attachment download token failed", "identifie", application.Identifie, "version", application.Version, "err", err)
			return "", ""
		}
	}

	domain := facade.GetConfig().GetString("setting.depot.external_domain")
	zipUrl := fmt.Sprintf("https://%s/zpk/zip/download/%s", domain, token)

	return zipUrl, token
}

func (self *Depot) DeleteFormula(formula *Formula) error {
	err := dao.Q.Transaction(func(tx *dao.Query) error {
		return deleteFormulaData(tx, formula.ID)
	})
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

func (self *Depot) SaveManifestFile(formula *Formula, filename string, content string) error {
	filename, err := NormalizeFormulaFilePath(filename)
	if err != nil {
		return err
	}
	if !IsFormulaManifestPath(filename) {
		return fmt.Errorf("只允许保存 manifest 文件: %s", filename)
	}
	return self.saveFile(filepath.Join(self.basePath, formula.GetFilesRelativeDir()), filename, content)
}

func (self *Depot) SaveSharedFile(formula *Formula, filename string, content string) error {
	filename, err := NormalizeFormulaFilePath(filename)
	if err != nil {
		return err
	}
	if IsFormulaManifestPath(filename) {
		return fmt.Errorf("manifest 文件必须保存到版本目录: %s", filename)
	}
	return self.saveFile(self.getSharedFilesDir(formula.Name), filename, content)
}

func (self *Depot) saveFile(filesDir, filename, content string) error {
	filePath, err := resolveFormulaFilePath(filesDir, filename)
	if err != nil {
		return err
	}
	if content == "" {
		if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}

	function.CreateDirIfNotExist(filepath.Dir(filePath), os.ModePerm)
	return os.WriteFile(filePath, []byte(content), 0o666)
}

func resolveFormulaFilePath(filesDir, filename string) (string, error) {
	cleanFilename, err := NormalizeFormulaFilePath(filename)
	if err != nil {
		return "", err
	}
	filePath := filepath.Join(filesDir, filepath.FromSlash(cleanFilename))
	if !function.IsPathInDir(filePath, filesDir) {
		return "", fmt.Errorf("非法文件路径: %s", filename)
	}
	return filePath, nil
}

func NormalizeFormulaFilePath(filename string) (string, error) {
	normalizedFilename := strings.ReplaceAll(filename, "\\", "/")
	if strings.HasPrefix(normalizedFilename, "/") {
		return "", fmt.Errorf("非法文件路径: %s", filename)
	}

	cleanFilename, err := function.CleanZipFilePath(normalizedFilename)
	if err != nil {
		return "", err
	}
	return cleanFilename, nil
}

var formulaManifestPathPattern = regexp.MustCompile(`(^|/)manifest\.yaml$`)

func IsFormulaManifestPath(path string) bool {
	return formulaManifestPathPattern.MatchString(normalizeFormulaFilePath(path))
}

func normalizeFormulaFilePath(path string) string {
	return strings.ReplaceAll(strings.TrimPrefix(path, "/"), "\\", "/")
}

func (self *Depot) GetManifestFileList(formula *Formula) (map[string]string, error) {
	files, err := self.getFileList(filepath.Join(self.basePath, formula.GetFilesRelativeDir()))
	if err != nil {
		return nil, err
	}
	for path := range files {
		if !IsFormulaManifestPath(path) {
			delete(files, path)
		}
	}
	return files, nil
}

func (self *Depot) GetSharedFileList(formula *Formula) (map[string]string, error) {
	files, err := self.getFileList(self.getSharedFilesDir(formula.Name))
	if err != nil {
		return nil, err
	}
	for path := range files {
		if IsFormulaManifestPath(path) {
			delete(files, path)
		}
	}
	return files, nil
}

func (self *Depot) getFileList(dir string) (map[string]string, error) {
	function.CreateDirIfNotExist(dir, os.ModePerm)
	files := map[string]string{}
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[path[len(dir)+1:]] = string(content)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func (self *Depot) getSharedFilesDir(formulaName string) string {
	return filepath.Join(self.basePath, GetFormulaRelativeDir(formulaName, 0), "files")
}

func (self *Depot) GetBackendZipFileList(formula *Formula) ([]string, error) {
	result := make([]string, 0)

	if formula.ZipPath == "" {
		return result, nil
	}

	fileList, _ := self.GetManifestFileList(formula)
	sharedFiles, _ := self.GetSharedFileList(formula)
	for path, content := range sharedFiles {
		fileList[path] = content
	}
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

func (self *Depot) GetFrontendZipFileContent(formula *Formula, identifie, path string) ([]byte, error) {
	frontendIdentifie := strings.ReplaceAll(identifie, "_", "-")
	webZipPath, exists := formula.WebZipPaths[frontendIdentifie]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrFrontendPackageNotFound, identifie)
	}

	zipPath := filepath.Join(self.basePath, webZipPath)
	cacheRoot := filepath.Join(os.TempDir(), "w7panel-zpk", "zip_file_cache")
	return attachlogic.Attach{}.GetZipFileContent(cacheRoot, zipPath, path)
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

func sortManifestsByApplicationOrder(manifests []*logic.Manifest) {
	sort.SliceStable(manifests, func(i, j int) bool {
		if manifests[i] == nil {
			return false
		}
		if manifests[j] == nil {
			return true
		}
		return manifests[i].Application.Order < manifests[j].Application.Order
	})
}
