package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	copy2 "github.com/otiai10/copy"
	"github.com/w7panel/w7panel-zpk/common/accessor"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	"github.com/w7panel/w7panel-zpk/common/function"
	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/w7panel/w7panel-zpk/common/service/w7"
	"github.com/w7panel/w7panel-zpk/common/service/w7/devcenter"
	"sigs.k8s.io/yaml"
)

type CloudApp struct {
}

func (l CloudApp) GetNotAppName(formula *Formula, consoleUid int) string {
	return formula.Name + strconv.Itoa(consoleUid)
}

func (l CloudApp) UnpackNotAppToFormula(notAppId int, user *entity.RegistryUser, consoleUid int32) error {
	notAppInfo, err := w7.DevCenterNotAppSdk.GetNotAppInfo(devcenter.NotAppInfoReq{
		ConsoleUid: consoleUid,
		Id:         notAppId,
	})
	if err != nil {
		return err
	}
	notAppBranch, err := w7.DevCenterNotAppSdk.GetNotAppBranch(devcenter.NotApp{
		Id:         notAppId,
		ConsoleUid: consoleUid,
	})
	if err != nil {
		return err
	}
	//同步价格
	appBranchInfo, err := w7.DevCenterNotAppSdk.GetNotAppBranchInfo(devcenter.NotAppBranchInfoReq{
		ConsoleUid: consoleUid,
		AppId:      notAppId,
		BranchId:   notAppBranch.Id,
	})
	if err != nil {
		return err
	}
	appServicePackages, err := w7.DevCenterNotAppSdk.GetNotAppServicePackages(devcenter.GetNotAppServicePackagesReq{
		ConsoleUid: consoleUid,
		AppId:      notAppId,
	})
	if err != nil {
		return err
	}
	versionPrices, err := w7.DevCenterNotAppSdk.GetNotAppBranchVersionPriceList(devcenter.GetNotAppBranchVersionPriceListReq{
		ConsoleUid: consoleUid,
		AppId:      notAppId,
		BranchId:   notAppBranch.Id,
	})
	if err != nil {
		return err
	}
	notAppBranchVersions, err := w7.DevCenterNotAppSdk.NotAppBranchVersionList(devcenter.NotAppVersionListReq{
		ConsoleUid: consoleUid,
		AppId:      notAppId,
		BranchId:   notAppBranch.Id,
		Page:       1,
		PageSize:   1000,
	})
	if err != nil {
		return err
	}
	var latestVersion devcenter.NotAppBranchVersion
	for _, item := range notAppBranchVersions.List {
		if item.Status == 3 {
			latestVersion = item
			break
		}
	}
	if latestVersion.Id == 0 {
		return errors.New("找不到可用的应用版本")
	}
	goodsId := 0
	goodsProductId := 0
	goodsPrice := float64(0)
	productType := 0
	isFreeUpgrade := 0
	goodsLabels := make([]devcenter.Label, 0)
	goodsExt := make(map[string]interface{})
	crossUpgradeFormulas := make([]accessor.CrossUpgradeFormula, 0)
	if notAppInfo.GoodsId > 0 {
		goodsInfo, err := w7.DevCenterGoodsSdk.PublishGoodsInfo(devcenter.PublishGoodsInfoReq{
			ConsoleUid: int(consoleUid),
			Id:         notAppInfo.GoodsId,
		})
		if err != nil {
			return err
		}
		goodsId = goodsInfo.Id
		if goodsInfo.ProductsInfo != nil && len(goodsInfo.ProductsInfo) > 0 {
			goodsPrice = goodsInfo.ProductsInfo[0].Price
			goodsProductId = goodsInfo.ProductsInfo[0].Id
		}
		if goodsInfo.Labels != nil {
			goodsLabels = goodsInfo.Labels
		}
		if goodsInfo.Ext != nil {
			_, ok := goodsInfo.Ext.(map[string]interface{})
			if ok {
				goodsExt = goodsInfo.Ext.(map[string]interface{})
			}
		}
		if _, exists := goodsExt["extra"]; exists {
			extra := goodsExt["extra"].(map[string]interface{})
			if _, exists := extra["product_type"]; exists {
				tmp, ok := extra["product_type"].(string)
				if ok {
					productType, err = strconv.Atoi(tmp)
					if err != nil {
						return err
					}
				} else {
					tmp1, ok := extra["product_type"].(int)
					if ok {
						productType = tmp1
					}
				}
			}
			if _, exists := extra["is_free_upgrade"]; exists {
				tmp, ok := extra["is_free_upgrade"].(string)
				if ok {
					isFreeUpgrade, err = strconv.Atoi(tmp)
					if err != nil {
						return err
					}
				} else {
					tmp1, ok := extra["is_free_upgrade"].(int)
					if ok {
						isFreeUpgrade = tmp1
					}
				}
			}
			if rawCrossUpgradeFormulas, exists := extra["cross_upgrade_formulas"]; exists {
				if tmp, ok := rawCrossUpgradeFormulas.(string); ok {
					_ = json.Unmarshal([]byte(tmp), &crossUpgradeFormulas)
				} else {
					tmpContent, _ := json.Marshal(rawCrossUpgradeFormulas)
					_ = json.Unmarshal(tmpContent, &crossUpgradeFormulas)
				}
			}
		}
		if goodsInfo.ServiceConfig.GiveMonth > 0 && appServicePackages != nil && len(appServicePackages) > 0 {
			for i, item := range appServicePackages {
				if item.Month == goodsInfo.ServiceConfig.GiveMonth {
					item.IsGift = true
					appServicePackages[i] = item
				}
			}
		}
	}

	formulaName := strings.TrimSuffix(notAppInfo.Name, strconv.Itoa(int(consoleUid)))
	existsFormula := GetFormulaByName(formulaName)
	err = l.unpackNotAppVersionToFormula(*notAppInfo, latestVersion, user, consoleUid)
	if err != nil && existsFormula == nil {
		DeleteFormulaByName(formulaName)
		return err
	}
	curFormula := GetFormulaByName(formulaName)
	if curFormula == nil {
		return errors.New("重新获取制品失败")
	}

	updateFormula := entity.Formula{
		RemoteUID:         int32(consoleUid),
		GoodsID:           int32(goodsId),
		GoodsProductID:    int32(goodsProductId),
		InstallServiceFee: goodsPrice,
		ProductType:       int32(productType),
		IsFreeUpgrade:     int32(isFreeUpgrade),
		CrossUpgradeFormulas: &accessor.CrossUpgradeFormulasOption{
			List: crossUpgradeFormulas,
		},
	}
	if appBranchInfo.Price != nil && appBranchInfo.Price.Price != 0 {
		updateFormula.InstallServiceFee = appBranchInfo.Price.Price
	}
	if appServicePackages != nil && len(appServicePackages) > 0 {
		updateFormula.ServicePackages = &accessor.ServicePackagesOption{
			List: appServicePackages,
		}
	}
	if versionPrices != nil && len(versionPrices) > 0 {
		updateFormula.VersionPrices = &accessor.VersionPricesOption{
			List: versionPrices,
		}
	}

	err = dao.Q.Transaction(func(tx *dao.Query) error {
		if goodsLabels != nil {
			_, err = tx.TagFormula.Where(tx.TagFormula.FormulaID.Eq(curFormula.ID)).Delete()
			if err != nil {
				return err
			}

			for _, item := range goodsLabels {
				tag, _ := tx.Tag.Where(tx.Tag.Name.Eq(item.Title)).First()
				if tag == nil {
					tag = &entity.Tag{
						Name: item.Title,
					}
					err = tx.Tag.Create(tag)
					if err != nil {
						return err
					}
				}

				err = tx.TagFormula.Create(&entity.TagFormula{
					FormulaID: curFormula.ID,
					TagID:     tag.ID,
				})
				if err != nil {
					return err
				}
			}
		}

		_, err = tx.Formula.Where(tx.Formula.ID.Eq(curFormula.ID)).Updates(updateFormula)

		return err
	})
	if err != nil {
		slog.Error("update formula for unpack", "formula", formulaName, "err", err)
		return err
	}

	return nil
}

