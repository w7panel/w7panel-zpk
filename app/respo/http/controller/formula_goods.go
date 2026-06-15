package controller

import (
	"errors"
	"slices"
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/app/respo/logic"
	"github.com/w7panel/w7panel-zpk/common/accessor"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	"github.com/w7panel/w7panel-zpk/common/function"
	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/w7panel/w7panel-zpk/common/service/w7"
	"github.com/w7panel/w7panel-zpk/common/service/w7/devcenter"
)

type FormulaGoods struct {
	Abstract
}

func (c FormulaGoods) GetCanFeeUpgradeVersions(ctx *gin.Context) {
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
		c.JsonResponseWithError(ctx, errors.New("仓库不存在"), 500)
		return
	}

	versionModels, _ := dao.Q.Version.Where(dao.Q.Version.FormulaID.Eq(formula.ID)).Find()
	versions := make([]string, 0)
	for _, item := range versionModels {
		versions = append(versions, item.Name)
	}
	canUpgradeVersion := make([]int, 0)
	majorVersion, err := function.ExtractMajorVersions(versions)
	if err == nil {
		sort.Ints(majorVersion)
		if len(majorVersion) > 0 {
			majorVersion = majorVersion[1:]
		}
		canUpgradeVersion = majorVersion
		if formula.VersionPrices != nil && formula.VersionPrices.List != nil {
			for _, item := range formula.VersionPrices.List {
				if !slices.Contains(majorVersion, int(item.Version)) {
					canUpgradeVersion = append(canUpgradeVersion, int(item.Version))
				}
			}
		}
	}
	if !slices.Contains(canUpgradeVersion, logic.FormulaVersionElse) {
		canUpgradeVersion = append(canUpgradeVersion, logic.FormulaVersionElse)
	}

	c.JsonResponseWithoutError(ctx, canUpgradeVersion)
}

func (c FormulaGoods) SetServiceFee(ctx *gin.Context) {
	type ParamsValidate struct {
		Identifie         string                                   `form:"identifie" json:"identifie" binding:"required"`
		ProductType       int                                      `form:"product_type" json:"product_type" binding:"required"`
		InstallServiceFee float64                                  `form:"service_fee" json:"service_fee" binding:"required"`
		IsFreeUpgrade     int                                      `form:"is_free_upgrade" json:"is_free_upgrade"`
		ServicePackages   []devcenter.NotAppServicePackage         `form:"service_packages" json:"service_packages"`
		VersionPrices     []devcenter.NotAppBranchVersionPriceInfo `form:"version_prices" json:"version_prices"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	if params.IsFreeUpgrade > 0 {
		params.IsFreeUpgrade = 1
	} else {
		params.IsFreeUpgrade = -1
	}

	depotLogin := c.getDepot()
	formula, err := depotLogin.GetFormula(params.Identifie, "", logic2.User{}.GetUser(ctx))
	if err != nil {
		c.JsonResponseWithError(ctx, errors.New("制品不存在"), 500)
		return
	}
	if formula.ProductType == logic.FORMULA_PRODUCT_CONSOLE_APP && int32(params.ProductType) == logic.FORMULA_PRODUCT_LOCAL_APP {
		params.ServicePackages = formula.ServicePackages.List
		params.VersionPrices = formula.VersionPrices.List
	} else {
		for i, item := range params.ServicePackages {
			if item.Id == 0 {
				params.ServicePackages[i].Id = i + 1
			}
		}
		for i, item := range params.VersionPrices {
			if item.Id == 0 {
				params.VersionPrices[i].Id = i + 1
			}
		}
	}
	if int32(params.ProductType) == logic.FORMULA_PRODUCT_CONSOLE_APP {
		params.IsFreeUpgrade = -1
	}

	_, err = dao.Q.Formula.Where(dao.Formula.ID.Eq(formula.ID)).Updates(entity.Formula{
		ProductType:       int32(params.ProductType),
		InstallServiceFee: params.InstallServiceFee,
		ServicePackages: &accessor.ServicePackagesOption{
			List: params.ServicePackages,
		},
		VersionPrices: &accessor.VersionPricesOption{
			List: params.VersionPrices,
		},
		IsFreeUpgrade: int32(params.IsFreeUpgrade),
	})
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}
	c.JsonSuccessResponse(ctx)
}

func (c FormulaGoods) SetCrossUpgradeFormulas(ctx *gin.Context) {
	type ParamsValidate struct {
		Identifie            string   `form:"identifie" json:"identifie" binding:"required"`
		CrossUpgradeFormulas []string `form:"cross_upgrade_formulas" json:"cross_upgrade_formulas"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	depotLogin := c.getDepot()
	formula, err := depotLogin.GetFormula(params.Identifie, "", logic2.User{}.GetUser(ctx))
	if err != nil {
		c.JsonResponseWithError(ctx, errors.New("制品不存在"), 500)
		return
	}

	crossUpgradeFormulas, err := c.getCrossUpgradeFormulasByIdentifies(formula.ID, params.CrossUpgradeFormulas, logic2.User{}.GetUser(ctx))
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}

	_, err = dao.Q.Formula.Where(dao.Formula.ID.Eq(formula.ID)).Updates(entity.Formula{
		CrossUpgradeFormulas: &accessor.CrossUpgradeFormulasOption{
			List: crossUpgradeFormulas,
		},
	})
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}
	c.JsonSuccessResponse(ctx)
}

