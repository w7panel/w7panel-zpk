package logic

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	"github.com/w7panel/w7panel-zpk/common/service/w7"
	"github.com/w7panel/w7panel-zpk/common/service/w7/devcenter"
	"github.com/w7panel/w7panel-zpk/common/service/w7/ip"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
)

const FormulaVersionElse = 9999
const ServicePackageEnable = 2

type FormulaGoods struct {
}

func (l FormulaGoods) PublishGoods(formula *Formula, publishGoodsReq devcenter.PublishGoodsReq) error {
	tags, err := dao.Q.TagFormula.Preload(dao.Q.TagFormula.Tag).Where(dao.Q.TagFormula.FormulaID.Eq(formula.ID)).Find()
	if err != nil {
		return err
	}
	if len(tags) == 0 {
		return errors.New("请先设置制品标签")
	}
	if len(tags) > 5 {
		tags = tags[:5]
	}
	goodsTags, err := w7.DevCenterGoodsSdk.GoodsLabels(devcenter.GoodsLabelsReq{
		Page:     1,
		PageSize: 10000,
	})
	if err != nil {
		return err
	}
	goodsTagNameIdMap := make(map[string]int)
	for _, item := range goodsTags.List {
		goodsTagNameIdMap[item.Title] = item.Id
	}
	for _, item := range tags {
		tagName := strings.Trim(item.Tag.Name, " ")
		if _, exists := goodsTagNameIdMap[tagName]; !exists {
			label, err := w7.DevCenterGoodsSdk.AddLabel(devcenter.AddLabelReq{
				Title: strings.Trim(tagName, " "),
			})
			if err != nil {
				return err
			}
			goodsTagNameIdMap[tagName] = label.Id
		}
		publishGoodsReq.LabelIds = append(publishGoodsReq.LabelIds, goodsTagNameIdMap[tagName])
	}

	publishGoodsReq.Title = formula.Manifest.Application.Name
	publishGoodsReq.Summary = formula.Manifest.Application.Description
	if publishGoodsReq.Summary == "" {
		publishGoodsReq.Summary = publishGoodsReq.Title
	}
	publishGoodsReq.Description = publishGoodsReq.Summary

	servicePackages := make([]devcenter.NotAppServicePackage, 0)
	if formula.ServicePackages != nil && formula.ServicePackages.List != nil {
		for _, item := range formula.ServicePackages.List {
			if item.IsEnable == ServicePackageEnable {
				servicePackages = append(servicePackages, item)
			}
		}
	}
	servicePackagesContent, err := json.Marshal(servicePackages)
	if err != nil {
		return err
	}
	versionPrices := make([]devcenter.NotAppBranchVersionPriceInfo, 0)
	if formula.VersionPrices != nil && formula.VersionPrices.List != nil {
		versionPrices = formula.VersionPrices.List
	}
	versionPricesContent, err := json.Marshal(versionPrices)
	if err != nil {
		return err
	}

	if formula.GoodsId > 0 {
		info, err := w7.DevCenterGoodsSdk.PublishGoodsInfo(devcenter.PublishGoodsInfoReq{
			ConsoleUid: publishGoodsReq.ConsoleUid,
			Id:         int(formula.GoodsId),
		})
		if err != nil {
			return err
		}
		publishGoodsReq.Id = int(formula.GoodsId)
		publishGoodsReq.Products = info.Products
	} else {
		publishGoodsReq.OnShelf = 1
		publishGoodsReq.AuditStatus = 2
	}
	publishGoodsReq.UnitPrice = formula.InstallServiceFee
	publishGoodsReq.GoodsType = devcenter.W7ZpkGoodsCategoryId
	publishGoodsReq.Enable = 1
	publishGoodsReq.Water = 2
	publishGoodsReq.Extra = map[string]interface{}{
		"respo_identify":       formula.Name,
		"respo_latest_version": formula.Version,
		"service_packages":     string(servicePackagesContent),
		"version_prices":       string(versionPricesContent),
		"product_type":         formula.ProductType,
		"is_free_upgrade":      formula.IsFreeUpgrade,
	}
	publishGoodsReq.RespoUrl = fmt.Sprintf("https://%s/zpk/respo/info/%s", facade.GetConfig().GetString("setting.depot.external_domain"), formula.Name)

	goods, err := w7.DevCenterGoodsSdk.PublishGoods(publishGoodsReq)
	if err != nil {
		return err
	}

	marketBaseUrl := facade.GetConfig().GetString("setting.depot_market.base_url")
	err = w7.IpGoodsSdk.SetOrderSetting(ip.SetGoodsSettingReq{
		GoodsId:         goods.Id,
		PayNotifyUrl:    fmt.Sprintf("%s/%s", marketBaseUrl, "zpk-market/order/pay-notify"),
		RefundNotifyUrl: fmt.Sprintf("%s/%s", marketBaseUrl, "zpk-market/order/refund-notify"),
	})
	if err != nil {
		return err
	}

	goodsInfo, err := w7.DevCenterGoodsSdk.PublishGoodsInfo(devcenter.PublishGoodsInfoReq{
		ConsoleUid: publishGoodsReq.ConsoleUid,
		Id:         goods.Id,
	})
	if err != nil {
		return err
	}
	goodsProductId := 0
	if goodsInfo.ProductsInfo != nil && len(goodsInfo.ProductsInfo) > 0 {
		goodsProductId = goodsInfo.ProductsInfo[0].Id
	}

	_, err = dao.Formula.Where(dao.Formula.ID.Eq(formula.ID)).Updates(entity.Formula{
		GoodsID:        int32(goods.Id),
		GoodsProductID: int32(goodsProductId),
		RemoteUID:      int32(publishGoodsReq.ConsoleUid),
	})
	return err
}