func (l CloudApp) unpackNotAppVersionToFormula(notAppInfo devcenter.NotApp, notAppVersion devcenter.NotAppBranchVersion, user *entity.RegistryUser, consoleUid int32) error {
	depot, _ := NewDepot()
	notAppDelativeDir := filepath.Join("notapp_unpack", notAppInfo.Name)
	notAppDir := filepath.Join(depot.GetBasePath(), notAppDelativeDir)
	backendZipPath := filepath.Join(notAppDir, "backend/backend.zip")
	frontendZipPath := filepath.Join(notAppDir, "frontend/frontend.zip")
	iconPath := filepath.Join(notAppDir, "icon.png")
	manifestContent := ""
	os.RemoveAll(notAppDir)
	function.CreateDirIfNotExist(filepath.Dir(backendZipPath), os.ModePerm)
	function.CreateDirIfNotExist(filepath.Dir(frontendZipPath), os.ModePerm)
	defer os.RemoveAll(notAppDir)

	//下载代码包
	for _, item := range notAppVersion.SupportTypes {
		md5, err := w7.DevCenterNotAppSdk.NotAppBranchVersionAttach(devcenter.NotAppVersionAttachReq{
			ConsoleUid:  consoleUid,
			AppId:       notAppInfo.Id,
			BranchId:    notAppVersion.BranchId,
			VersionId:   notAppVersion.Id,
			SupportType: item.SupportType,
		})
		if err != nil {
			return err
		}

		savePath := ""
		if item.SupportType == "notapp" {
			savePath = backendZipPath

			tmpContent, err1 := w7.W7CloudAttach.GetAttachContent(md5.VersionZipMd5, "manifest.xml")
			if err1 != nil && err1.IsError() {
				return err1
			}
			manifestContent = tmpContent
		} else {
			savePath = frontendZipPath
		}

		downloadUrl, err1 := w7.W7CloudAttach.GetAttachDownloadUrl(md5.VersionZipMd5)
		if err1 != nil && err1.IsError() {
			return err1
		}

		err = function.DownloadFile(context.Background(), downloadUrl, savePath, nil)
		if err != nil {
			return err
		}
	}
	if notAppInfo.Logo != "" {
		err := function.DownloadFile(context.Background(), notAppInfo.Logo, iconPath, nil)
		if err != nil {
			return err
		}
	}

	remoteNotAppName := notAppInfo.Name
	notAppInfo.Name = strings.TrimSuffix(notAppInfo.Name, strconv.Itoa(int(consoleUid)))
	err := depot.AddFormula(notAppInfo.Name, notAppVersion.Version, user)
	if err != nil {
		return err
	}
	curFormula, err := depot.GetFormula(notAppInfo.Name, notAppVersion.Version, user)
	if err != nil {
		return err
	}

	webZipPath := ""
	if function.FileExists(frontendZipPath) {
		webZipPath, err = l.unpackNotAppVersionFrontendPkg(curFormula, frontendZipPath, remoteNotAppName)
		slog.Info("unpackNotAppVersionFrontendPkg", "formula", curFormula.Name, "version", notAppVersion.Version, "remoteNotAppName", remoteNotAppName, "err", err)
		if err != nil {
			return err
		}
	}
	codeZipPath := ""
	if function.FileExists(backendZipPath) {
		codeZipPath, err = l.unpackNotAppVersionBackendPkg(curFormula, backendZipPath, remoteNotAppName)
		slog.Info("unpackNotAppVersionBackendPkg", "formula", curFormula.Name, "version", notAppVersion.Version, "remoteNotAppName", remoteNotAppName, "err", err)
		if err != nil {
			return err
		}
	}
	err = copy2.Copy(iconPath, filepath.Join(depot.GetBasePath(), curFormula.GetIconRelativePath()))
	if err != nil {
		return err
	}
	manifestRow := &logic2.Manifest{}
	err = yaml.Unmarshal([]byte(manifestContent), manifestRow)
	if err != nil {
		return err
	}
	manifestRow.Source.Url = "file://" + codeZipPath
	manifestRow.Source.Type = "zip"
	if webZipPath != "" {
		manifestRow.Web.Url = "file://" + webZipPath
		manifestRow.Web.Type = "zip"
	}
	if manifestRow.Application.Name == "" {
		manifestRow.Application.Name = notAppInfo.Title
	}
	if manifestRow.Application.Description == "" {
		manifestRow.Application.Description = notAppInfo.Description
	}
	manifestRow.Application.Identifie = strings.ReplaceAll(notAppInfo.Name, "_", "-")
	newManifestContent, err := yaml.Marshal(manifestRow)
	if err != nil {
		return err
	}

	formulaFilesDir := filepath.Join(depot.GetBasePath(), curFormula.GetFilesRelativeDir())
	function.CreateDirIfNotExist(formulaFilesDir, os.ModePerm)
	err = os.WriteFile(filepath.Join(formulaFilesDir, "manifest.yaml"), newManifestContent, os.ModePerm)
	if err != nil {
		return err
	}

	err = depot.Pack(curFormula, false)
	if err != nil {
		return err
	}

	return dao.Q.Transaction(func(tx *dao.Query) error {
		_, err = tx.Version.Where(tx.Version.FormulaID.Eq(curFormula.ID)).Where(tx.Version.Name.Eq(notAppVersion.Version)).Updates(entity.Version{
			Description: notAppVersion.Description,
		})
		if err != nil {
			return err
		}
		_, err = tx.Formula.Where(tx.Formula.ID.Eq(curFormula.ID)).Updates(entity.Formula{
			Title: manifestRow.Application.Name,
		})

		return err
	})
}

