package logic

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	"github.com/w7panel/w7panel-zpk/common/function"
	"github.com/w7panel/w7panel-zpk/common/service/w7"
	"github.com/w7panel/w7panel-zpk/common/service/w7/devcenter"
	"github.com/w7panel/w7panel-zpk/common/service/w7/ip"
)

const (
	OrderTypeBase          = int32(1)
	OrderTypePeriodService = int32(2)
	OrderTypeUpgrade       = int32(3)
)

const (
	OrderPayStatusIng        = int32(0)
	OrderPayStatusSuccess    = int32(1)
	OrderRefundStatusSuccess = int32(2)
)

type OrderPayNotify struct {
	ConsoleUid   int    `form:"uid" binding:"required"`
	OrderSn      string `form:"ip_order_sn" binding:"required"`
	OrderId      int    `form:"ip_order_id" binding:"required"`
	GoodsId      int    `form:"ip_goods_id" binding:"required"`
	CategoryType int    `form:"category_type"`
}

type OrderRefundNotify struct {
	ConsoleUid   int    `form:"uid" binding:"required"`
	OrderSn      string `form:"ip_order_sn" binding:"required"`
	OrderId      int    `form:"ip_order_id" binding:"required"`
	GoodsId      int    `form:"ip_goods_id" binding:"required"`
	CategoryType int    `form:"category_type"`
}

type Order struct {
}

func generateOrderSn() string {
	return fmt.Sprintf("%s%s", time.Now().Format("20060102150405"), function.GetRandomStringNotContainerNumber(8))
}

func (l Order) Pay(consoleUid int32, formula *Formula, orderSn string, servicePackageId int, versionUpgradeId int) (*ip.CreateOrderResp, *entity.Order, error) {
	if formula.GoodsId == 0 || formula.GoodsProductId == 0 {
		return nil, nil, errors.New("暂不支持购买，请先设置价格并发布")
	}

	orderType := OrderTypeBase
	buyShop := ""
	price := 0.0
	if orderSn == "" {
		price = formula.InstallServiceFee
	}

	existsServiceOrderFirst := false
	if servicePackageId > 0 {
		if formula.ServicePackages == nil || formula.ServicePackages.List == nil {
			return nil, nil, errors.New("invalid servicePackageId")
		}
		exists := false
		for _, item := range formula.ServicePackages.List {
			if item.Id == servicePackageId {
				if item.IsEnable != ServicePackageEnable {
					return nil, nil, errors.New("当前周期服务不可用")
				}
				orderType = OrderTypePeriodService
				if orderSn == "" && item.IsGift {
					orderType = OrderTypeBase
					item.Price = 0
				}
				existsServiceOrderFirst = orderSn == "" && !item.IsGift
				buyShop = strconv.Itoa(item.Month)
				price += float64(item.Price)
				exists = true
				break
			}
		}
		if !exists {
			return nil, nil, errors.New("invalid servicePackageId")
		}
	} else if versionUpgradeId > 0 {
		if formula.VersionPrices == nil || formula.VersionPrices.List == nil {
			return nil, nil, errors.New("invalid versionUpgradeId")
		}
		exists := false
		for _, item := range formula.VersionPrices.List {
			if item.Id == versionUpgradeId {
				orderType = OrderTypeUpgrade
				buyShop = strconv.FormatInt(item.Version, 10)
				exists = true
				price = float64(item.Price)
				break
			}
		}
		if !exists {
			return nil, nil, errors.New("invalid versionUpgradeId")
		}
	}

	if orderType == OrderTypeUpgrade && orderSn == "" {
		return nil, nil, errors.New("order sn 缺失")
	}

	var baseOrder *entity.Order
	if orderSn != "" {
		baseOrder, _ = dao.Q.Order.
			Where(dao.Q.Order.OrderSn.Eq(orderSn)).
			Where(dao.Q.Order.OrderType.Eq(OrderTypeBase)).
			Where(dao.Q.Order.PayStatus.Eq(OrderPayStatusSuccess)).
			Where(dao.Q.Order.FormulaID.Eq(formula.ID)).
			Where(dao.Q.Order.RemoteBuyerUID.Eq(consoleUid)).
			First()
		if baseOrder == nil {
			return nil, nil, errors.New("order_sn 不存在")
		}
	}

	order, err := w7.IpOrderSdk.CreateOrder(ip.CreateOrderReq{
		ConsoleUid: consoleUid,
		ProductId:  int(formula.GoodsProductId),
		Quantity:   1,
		Price:      price,
	})
	if err != nil {
		return nil, nil, err
	}

	localOrderSn := order.OrderSn
	remoteOrderSn := order.OrderSn
	var orderModel *entity.Order
	err = dao.Q.Transaction(func(tx *dao.Query) error {
		parentOId := int32(0)
		if existsServiceOrderFirst {
			baseOrderModel := &entity.Order{
				OrderSn:        localOrderSn,
				OutOrderSn:     remoteOrderSn,
				RemoteBuyerUID: consoleUid,
				OrderType:      OrderTypeBase,
				BuyShop:        "",
				TotalFee:       formula.InstallServiceFee,
				FormulaID:      formula.ID,
				ProductType:    formula.ProductType,
				PayStatus:      0,
			}
			err = tx.Order.Create(baseOrderModel)
			if err != nil {
				return err
			}
			parentOId = baseOrderModel.ID
			localOrderSn = localOrderSn + "s"
			remoteOrderSn = baseOrderModel.OrderSn
			orderModel = baseOrderModel
		}

		if !existsServiceOrderFirst && baseOrder != nil {
			parentOId = baseOrder.ID
		}

		tmpOrderModel := &entity.Order{
			ParentID:       parentOId,
			OrderSn:        localOrderSn,
			OutOrderSn:     remoteOrderSn,
			RemoteBuyerUID: consoleUid,
			OrderType:      orderType,
			BuyShop:        buyShop,
			TotalFee:       price,
			FormulaID:      formula.ID,
			ProductType:    formula.ProductType,
			PayStatus:      0,
		}
		err = tx.Order.Create(tmpOrderModel)
		if err != nil {
			return err
		}
		if orderModel == nil {
			orderModel = tmpOrderModel
		}

		return nil
	})

	return order, orderModel, err
}

