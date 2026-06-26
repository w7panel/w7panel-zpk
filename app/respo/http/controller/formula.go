package controller

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/app/respo/logic"
	"github.com/w7panel/w7panel-zpk/common/accessor"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	"github.com/w7panel/w7panel-zpk/common/function"
	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/w7panel/w7panel-zpk/common/service/w7"
	"github.com/w7panel/w7panel-zpk/common/service/w7/devcenter"
	"github.com/w7panel/w7panel-zpk/common/service/w7/ip"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"github.com/we7coreteam/w7-rangine-go/v2/src/core/err_handler"
	"gorm.io/gen/field"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

type Formula struct {
	Abstract
}

func (c Formula) Add(ctx *gin.Context) {
	type ParamsValidate struct {
		Identifie string `form:"identifie" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	depotLogin := c.getDepot()
	err := depotLogin.AddFormula(params.Identifie, "1.0.0", logic2.User{}.GetUser(ctx))
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}

	c.JsonSuccessResponse(ctx)
}

func (c Formula) BaseInfo(ctx *gin.Context) {
	type ParamsValidate struct {
		Identifie string `form:"identifie" json:"identifie" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	depotLogin := c.getDepot()
	formula, err := depotLogin.GetFormula(params.Identifie, "", nil)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	latestVersion, _ := dao.Q.Version.Where(dao.Q.Version.FormulaID.Eq(formula.ID)).Order(dao.Q.Version.ID.Desc()).First()
	if latestVersion != nil {
		formula.Version = latestVersion.Name
	}
	c.JsonResponseWithoutError(ctx, map[string]interface{}{"latest_version": formula.Version})
}

func (c Formula) Info(ctx *gin.Context) {
	type ParamsValidate struct {
		Identifie    string `uri:"id" json:"identifie" binding:"required"`
		Version      string `uri:"version" json:"version"`
		CName        string `uri:"cid" json:"cname"`
		IsUpgrade    int32  `form:"is_upgrade" json:"is_upgrade"`
		CheckUpgrade int32  `form:"check_upgrade" json:"check_upgrade"`
		CurVersion   string `form:"cur_version" json:"cur_version"`
		Token        string `form:"token" json:"token"`
		OrderSn      string `form:"order_sn" json:"order_sn"`
		ConsoleUid   int32  `form:"console_uid" json:"console_uid"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	depotLogin := c.getDepot()
	formula, err := depotLogin.GetFormula(params.Identifie, params.Version, nil)
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}

	consoleUid := logic2.User{}.GetConsoleUid(ctx)
	if consoleUid == 0 && ctx.GetString("appid") != "" {
		consoleUid = params.ConsoleUid
	}
	canUpgradeVersion := ""
	formulaExpire := false
	targetFormulaIdentify := ""
	if params.Token != "" {
		if err := w7.ZpkMarketSdk.CheckToken(params.Token, formula.Name); err != nil {
			c.JsonResponseWithServerError(ctx, err)
			return
		}
	} else {
		ok, formulaIdentify := logic.Order{}.CheckFormulaCanInstallOrUpgrade(*formula, consoleUid, params.OrderSn, params.IsUpgrade > 0)
		if !ok {
			c.JsonResponseWithError(ctx, errors.New("请先购买后再安装"), 500)
			return
		}
		canUpgradeVersion, formulaExpire, formulaIdentify, err = logic.Order{}.GetFormulaCanUpgradeVersion(*formula, consoleUid, params.OrderSn)
		slog.Info("formula can upgrade version", "formula", formula.Name, "consoleUid", consoleUid, "version", canUpgradeVersion, "formulaIdentify", formulaIdentify, "err", err)
		if err != nil {
			c.JsonResponseWithError(ctx, err, 500)
			return
		}
		if formulaIdentify != "" && formulaIdentify != formula.Name {
			targetFormulaIdentify = formulaIdentify
		}
	}
	if targetFormulaIdentify != "" {
		slog.Info("switch formula by order formula identify", "from", formula.Name, "to", targetFormulaIdentify, "orderSn", params.OrderSn)
		formula, err = depotLogin.GetFormula(targetFormulaIdentify, params.Version, nil)
		if err != nil {
			c.JsonResponseWithError(ctx, err, 500)
			return
		}
		params.CurVersion = formula.Version
	}

	var version *entity.Version
	if params.CurVersion != "" {
		if params.IsUpgrade > 0 {
			version, _ = logic.Version{}.FindNextUpgrade(formula, params.CurVersion, canUpgradeVersion)
		} else {
			version, _ = dao.Version.
				Where(dao.Version.FormulaID.Eq(formula.ID)).
				Where(dao.Q.Version.Name.Eq(params.CurVersion)).
				Where(dao.Q.Version.PublishStatus.In(logic.FormulaPublishStatusSuccess, 0)).
				First()
		}
	}
	if version == nil && canUpgradeVersion != "" {
		version, _ = dao.Version.Where(dao.Version.Name.Eq(canUpgradeVersion)).First()
	}
	if version == nil {
		version, _ = dao.Version.Where(dao.Version.ID.Eq(formula.LatestVersionId)).First()
	}
	if params.CurVersion != "" && version != nil && version.ID != formula.VersionId {
		formula, err = depotLogin.GetFormula(formula.Name, version.Name, nil)
		if err != nil {
			c.JsonResponseWithError(ctx, err, 500)
			return
		}
	}

	schemaHttp := "https://"
	_ = depotLogin.GetFormulaBackendZipDownloadUrl(formula, false)
	zipUrl := depotLogin.GetFormulaBackendZipDownloadUrl(formula, true)
	domain := facade.GetConfig().GetString("setting.depot.external_domain")
	webzipUrl := map[string]string{}
	if len(formula.WebZipPaths) > 0 {
		for k, v := range formula.WebZipPaths {
			token := function.GetRandomString(20)
			dUrl := fmt.Sprintf("https://%s/zpk/zip/download/%s", domain, token)
			depotLogin.DownloadMapping.Store(token, v)
			webzipUrl[k] = dUrl
		}
	}
	helmPackageUrl := depotLogin.GetFormulaHelmDownloadUrl(formula)

	responseManifest := *formula.Manifest
	if params.CName != "" {
		for _, item := range formula.AllManifest {
			if item.Application.Identifie == params.CName {
				responseManifest = *item
				break
			}
		}
	}

	type FormulaInstallInfo struct {
		Name        string               `json:"name"`
		Title       string               `json:"title"`
		Required    bool                 `json:"required"`
		StartParams []logic2.StartParams `json:"start_params"`
		RequirePvc  bool                 `json:"requirepvc"`
		Volumes     []v1.Volume          `json:"volumes"`
	}
	installFormulas := make([]FormulaInstallInfo, 0)
	if params.CName == "" {
		if formula.Manifest.Platform.StartParams == nil {
			formula.Manifest.Platform.StartParams = make([]logic2.StartParams, 0)
		}
		if formula.Manifest.Platform.Volumes == nil {
			formula.Manifest.Platform.Volumes = make([]v1.Volume, 0)
		}
		needPvc := false
		for _, item := range formula.Manifest.Platform.Volumes {
			if item.PersistentVolumeClaim != nil {
				needPvc = true
				break
			}
		}
		installFormulas = append(installFormulas, FormulaInstallInfo{
			Name:        formula.Manifest.Application.Identifie,
			Title:       formula.Manifest.Application.Name,
			Required:    true,
			StartParams: formula.Manifest.Platform.StartParams,
			RequirePvc:  needPvc,
			Volumes:     formula.Manifest.Platform.Volumes,
		})
		formulaRequiredMap := make(map[string]bool)
		for _, item := range formula.Manifest.Platform.Depends {
			formulaRequiredMap[item.Identifie] = item.Required
		}
		for _, item := range formula.AllManifest {
			if item.Application.Identifie != formula.Manifest.Application.Identifie {
				ineedPvc := false
				for _, iitem := range item.Platform.Volumes {
					if iitem.PersistentVolumeClaim != nil {
						ineedPvc = true
						break
					}
				}
				if item.Platform.StartParams == nil {
					item.Platform.StartParams = make([]logic2.StartParams, 0)
				}
				if item.Platform.Volumes == nil {
					item.Platform.Volumes = make([]v1.Volume, 0)
				}
				installFormulas = append(installFormulas, FormulaInstallInfo{
					Name:        item.Application.Identifie,
					Title:       item.Application.Name,
					Required:    formulaRequiredMap[item.Application.Identifie],
					StartParams: item.Platform.StartParams,
					RequirePvc:  ineedPvc,
					Volumes:     item.Platform.Volumes,
				})
			}
		}
	}

	servicePackages := make([]devcenter.NotAppServicePackage, 0)
	if formula.ServicePackages != nil && formula.ServicePackages.List != nil {
		servicePackages = formula.ServicePackages.List
	}
	versionPrices := make([]devcenter.NotAppBranchVersionPriceInfo, 0)
	if formula.VersionPrices != nil && formula.VersionPrices.List != nil {
		versionPrices = formula.VersionPrices.List
	}
	crossUpgradeFormulas := make([]accessor.CrossUpgradeFormula, 0)
	if formula.Setting != nil && formula.Setting.SupportCrossUpgrade && formula.CrossUpgradeFormulas != nil && formula.CrossUpgradeFormulas.List != nil {
		for _, item := range formula.CrossUpgradeFormulas.List {
			item.Icon = "/zpk/zip/icon/" + item.Identifie
			crossUpgradeFormulas = append(crossUpgradeFormulas, item)
		}
	}
	ticket, _ := logic.Ticket{}.GetTicket(logic.TicketInfo{
		FormulaId:      formula.ID,
		ConsoleUid:     consoleUid,
		FormulaVersion: version.Name,
		OrderSn:        params.OrderSn,
		IsUpgrade:      params.IsUpgrade > 0,
	})

	if formula.IsFreeUpgrade == -1 {
		formula.IsFreeUpgrade = 0
	}

	responseManifest.Version = 3
	responseManifest.VersionV2 = 3
	tmpContent, _ := yaml.Marshal(responseManifest)
	responseManifestMap := map[string]interface{}{}
	_ = yaml.Unmarshal(tmpContent, &responseManifestMap)
	if platform, ok := responseManifestMap["platform"].(map[string]interface{}); ok {
		delete(platform, "container")
	}
	tmpContent, _ = yaml.Marshal(responseManifestMap)
	manifestContent := string(tmpContent)

	c.JsonResponseWithoutError(ctx, gin.H{
		"zip_url":                zipUrl,
		"manifest":               manifestContent,
		"version":                version,
		"webzip_url":             webzipUrl,
		"icon_url":               fmt.Sprintf("%s%s%s", schemaHttp, domain, formula.Icon),
		"install_service_fee":    formula.InstallServiceFee,
		"service_packages":       servicePackages,
		"version_prices":         versionPrices,
		"cross_upgrade_formulas": crossUpgradeFormulas,
		"is_free_upgrade":        formula.IsFreeUpgrade,
		"product_type":           formula.ProductType,
		"ticket":                 ticket,
		"service_expire":         formulaExpire,
		"goods_id":               formula.GoodsId,
		"helm_url":               helmPackageUrl,
		"tags":                   formula.Tags,
		"install_formulas":       installFormulas,
		"formula_type":           formula.Manifest.Application.Type,
	})
}

func (c Formula) Detail(ctx *gin.Context) {
	type ParamsValidate struct {
		Identifie string `uri:"id" binding:"required"`
		Version   string `uri:"version"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	depotLogin := c.getDepot()
	formula, err := depotLogin.GetFormula(params.Identifie, params.Version, logic2.User{}.GetUser(ctx))
	if err_handler.Found(err) {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}

	fileList, err := depotLogin.GetFileList(formula)
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}
	detailMds := map[string]string{}
	for key, val := range fileList {
		if strings.HasSuffix(key, "manifest.yaml") {
			continue
		}
		detailMds[key] = val
	}

	var versionList []map[string]interface{}
	err = dao.Version.
		Where(dao.Version.FormulaID.Eq(formula.ID)).
		Limit(10).Select(dao.Version.Name, dao.Version.ID, dao.Version.Description).
		Where(dao.Q.Version.PublishStatus.In(logic.FormulaPublishStatusSuccess, 0)).
		Order(dao.Version.ID.Desc()).
		Scan(&versionList)
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}

	c.JsonResponseWithoutError(ctx, gin.H{
		"title":        formula.Manifest.Application.Name,
		"description":  formula.Manifest.Application.Description,
		"icon":         formula.Icon,
		"mds":          detailMds,
		"version_list": versionList,
	})
	return
}

