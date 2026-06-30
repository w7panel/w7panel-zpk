package logic

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/w7panel/w7panel-zpk/common/accessor"
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

func (l FormulaGoods) uploadImage(formula *Formula) (string, error) {
	iconFile, err := GetLocalClient().GetFile(formula.GetIconRelativePath())
	iconPath := ""
	if err == nil {
		iconPath = iconFile.Name()
	} else {
		iconPath = facade.GetConfig().GetString("setting.depot.default_icon_path")
	}

	imgFile, err := os.Open(iconPath)
	if err != nil {
		return "", err
	}
	host := "console.w7.cc"
	ticket, err1 := w7.W7CloudAttach.GetJsTicketByHost(host)
	if err1 != nil && err1.IsError() {
		return "", err1
	}

	img, err := w7.W7CloudAttach.UploadImg(ticket, imgFile, imgFile.Name())
	if err != nil {
		return "", err
	}

	return img.Attach.Path, nil
}

func (l FormulaGoods) PublishGoods(formula *Formula, publishGoodsReq devcenter.PublishGoodsReq) error {
	iconPath, err := l.uploadImage(formula)
	if err != nil {
		return err
	}
	publishGoodsReq.Logo = iconPath
	publishGoodsReq.WindowLogo = iconPath
	publishGoodsReq.GoodsImgs = []map[string]string{
		map[string]string{
			"url": iconPath,
		},
	}

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

	enableServicePackageFee := formula.Setting != nil && formula.Setting.EnableServicePackageFee
	servicePackages := make([]devcenter.NotAppServicePackage, 0)
	if enableServicePackageFee && formula.ServicePackages != nil && formula.ServicePackages.List != nil {
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

	var crossUpgradeFormulaContent []byte
	supportCrossUpgrade := 0
	if formula.Setting != nil && formula.Setting.SupportCrossUpgrade {
		supportCrossUpgrade = 1
		crossUpgradeFormulas := make([]accessor.CrossUpgradeFormula, 0)
		if formula.CrossUpgradeFormulas != nil && formula.CrossUpgradeFormulas.List != nil {
			crossUpgradeFormulas = formula.CrossUpgradeFormulas.List
		}
		crossUpgradeFormulaContent, err = json.Marshal(crossUpgradeFormulas)
		if err != nil {
			return err
		}
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
		"respo_identify":         formula.Name,
		"respo_latest_version":   formula.Version,
		"service_packages":       string(servicePackagesContent),
		"version_prices":         string(versionPricesContent),
		"support_cross_upgrade":  supportCrossUpgrade,
		"cross_upgrade_formulas": string(crossUpgradeFormulaContent),
	}
	publishGoodsReq.RespoUrl = fmt.Sprintf("https://%s/zpk/respo/info/%s", facade.GetConfig().GetString("setting.depot.external_domain"), formula.Name)

	goods, err := w7.DevCenterGoodsSdk.PublishGoods(publishGoodsReq)
	if err != nil {
		return err
	}

	marketBaseUrl := facade.GetConfig().GetString("setting.depot_market.base_url")
	err = w7.IpGoodsSdk.SetOrderSetting(ip.SetGoodsSettingReq{
		GoodsId:         goods.Id,
		Appid:           facade.GetConfig().GetString("setting.depot_market.appid"),
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