func (c FormulaGoods) getCrossUpgradeFormulasByIdentifies(formulaID int32, identifies []string, user *entity.RegistryUser) ([]accessor.CrossUpgradeFormula, error) {
	if len(identifies) == 0 {
		return []accessor.CrossUpgradeFormula{}, nil
	}

	query := dao.Q.Formula.Where(dao.Q.Formula.ID.Neq(formulaID)).
		Where(dao.Q.Formula.Name.In(identifies...)).
		Where(dao.Q.Formula.GoodsID.Gt(0))
	if user != nil && !(logic2.User{}).IsAdminUser(user) {
		query = query.Where(dao.Q.Formula.UserID.Eq(user.ID))
	}
	rows, err := query.Find()
	if err != nil {
		return nil, err
	}

	formulaMap := make(map[string]*entity.Formula, len(rows))
	for _, row := range rows {
		formulaMap[row.Name] = row
	}
	list := make([]accessor.CrossUpgradeFormula, 0, len(rows))
	for _, identifie := range identifies {
		row, ok := formulaMap[identifie]
		if !ok {
			continue
		}
		list = append(list, accessor.CrossUpgradeFormula{
			Identifie:      row.Name,
			Title:          row.Title,
			GoodsID:        row.GoodsID,
			GoodsProductID: row.GoodsProductID,
			Price:          row.InstallServiceFee,
		})
	}
	return list, nil
}

func (c FormulaGoods) GetCrossUpgradeFormulaCandidates(ctx *gin.Context) {
	type ParamsValidate struct {
		Identifie string `form:"identifie" json:"identifie" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	depotLogin := c.getDepot()
	formula, err := depotLogin.GetFormula(params.Identifie, "", logic2.User{}.GetUser(ctx))
	if err != nil {
		c.JsonResponseWithError(ctx, errors.New("制品不存在"), 500)
		return
	}

	query := dao.Q.Formula.Where(dao.Q.Formula.ID.Neq(formula.ID)).
		Where(dao.Q.Formula.GoodsID.Gt(0)).
		Where(dao.Q.Formula.Status.In(logic.FORMULA_DISPLAY, logic.FORMULA_RECOMMEND)).
		Order(dao.Q.Formula.ID.Desc())
	user := logic2.User{}.GetUser(ctx)
	if user != nil && !(logic2.User{}).IsAdminUser(user) {
		query = query.Where(dao.Q.Formula.UserID.Eq(user.ID))
	}
	rows, err := query.Find()
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}

	list := make([]accessor.CrossUpgradeFormula, 0, len(rows))
	for _, row := range rows {
		list = append(list, accessor.CrossUpgradeFormula{
			Identifie:      row.Name,
			Title:          row.Title,
			GoodsID:        row.GoodsID,
			GoodsProductID: row.GoodsProductID,
			Price:          row.InstallServiceFee,
		})
	}
	c.JsonResponseWithoutError(ctx, list)
}

func (c FormulaGoods) GetGoodsLabels(ctx *gin.Context) {
	type ParamsValidate struct {
		Title    string `form:"title"`
		Page     int    `form:"page"`
		PageSize int    `form:"page_size"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	if params.PageSize == 0 {
		params.PageSize = 1000
	}

	labels, err := w7.DevCenterGoodsSdk.GoodsLabels(devcenter.GoodsLabelsReq{
		Title:    params.Title,
		Page:     params.Page,
		PageSize: params.PageSize,
	})
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}

	c.JsonResponseWithoutError(ctx, labels)
}

func (c FormulaGoods) GetGoodsAuditStatus(ctx *gin.Context) {
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
		c.JsonResponseWithError(ctx, errors.New("制品不存在"), 500)
		return
	}
	if formula.GoodsId == 0 {
		c.JsonResponseWithoutError(ctx, gin.H{"audit_status": 0})
		return
	}

	goodsInfo, err := w7.DevCenterGoodsSdk.PublishGoodsInfo(devcenter.PublishGoodsInfoReq{
		ConsoleUid: int(formula.ConsoleUid),
		Id:         int(formula.GoodsId),
	})
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}

	c.JsonResponseWithoutError(ctx, gin.H{"audit_status": goodsInfo.AuditStatus})
}

func (c FormulaGoods) PublishGoods(ctx *gin.Context) {
	type ParamsValidate struct {
		Identifie string `form:"identifie" binding:"required"`
		Version   string `form:"version" binding:"required"`
		Logo      string `form:"logo" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	depotLogin := c.getDepot()
	formula, err := depotLogin.GetFormula(params.Identifie, params.Version, logic2.User{}.GetUser(ctx))
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}
	if formula.InstallServiceFee <= 0 {
		c.JsonResponseWithError(ctx, errors.New("请先设置安装费用"), 500)
		return
	}

	consoleUid := logic2.User{}.GetConsoleUid(ctx)
	if formula.ConsoleUid != 0 && formula.ConsoleUid != consoleUid {
		c.JsonResponseWithError(ctx, errors.New("非法操作"), 500)
		return
	}

	publishReq := devcenter.PublishGoodsReq{
		ConsoleUid: int(consoleUid),
		Logo:       params.Logo,
		WindowLogo: params.Logo,
		GoodsImgs: []map[string]string{
			map[string]string{
				"url": params.Logo,
			},
		},
	}
	err = logic.FormulaGoods{}.PublishGoods(formula, publishReq)
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}

	c.JsonSuccessResponse(ctx)
}