func (c Formula) Delete(ctx *gin.Context) {
	type ParamsValidate struct {
		Identifie string `form:"identifie" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	depotLogin := c.getDepot()
	formula, err := depotLogin.GetFormula(params.Identifie, "", logic2.User{}.GetUser(ctx))
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}

	err = depotLogin.DeleteFormula(formula)
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}

	c.JsonSuccessResponse(ctx)
}

func (c Formula) List(ctx *gin.Context) {
	depotLogin := c.getDepot()
	type ParamsValidate struct {
		Limit   int     `form:"limit,default=30" binding:"omitempty"`
		Page    int     `form:"page,default=1" binding:"omitempty,gt=0"`
		Tag     string  `form:"tag" binding:"omitempty"`
		Keyword string  `form:"keyword" binding:"omitempty"`
		Sort    string  `form:"sort,default=new" binding:"omitempty,oneof=hot new"`
		Status  []int32 `form:"status"`
		Owner   bool    `form:"owner"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	if params.Page < 1 {
		params.Page = 1
	}
	if len(params.Status) == 0 {
		params.Status = append(params.Status, logic.FORMULA_DISPLAY)
		params.Status = append(params.Status, logic.FORMULA_RECOMMEND)
	}

	type ResultNode struct {
		Name                 string                 `json:"name"`
		Description          string                 `json:"description"`
		Identifie            string                 `json:"identifie"`
		Icon                 string                 `json:"icon"`
		Tag                  []entity.Tag           `json:"tag"`
		InstallTotal         int32                  `json:"install_total"`
		Version              *entity.Version        `json:"version"`
		Status               int32                  `json:"status"`
		ProductType          int32                  `json:"product_type"`
		InstallOnlyOnce      bool                   `json:"install_only_once"`
		RemoteFormulaInfoURL string                 `json:"remote_formula_info_url"`
		AuditStatus          int32                  `json:"audit_status"`
		AuditRemark          string                 `json:"audit_remark"`
		Annotation           map[string]interface{} `yaml:"annotation" json:"annotation"`
		InstallUsersAvatar   []string               `json:"install_users_avatar"`
		GoodsId              int32                  `json:"goods_id"`
	}

	var user *entity.RegistryUser
	if params.Owner {
		user = logic2.User{}.GetUser(ctx)
		if user == nil {
			params.Owner = false
		}
	}

	var result []ResultNode
	query := dao.Q.Formula.Preload(dao.Formula.Tag, dao.Formula.Version)
	if params.Owner {
		query = query.Where(
			field.Or(
				dao.Q.Formula.RemoteFormulaInfoURL.Eq(""),
				dao.Q.Formula.RemoteFormulaInfoURL.IsNull(),
			),
		)
	}
	query = query.Where(dao.Q.Formula.Status.In(params.Status...))
	if params.Tag != "" {
		searchTag, _ := dao.Q.Tag.Where(dao.Q.Tag.Name.Eq(params.Tag)).First()
		if searchTag == nil {
			c.JsonResponseWithError(ctx, errors.New("标签不存在"), 500)
			return
		}
		formulaTagList, _, _ := dao.Q.TagFormula.Where(dao.Q.TagFormula.TagID.Eq(searchTag.ID)).
			FindByPage((params.Page-1)*params.Limit, params.Limit)
		if formulaTagList != nil {
			var searchFormulaIds []int32
			for _, tagItem := range formulaTagList {
				searchFormulaIds = append(searchFormulaIds, tagItem.FormulaID)
			}
			query = query.Where(dao.Q.Formula.ID.In(searchFormulaIds...))
		}
	}
	if params.Keyword != "" {
		query = query.Where(dao.Q.Formula.Title.Like("%" + params.Keyword + "%"))
	}
	if params.Sort == "new" {
		query = query.Order(dao.Formula.ID.Desc())
	}
	if params.Sort == "hot" {
		query = query.Order(dao.Formula.Status.Desc(), dao.Formula.InstallTotal.Desc())
	}
	isAdminUser := false
	if user != nil {
		isAdminUser = logic2.User{}.IsAdminUser(user)
		if !isAdminUser {
			query = query.Where(dao.Q.Formula.UserID.Eq(user.ID))
		}
	}

	formulaList, total, _ := query.FindByPage((params.Page-1)*params.Limit, params.Limit)
	goodsMap := make(map[int]ip.GoodsListItem)
	if params.Owner && formulaList != nil {
		goodsIds := make([]int, 0)
		goodsIdExists := make(map[int]struct{})
		for _, item := range formulaList {
			goodsId := int(item.GoodsID)
			if goodsId <= 0 {
				continue
			}
			if _, ok := goodsIdExists[goodsId]; ok {
				continue
			}
			goodsIds = append(goodsIds, goodsId)
			goodsIdExists[goodsId] = struct{}{}
		}
		if len(goodsIds) > 0 {
			goodsList, err := w7.IpGoodsSdk.GoodsBatchList(ip.GoodsBatchListReq{
				GoodsIds: goodsIds,
			})
			if err != nil {
				c.JsonResponseWithError(ctx, err, 500)
				return
			}
			for _, item := range goodsList {
				if item.Id > 0 {
					goodsMap[item.Id] = item
				}
			}
		}
	}
	if formulaList != nil {
		for _, item := range formulaList {
			formula, err := depotLogin.GetFormula(item.Name, "", user)
			if err == nil {
				auditStatus := int32(0)
				auditMessage := ""
				if goodsInfo, ok := goodsMap[int(item.GoodsID)]; ok {
					auditStatus = int32(goodsInfo.AuditStatus)
					auditMessage = goodsInfo.AuditMessage
				}
				resultItem := ResultNode{
					Name:                 item.Title,
					Description:          formula.Manifest.Application.Description,
					Identifie:            item.Name,
					Icon:                 formula.Icon,
					Tag:                  item.Tag,
					Version:              item.Version,
					Status:               item.Status,
					ProductType:          formula.ProductType,
					InstallOnlyOnce:      formula.Manifest.Application.InstallOnlyOnce,
					AuditStatus:          auditStatus,
					AuditRemark:          auditMessage,
					RemoteFormulaInfoURL: "",
					Annotation:           formula.Manifest.Application.Annotation,
					GoodsId:              item.GoodsID,
				}
				result = append(result, resultItem)
			}
		}
	}

	c.JsonResponseWithoutError(ctx, gin.H{
		"total":  total,
		"limit":  params.Limit,
		"page":   params.Page,
		"list":   result,
		"webUrl": "https://" + facade.GetConfig().GetString("setting.depot.external_domain") + "/zpk",
	})
	return
}

func (c Formula) Status(ctx *gin.Context) {
	type ParamsValidate struct {
		Identifie string `form:"identifie" binding:"required"`
		Status    int    `form:"status" binding:"required,oneof=1 2 99"`
	}

	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	depotLogin := c.getDepot()
	formula, err := depotLogin.GetFormula(params.Identifie, "", logic2.User{}.GetUser(ctx))
	if err != nil {
		c.JsonResponseWithError(ctx, errors.New("请先添加仓库"), 500)
		return
	}

	if formula.GoodsId > 0 {
		status := params.Status
		if status > 2 {
			status = 2
		}
		err = w7.DevCenterGoodsSdk.ChangeGoodsStatus(devcenter.ChangeGoodsStatusReq{
			ConsoleUid: int(logic2.User{}.GetConsoleUid(ctx)),
			GoodsId:    int(formula.GoodsId),
			Status:     status,
		})
		if err != nil {
			c.JsonResponseWithError(ctx, err, 500)
			return
		}
	}

	_, err = dao.Formula.Where(dao.Formula.ID.Eq(formula.ID)).Update(dao.Formula.Status, params.Status)
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}
	c.JsonSuccessResponse(ctx)
}

func (c Formula) InstallComplete(ctx *gin.Context) {
	type ParamsValidate struct {
		Ticket string `form:"ticket" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	ticketInfo, err := logic.Ticket{}.ParseTicket(params.Ticket)
	slog.Info("核销订单", "ticket", params.Ticket, "info", ticketInfo, "err", err)
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}
	err = logic.Order{}.UseOrder(*ticketInfo)
	slog.Info("核销订单完成", "ticket", params.Ticket, "info", ticketInfo, "err", err)
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}
	c.JsonSuccessResponse(ctx)
}

func (c Formula) UnInstallComplete(ctx *gin.Context) {
	type ParamsValidate struct {
		Ticket string `form:"ticket" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	ticketInfo, err := logic.Ticket{}.ParseTicket(params.Ticket)
	slog.Info("废弃订单", "ticket", params.Ticket, "info", ticketInfo, "err", err)
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}
	err = logic.Order{}.DiscardUsedOrder(*ticketInfo)
	slog.Info("废弃订单完成", "ticket", params.Ticket, "info", ticketInfo, "err", err)
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}
	c.JsonSuccessResponse(ctx)
}