func (l CloudApp) unpackNotAppVersionBackendPkg(formula *Formula, zipPath string, remoteNotAppName string) (string, error) {
	depot, _ := NewDepot()
	backendDir := filepath.Dir(zipPath)
	slog.Info("执行命令", "cmd", "unzip", zipPath)
	cmd := exec.Command("unzip", zipPath, "-d", backendDir)
	message, err := cmd.CombinedOutput()
	slog.Info("执行命令完成", "cmd", "unzip", zipPath, "dir", backendDir, "message", string(message), "err", err)
	if err != nil {
		return "", err
	}

	pathInfo := function.GetPathInfo(zipPath)
	formulaBackendSavePath := fmt.Sprintf("/Storage/%s/%s%s", time.Now().Format("200601"), function.GetMd5(pathInfo.Filename+formula.Name+formula.Version), pathInfo.Extension)

	backendDir = filepath.Join(backendDir, remoteNotAppName)

	os.Remove(zipPath)

	formulaFilesDir := filepath.Join(depot.GetBasePath(), formula.GetFilesRelativeDir())
	function.CreateDirIfNotExist(formulaFilesDir, os.ModePerm)
	//复制所有 manifest文件到 formula对应的文件目录
	err = filepath.Walk(backendDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录，只处理文件
		if info.IsDir() {
			return nil
		}

		// 匹配目标文件名
		if filepath.Base(path) == "manifest.yaml" {
			// 获取文件相对路径
			relPath, _ := filepath.Rel(backendDir, filepath.Dir(path))
			// 创建目标目录结构
			targetDir := filepath.Join(formulaFilesDir, relPath)
			function.CreateDirIfNotExist(targetDir, os.ModePerm)

			// 构建目标文件路径
			destPath := filepath.Join(targetDir, "manifest.yaml")

			err = copy2.Copy(path, destPath)
			if err != nil {
				return err
			}
			os.Remove(path)
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	slog.Info("执行命令", "cmd", "zip -r", formula.Name+"_backend.zip", "dir", backendDir)
	cmd = exec.Command("zip", "-r", formula.Name+"_backend.zip", "./")
	cmd.Dir = backendDir
	message, err = cmd.CombinedOutput()
	slog.Info("执行命令完成", "cmd", "zip -r", formula.Name+"_backend.zip", "dir", filepath.Dir(backendDir), "message", string(message), "err", err)
	if err != nil {
		return "", err
	}

	err = copy2.Copy(filepath.Join(backendDir, formula.Name+"_backend.zip"), filepath.Join(depot.GetBasePath(), formulaBackendSavePath))
	if err != nil {
		return "", err
	}
	return formulaBackendSavePath, nil
}

func (l CloudApp) unpackNotAppVersionFrontendPkg(formula *Formula, zipPath string, remoteNotAppName string) (string, error) {
	frontendDir := filepath.Dir(zipPath)
	slog.Info("执行命令", "cmd", "unzip", zipPath)
	cmd := exec.Command("unzip", zipPath, "-d", frontendDir)
	message, err := cmd.CombinedOutput()
	slog.Info("执行命令完成", "cmd", "unzip", zipPath, "dir", frontendDir, "message", string(message), "err", err)
	if err != nil {
		return "", err
	}

	pathInfo := function.GetPathInfo(zipPath)
	formulaFrontendSavePath := fmt.Sprintf("/Storage/%s/%s%s", time.Now().Format("200601"), function.GetMd5(pathInfo.Filename+formula.Name+formula.Version), pathInfo.Extension)
	frontendDir = filepath.Join(frontendDir, remoteNotAppName)

	slog.Info("执行命令", "cmd", "zip -r", formula.Name+"_backend.zip", "dir", frontendDir)
	cmd = exec.Command("zip", "-r", formula.Name+"_frontend.zip", "./")
	cmd.Dir = frontendDir
	message, err = cmd.CombinedOutput()
	slog.Info("执行命令完成", "cmd", "zip -r", formula.Name+"_frontend.zip", "dir", filepath.Dir(frontendDir), "message", string(message), "err", err)
	if err != nil {
		return "", err
	}

	depot, _ := NewDepot()
	err = copy2.Copy(filepath.Join(frontendDir, formula.Name+"_frontend.zip"), filepath.Join(depot.GetBasePath(), formulaFrontendSavePath))
	if err != nil {
		return "", err
	}
	return formulaFrontendSavePath, nil
}