func (l Order) OrderPayNotify(req OrderPayNotify) error {
	slog.Info("order pay notify", "req", req)
	existsOrder, _ := dao.Q.Order.
		Where(dao.Q.Order.OrderSn.Eq(req.OrderSn)).
		First()
	if existsOrder == nil && req.CategoryType == devcenter.W7ZpkGoodsCategoryId {
		//新建周期服务费订单
		formula, _ := dao.Q.Formula.Where(dao.Q.Formula.GoodsID.Eq(int32(req.GoodsId))).First()
		if formula == nil {
			return errors.New("商品对应的制品不存在")
		}
		orderModel := &entity.Order{
			OrderSn:        req.OrderSn,
			OutOrderSn:     req.OrderSn,
			RemoteBuyerUID: int32(req.ConsoleUid),
			OrderType:      OrderTypeBase,
			BuyShop:        "",
			TotalFee:       0,
			FormulaID:      formula.ID,
			ProductType:    formula.ProductType,
			PayStatus:      OrderPayStatusIng,
		}
		err := dao.Q.Order.Create(orderModel)
		if err != nil {
			return err
		}
		existsOrder = orderModel
	}
	if existsOrder == nil {
		return errors.New("order not exists")
	}
	if existsOrder.PayStatus != OrderPayStatusIng {
		return nil
	}

	formula, _ := dao.Q.Formula.Where(dao.Q.Formula.ID.Eq(existsOrder.FormulaID)).First()
	if formula == nil {
		return errors.New("invalid formula")
	}

	giftServicePackageMonth := 0
	var giftServicePackageExpireTime *time.Time
	if existsOrder.OrderType == OrderTypeBase && formula.ProductType == FORMULA_PRODUCT_CONSOLE_APP && formula.ServicePackages != nil {
		for _, item := range formula.ServicePackages.List {
			if item.IsEnable == ServicePackageEnable && item.IsGift {
				expireAt := time.Now().AddDate(0, item.Month, 0)
				giftServicePackageExpireTime = &expireAt
				giftServicePackageMonth = item.Month

				break
			}
		}
	}

	var serviceExpireTime *time.Time
	if existsOrder.OrderType == OrderTypePeriodService {
		existsServiceOrder, _ := dao.Q.Order.
			Where(dao.Q.Order.ParentID.Eq(existsOrder.ParentID)).
			Where(dao.Q.Order.OrderType.Eq(OrderTypePeriodService)).
			Where(dao.Q.Order.PayStatus.Eq(OrderPayStatusSuccess)).
			Where(dao.Q.Order.ServiceExpireTimeV2.Gt(time.Now())).
			Order(dao.Q.Order.ID.Desc()).
			First()
		beginTime := time.Now()
		if existsServiceOrder != nil && existsServiceOrder.ServiceExpireTimeV2 != nil {
			beginTime = *existsServiceOrder.ServiceExpireTimeV2
		}
		month, err := strconv.Atoi(existsOrder.BuyShop)
		if err != nil {
			return err
		}
		expireAt := beginTime.AddDate(0, month, 0)
		serviceExpireTime = &expireAt
	}

	var mergeServiceExpireTime *time.Time
	mergeServiceOrder, _ := dao.Q.Order.Where(dao.Q.Order.ParentID.Eq(existsOrder.ID)).Where(dao.Q.Order.OutOrderSn.Eq(existsOrder.OrderSn)).First()
	if mergeServiceOrder != nil {
		month, err := strconv.Atoi(mergeServiceOrder.BuyShop)
		if err != nil {
			return err
		}
		expireAt := time.Now().AddDate(0, month, 0)
		mergeServiceExpireTime = &expireAt
	}

	return dao.Q.Transaction(func(tx *dao.Query) error {
		nowTime := time.Now()
		_, err := tx.Order.Where(tx.Order.ID.Eq(existsOrder.ID)).Updates(entity.Order{
			PayStatus:           OrderPayStatusSuccess,
			ServiceExpireTimeV2: serviceExpireTime,
			PaidAt:              &nowTime,
		})
		if err != nil {
			return err
		}

		if mergeServiceOrder != nil {
			_, err = tx.Order.Where(tx.Order.ID.Eq(mergeServiceOrder.ID)).Updates(entity.Order{
				PayStatus:           OrderPayStatusSuccess,
				ServiceExpireTimeV2: mergeServiceExpireTime,
				PaidAt:              &nowTime,
			})
			if err != nil {
				return err
			}
		}

		if giftServicePackageMonth > 0 && giftServicePackageExpireTime != nil {
			orderModel := &entity.Order{
				ParentID:            existsOrder.ID,
				OrderSn:             "gift-" + existsOrder.OrderSn,
				RemoteBuyerUID:      existsOrder.RemoteBuyerUID,
				OrderType:           OrderTypePeriodService,
				BuyShop:             strconv.Itoa(giftServicePackageMonth),
				TotalFee:            0,
				FormulaID:           formula.ID,
				ProductType:         formula.ProductType,
				PayStatus:           OrderPayStatusSuccess,
				ServiceExpireTimeV2: giftServicePackageExpireTime,
				PaidAt:              &nowTime,
			}
			err = tx.Order.Create(orderModel)
			return err
		}
		return nil
	})
}

