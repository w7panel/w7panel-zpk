package logic

import (
	"errors"
	"log/slog"
	"strconv"

	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/service/w7"
)

type Order struct {
}

func (l Order) DiscardUsedOrder(ticketInfo TicketInfo) error {
	return w7.ZpkMarketSdk.DiscardUsedOrder(ticketInfo.OrderSn)
}

func (l Order) UseOrder(ticketInfo TicketInfo) error {
	return w7.ZpkMarketSdk.UseOrder(ticketInfo.OrderSn, ticketInfo.FormulaVersion, ticketInfo.IsUpgrade)
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

	ok, _ := w7.ZpkMarketSdk.CheckFormulaCanInstallOrUpgrade(formula.GoodsId, consoleUid, orderSn, isUpgrade)
	return ok
}

func (l Order) GetFormulaCanUpgradeVersion(formula Formula, consoleUid int32, orderSn string) (string, bool, error) {
	slog.Info("check order permission can upgrade", "formula", formula, "consoleuid", consoleUid, "orderSn", orderSn)

	if formula.GoodsId == 0 || formula.ConsoleUid == int64(consoleUid) {
		return "", false, nil
	}

	if formula.GoodsId > 0 && orderSn == "" {
		return "", false, errors.New("order not exists")
	}

	version, ok, err := w7.ZpkMarketSdk.GetFormulaCanUpgradeVersion(formula.GoodsId, consoleUid, orderSn)
	if err != nil {
		return "", false, err
	}

	_, err = strconv.Atoi(version)
	if err == nil {
		realVersion, err := dao.Q.Version.
			Where(dao.Q.Version.FormulaID.Eq(formula.ID)).
			Where(dao.Q.Version.Name.Like(version + ".%")).
			Where(dao.Q.Version.PublishStatus.In(FormulaPublishStatusSuccess, 0)).
			Order(dao.Q.Version.ID.Desc()).First()
		if err != nil {
			return "", false, err
		}
		return realVersion.Name, false, nil
	}

	return version, ok, nil
}
