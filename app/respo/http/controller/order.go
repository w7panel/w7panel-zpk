package controller

import (
	"errors"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/app/respo/logic"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/w7panel/w7panel-zpk/common/service/w7/devcenter"
)

type Order struct {
	Abstract
}

func (c Order) List(ctx *gin.Context) {
	type ParamsValidate struct {
		Limit           int    `form:"limit,default=10" json:"limit" binding:"omitempty"`
		Page            int    `form:"page,default=1" json:"page" binding:"omitempty,gt=0"`
		Keyword         string `form:"keyword" json:"keyword"`
		FormulaIdentify string `form:"formula_identify" json:"formula_identify"`
		EnableStatus    int    `form:"enable_status" json:"enable_status"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	if params.Page < 1 {
		params.Page = 1
	}
	if params.Limit < 1 {
		params.Limit = 10
	}

	type ResultNode struct {
		ID                    int32                                    `json:"id"`
		OrderSn               string                                   `json:"order_sn"`
		OrderType             int32                                    `json:"order_type"`
		FormulaID             int32                                    `json:"formula_id"`
		GoodsID               int32                                    `json:"goods_id"`
		FormulaName           string                                   `json:"formula_name"`
		FormulaIdentifie      string                                   `json:"formula_identifie"`
		FormulaVersion        string                                   `json:"formula_version"`
		ProductType           int32                                    `json:"product_type"`
		ServiceExpireTime     string                                   `json:"service_expire_time"`
		PayStatus             int32                                    `json:"pay_status"`
		CreatedAt             string                                   `json:"created_at"`
		UsedTime              string                                   `json:"used_time"`
		ServicePackages       []devcenter.NotAppServicePackage         `json:"service_packages"`
		VersionPricesPackages []devcenter.NotAppBranchVersionPriceInfo `json:"version_prices"`
		IsFreeUpgrade         int32                                    `json:"is_free_upgrade"`
		FormulaLatestVersion  string                                   `json:"formula_latest_version"`
	}

	consoleUID := logic2.User{}.GetConsoleUid(ctx)
	query := dao.Q.Order.
		Where(dao.Q.Order.RemoteBuyerUID.Eq(consoleUID)).
		Where(dao.Order.OrderType.Eq(logic.OrderTypeBase)).
		Where(dao.Order.PayStatus.Eq(logic.OrderPayStatusSuccess)).
		Order(dao.Q.Order.ID.Desc())
	if params.Keyword != "" {
		formulaIDs := make([]int32, 0)
		_ = dao.Q.Formula.
			Select(dao.Q.Formula.ID).
			Where(dao.Q.Formula.Title.Like("%" + params.Keyword + "%")).
			Or(dao.Q.Formula.Name.Like("%" + params.Keyword + "%")).
			Scan(&formulaIDs)
		if len(formulaIDs) == 0 {
			c.JsonResponseWithoutError(ctx, gin.H{
				"total": 0,
				"limit": params.Limit,
				"page":  params.Page,
				"list":  []ResultNode{},
			})
			return
		}
		query = query.Where(dao.Q.Order.FormulaID.In(formulaIDs...))
	}
	if params.FormulaIdentify != "" {
		tmpFormula, _ := dao.Q.Formula.Where(dao.Q.Formula.Name.Eq(params.FormulaIdentify)).First()
		if tmpFormula != nil {
			query = query.Where(dao.Q.Order.FormulaID.Eq(tmpFormula.ID))
		}
	}
	if params.EnableStatus == 1 {
		query = query.Where(dao.Q.Order.UsedTime.IsNull())
	}
	if params.EnableStatus == 2 {
		query = query.Where(dao.Q.Order.UsedTime.IsNotNull())
	}

	orderList, total, err := query.FindByPage((params.Page-1)*params.Limit, params.Limit)
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}

	formulaMap := map[int32]*entity.Formula{}
	serviceExpireTimeMap := map[int32]time.Time{}
	if len(orderList) > 0 {
		formulaIDs := make([]int32, 0, len(orderList))
		formulaIDSet := make(map[int32]struct{})
		orderIDs := make([]int32, 0, len(orderList))
		for _, item := range orderList {
			orderIDs = append(orderIDs, item.ID)
			if _, exists := formulaIDSet[item.FormulaID]; exists {
				continue
			}
			formulaIDSet[item.FormulaID] = struct{}{}
			formulaIDs = append(formulaIDs, item.FormulaID)
		}
		formulas, _ := dao.Q.Formula.Where(dao.Q.Formula.ID.In(formulaIDs...)).Find()
		for _, item := range formulas {
			formulaMap[item.ID] = item
		}

		type serviceOrderGroupRow struct {
			ParentID          int32      `json:"parent_id"`
			ServiceExpireTime *time.Time `json:"service_expire_time"`
		}
		serviceOrders := make([]serviceOrderGroupRow, 0)
		_ = dao.Q.Order.
			Select(
				dao.Q.Order.ParentID,
				dao.Q.Order.ServiceExpireTimeV2.Max().As("service_expire_time"),
			).
			Where(dao.Q.Order.ParentID.In(orderIDs...)).
			Where(dao.Q.Order.OrderType.Eq(logic.OrderTypePeriodService)).
			Where(dao.Q.Order.PayStatus.Eq(logic.OrderPayStatusSuccess)).
			Group(dao.Q.Order.ParentID).
			Scan(&serviceOrders)
		for _, item := range serviceOrders {
			if item.ServiceExpireTime == nil {
				continue
			}
			serviceExpireTimeMap[item.ParentID] = *item.ServiceExpireTime
		}
	}

	result := make([]ResultNode, 0, len(orderList))
	for _, item := range orderList {
		formulaName := ""
		formulaIdentifie := ""
		goodsID := int32(0)
		servicePackages := make([]devcenter.NotAppServicePackage, 0)
		versionPackages := make([]devcenter.NotAppBranchVersionPriceInfo, 0)
		isFreeUpgrade := 0
		if formulaMap[item.FormulaID] != nil {
			formulaName = formulaMap[item.FormulaID].Title
			formulaIdentifie = formulaMap[item.FormulaID].Name
			goodsID = formulaMap[item.FormulaID].GoodsID
			if formulaMap[item.FormulaID].ServicePackages != nil {
				for _, sitem := range formulaMap[item.FormulaID].ServicePackages.List {
					if sitem.IsEnable == logic.ServicePackageEnable {
						servicePackages = append(servicePackages, sitem)
					}
				}
			}
			if formulaMap[item.FormulaID].VersionPrices != nil && formulaMap[item.FormulaID].VersionPrices.List != nil {
				versionPackages = formulaMap[item.FormulaID].VersionPrices.List
			}
			isFreeUpgrade = int(formulaMap[item.FormulaID].IsFreeUpgrade)
			if isFreeUpgrade < 0 {
				isFreeUpgrade = 0
			}
		}
		serviceExpireTime, _ := serviceExpireTimeMap[item.ID]
		result = append(result, ResultNode{
			ID:                item.ID,
			OrderSn:           item.OrderSn,
			OrderType:         item.OrderType,
			FormulaID:         item.FormulaID,
			GoodsID:           goodsID,
			FormulaName:       formulaName,
			FormulaIdentifie:  formulaIdentifie,
			FormulaVersion:    item.FormulaVersion,
			ProductType:       item.ProductType,
			ServiceExpireTime: serviceExpireTime.Format("2006-01-02 15:04:05"),
			PayStatus:         item.PayStatus,
			CreatedAt: func() string {
				if item.PaidAt == nil {
					return ""
				}
				return item.PaidAt.Format("2006-01-02 15:04:05")
			}(),
			UsedTime: func() string {
				if item.UsedTime == nil {
					return ""
				}
				return item.UsedTime.Format("2006-01-02 15:04:05")
			}(),
			ServicePackages:       servicePackages,
			VersionPricesPackages: versionPackages,
			IsFreeUpgrade:         int32(isFreeUpgrade),
		})
	}

	c.JsonResponseWithoutError(ctx, gin.H{
		"total": total,
		"limit": params.Limit,
		"page":  params.Page,
		"list":  result,
	})
}

func (c Order) Query(ctx *gin.Context) {
	type ParamsValidate struct {
		OrderSn string `form:"order_sn" json:"order_sn" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	order, _ := dao.Q.Order.Where(dao.Q.Order.OrderSn.Eq(params.OrderSn)).First()
	if order == nil {
		c.JsonResponseWithError(ctx, errors.New("order_sn 不存在"), 500)
		return
	}

	c.JsonResponseWithoutError(ctx, order)
}

func (c Order) Pay(ctx *gin.Context) {
	type ParamsValidate struct {
		Identifie        string `form:"identifie" json:"identifie" binding:"required"`
		ServicePackageId int    `form:"service_package_id" json:"service_package_id"`
		VersionUpgradeId int    `form:"version_upgrade_id" json:"version_upgrade_id"`
		OrderSn          string `form:"order_sn" json:"order_sn"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	formula, err := c.getDepot().GetFormula(params.Identifie, "", nil)
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}

	payInfo, order, err := logic.Order{}.Pay(logic2.User{}.GetConsoleUid(ctx), formula, params.OrderSn, params.ServicePackageId, params.VersionUpgradeId)
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}

	c.JsonResponseWithoutError(ctx, map[string]string{
		"ticket":   payInfo.Ticket,
		"qr_code":  payInfo.QrcodeUrl,
		"order_sn": order.OrderSn,
	})
}

func (c Order) PayNotify(ctx *gin.Context) {
	params := logic.OrderPayNotify{}
	if !c.Validate(ctx, &params) {
		return
	}

	err := logic.Order{}.OrderPayNotify(params)
	slog.Info("pay notify", "params", params, "err", err)
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}
	c.JsonSuccessResponse(ctx)
}

func (c Order) RefundNotify(ctx *gin.Context) {
	params := logic.OrderRefundNotify{}
	if !c.Validate(ctx, &params) {
		return
	}

	err := logic.Order{}.OrderRefundNotify(params)
	slog.Info("pay notify", "params", params, "err", err)
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}
	c.JsonSuccessResponse(ctx)
}