func (l Order) OrderRefundNotify(req OrderRefundNotify) error {
	slog.Info("order pay notify", "req", req)
	existsOrder, _ := dao.Q.Order.
		Where(dao.Q.Order.OrderSn.Eq(req.OrderSn)).
		First()
	if existsOrder == nil {
		return errors.New("order not exists")
	}

	if existsOrder.PayStatus == OrderRefundStatusSuccess {
		return nil
	}

	return dao.Q.Transaction(func(tx *dao.Query) error {
		_, err := tx.Order.Where(dao.Q.Order.ID.Eq(existsOrder.ID)).Updates(entity.Order{
			PayStatus: OrderRefundStatusSuccess,
		})
		if err != nil {
			return err
		}

		_, err = tx.Order.Where(tx.Order.ParentID.Eq(existsOrder.ID)).Where(tx.Order.OutOrderSn.Eq(existsOrder.OrderSn)).Updates(entity.Order{
			PayStatus: OrderRefundStatusSuccess,
		})
		if err != nil {
			return err
		}

		if existsOrder.OrderType == OrderTypePeriodService {
			month, err := strconv.Atoi(existsOrder.BuyShop)
			if err != nil {
				return err
			}
			serviceOrders, err := tx.Order.
				Where(tx.Order.ParentID.Eq(existsOrder.ParentID)).
				Where(tx.Order.OrderType.Eq(OrderTypePeriodService)).
				Where(tx.Order.PayStatus.Eq(OrderPayStatusSuccess)).
				Where(tx.Order.ID.Gt(existsOrder.ID)).
				Find()
			if err != nil {
				return err
			}

			for _, item := range serviceOrders {
				if item.ServiceExpireTimeV2 == nil {
					continue
				}
				expireTime := item.ServiceExpireTimeV2.AddDate(0, -month, 0)
				_, err = tx.Order.
					Where(tx.Order.ID.Eq(item.ID)).
					Update(tx.Order.ServiceExpireTimeV2, expireTime)
				if err != nil {
					return err
				}
			}
			return nil
		} else {
			_, err = tx.Order.Where(dao.Q.Order.OrderSn.Eq("gift-" + existsOrder.OrderSn)).Updates(entity.Order{
				PayStatus: OrderRefundStatusSuccess,
			})
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (l Order) DiscardUsedOrder(ticketInfo TicketInfo) error {
	_, err := dao.Q.Order.
		Where(dao.Q.Order.OrderSn.Eq(ticketInfo.OrderSn)).
		Where(dao.Q.Order.OrderType.Eq(OrderTypeBase)).
		Where(dao.Q.Order.PayStatus.Eq(OrderPayStatusSuccess)).
		UpdateSimple(dao.Q.Order.UsedTime.Null())

	return err
}

func (l Order) UseOrder(ticketInfo TicketInfo) error {
	formula, _ := dao.Q.Formula.Where(dao.Q.Formula.ID.Eq(ticketInfo.FormulaId)).First()
	if formula == nil {
		return errors.New("制品不存在")
	}
	if formula.RemoteUID == ticketInfo.ConsoleUid {
		return nil
	}
	if formula.GoodsID == 0 {
		if ticketInfo.ConsoleUid > 0 {
			//添加免费订单
			orderSn := generateOrderSn()
			usedAt := time.Now()
			orderModel := &entity.Order{
				OrderSn:        orderSn,
				OutOrderSn:     orderSn,
				RemoteBuyerUID: ticketInfo.ConsoleUid,
				OrderType:      OrderTypeBase,
				BuyShop:        "",
				TotalFee:       0,
				FormulaID:      formula.ID,
				ProductType:    formula.ProductType,
				PayStatus:      OrderPayStatusSuccess,
				UsedTime:       &usedAt,
			}
			err := dao.Q.Order.Create(orderModel)
			if err != nil {
				return err
			}
		}
		return nil
	}
	if ticketInfo.OrderSn == "" {
		return errors.New("order_sn 缺失")
	}

	if ticketInfo.IsUpgrade && ticketInfo.FormulaVersion != "" {
		_, err := dao.Q.Order.
			Where(dao.Q.Order.OrderSn.Eq(ticketInfo.OrderSn)).
			Where(dao.Q.Order.OrderType.Eq(OrderTypeBase)).
			Where(dao.Q.Order.PayStatus.Eq(OrderPayStatusSuccess)).
			Update(dao.Q.Order.FormulaVersion, ticketInfo.FormulaVersion)
		return err
	}

	if !ticketInfo.IsUpgrade {
		existsOrder, _ := dao.Q.Order.
			Where(dao.Q.Order.OrderSn.Eq(ticketInfo.OrderSn)).
			Where(dao.Q.Order.OrderType.Eq(OrderTypeBase)).
			Where(dao.Q.Order.PayStatus.Eq(OrderPayStatusSuccess)).
			Where(dao.Q.Order.UsedTime.IsNull()).
			First()
		if existsOrder == nil {
			return errors.New("order not exists")
		}

		slog.Info("use order ing", "ticket", ticketInfo, "oid", existsOrder.ID)

		curTime := time.Now()
		_, err := dao.Q.Order.Where(dao.Q.Order.ID.Eq(existsOrder.ID)).Updates(entity.Order{
			UsedTime:       &curTime,
			FormulaVersion: ticketInfo.FormulaVersion,
		})
		return err
	}

	return nil
}

func (l Order) CheckFormulaCanInstallOrUpgrade(formula Formula, consoleUid int32, orderSn string, isUpgrade bool) bool {
	slog.Info("check order permission can install", "formula_name", formula.Name, "version", formula.Version, "consoleuid", consoleUid, "orderSn", orderSn, "isUpgrade", isUpgrade)

	if formula.GoodsId == 0 || formula.ConsoleUid == int64(consoleUid) {
		return true
	}

	if formula.GoodsId > 0 && orderSn == "" {
		return false
	}

	if isUpgrade && formula.IsFreeUpgrade == FORMULA_FREE_UPGRADE {
		return true
	}

	existsOrder, _ := dao.Q.Order.
		Where(dao.Q.Order.OrderSn.Eq(orderSn)).
		Where(dao.Q.Order.OrderType.Eq(OrderTypeBase)).
		Where(dao.Q.Order.PayStatus.Eq(OrderPayStatusSuccess)).
		Where(dao.Q.Order.FormulaID.Eq(formula.ID)).
		Where(dao.Q.Order.RemoteBuyerUID.Eq(consoleUid)).
		First()
	if existsOrder == nil {
		return false
	}
	if !isUpgrade {
		return existsOrder.UsedTime == nil
	}

	return true
}

func (l Order) GetFormulaCanUpgradeVersion(formula Formula, consoleUid int32, orderSn string) (string, bool, error) {
	slog.Info("check order permission can upgrade", "formula", formula, "consoleuid", consoleUid, "orderSn", orderSn)

	if formula.GoodsId == 0 || formula.ConsoleUid == int64(consoleUid) {
		return "", false, nil
	}

	if formula.GoodsId > 0 && orderSn == "" {
		return "", false, errors.New("order not exists")
	}

	baseOrder, _ := dao.Q.Order.
		Where(dao.Q.Order.OrderSn.Eq(orderSn)).
		Where(dao.Q.Order.OrderType.Eq(OrderTypeBase)).
		Where(dao.Q.Order.PayStatus.Eq(OrderPayStatusSuccess)).
		Where(dao.Q.Order.FormulaID.Eq(formula.ID)).
		Where(dao.Q.Order.RemoteBuyerUID.Eq(consoleUid)).
		First()
	if baseOrder == nil {
		return "", false, errors.New("order not exists")
	}

	if formula.ProductType == FORMULA_PRODUCT_LOCAL_APP {
		if formula.IsFreeUpgrade == FORMULA_FREE_UPGRADE {
			return "", false, nil
		}

		existsOrder, _ := dao.Q.Order.
			Where(dao.Q.Order.OrderType.Eq(OrderTypeBase)).
			Where(dao.Q.Order.PayStatus.Eq(OrderPayStatusSuccess)).
			Where(dao.Q.Order.FormulaID.Eq(formula.ID)).
			Where(dao.Q.Order.RemoteBuyerUID.Eq(consoleUid)).
			Where(dao.Q.Order.UsedTime.IsNull()).
			First()
		if existsOrder != nil {
			return "", false, nil
		}

		return baseOrder.FormulaVersion, false, nil
	}

	if formula.ProductType == FORMULA_PRODUCT_CONSOLE_APP {
		existsVersionPackage := formula.VersionPrices != nil && formula.VersionPrices.List != nil && len(formula.VersionPrices.List) > 0

		if formula.ServicePackages != nil && formula.ServicePackages.List != nil {
			needCheck := false
			for _, item := range formula.ServicePackages.List {
				if item.IsEnable == ServicePackageEnable {
					needCheck = true
					break
				}
			}
			if needCheck {
				//周期类型的订单只检测过期时间
				existsServiceOrder, _ := dao.Q.Order.
					Where(dao.Q.Order.ParentID.Eq(baseOrder.ID)).
					Where(dao.Q.Order.OrderType.Eq(OrderTypePeriodService)).
					Where(dao.Q.Order.PayStatus.Eq(OrderPayStatusSuccess)).
					Where(dao.Q.Order.FormulaID.Eq(formula.ID)).
					Where(dao.Q.Order.ServiceExpireTimeV2.Gt(time.Now())).
					Order(dao.Q.Order.ID.Desc()).
					First()
				if existsServiceOrder != nil {
					return "", false, nil
				}
				if !existsVersionPackage {
					return baseOrder.FormulaVersion, true, nil
				}
			}
		}

		if existsVersionPackage {
			majorVersion, err := function.ExtractMajorVersion(baseOrder.FormulaVersion)
			if err != nil {
				return baseOrder.FormulaVersion, true, nil
			}
			for _, item := range formula.VersionPrices.List {
				if item.Version > int64(majorVersion) {
					//升级费用的也只检测是否购买了对应版本的
					existsVersionOrder, _ := dao.Q.Order.
						Where(dao.Q.Order.ParentID.Eq(baseOrder.ID)).
						Where(dao.Q.Order.OrderType.Eq(OrderTypeUpgrade)).
						Where(dao.Q.Order.PayStatus.Eq(OrderPayStatusSuccess)).
						Where(dao.Q.Order.BuyShop.Eq(strconv.Itoa(int(item.Version)))).
						Where(dao.Q.Order.FormulaID.Eq(formula.ID)).
						Order(dao.Q.Order.ID.Desc()).
						First()
					if existsVersionOrder != nil {
						if existsVersionOrder.BuyShop == strconv.Itoa(FormulaVersionElse) {
							return "", false, nil
						}

						realVersion, err := dao.Q.Version.
							Where(dao.Q.Version.FormulaID.Eq(formula.ID)).
							Where(dao.Q.Version.Name.Like(existsVersionOrder.BuyShop + ".%")).
							Where(dao.Q.Version.PublishStatus.In(FormulaPublishStatusSuccess, 0)).
							Order(dao.Q.Version.ID.Desc()).First()
						if err != nil {
							return baseOrder.FormulaVersion, true, nil
						}
						return realVersion.Name, false, nil
					}
				}
			}

			return baseOrder.FormulaVersion, true, nil
		}
	}

	return "", false, nil
}
