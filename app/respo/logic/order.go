package logic

import (
	"log/slog"
	"strconv"

	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/service/w7"
)

type Order struct {
}

func (l Order) DiscardUsedOrder(ticketInfo TicketInfo) error {
	if ticketInfo.ConsoleUid <= 0 || ticketInfo.OrderSn == "" {
		return nil
	}
	return w7.ZpkMarketSdk.DiscardUsedOrder(ticketInfo.ConsoleUid, ticketInfo.OrderSn)
}

func (l Order) UseOrder(ticketInfo TicketInfo, panelDeviceSN, panelURL string) error {
	if ticketInfo.ConsoleUid <= 0 || ticketInfo.OrderSn == "" {
		return nil
	}
	return w7.ZpkMarketSdk.UseOrder(ticketInfo.ConsoleUid, ticketInfo.OrderSn, ticketInfo.FormulaVersion, ticketInfo.IsUpgrade, ticketInfo.Reinstall, panelDeviceSN, panelURL, ticketInfo.AppIdentify, ticketInfo.Domain)
}

func (l Order) CheckFormulaCanInstallOrUpgrade(formula Formula, consoleUid int32, orderSn string, isUpgrade, reinstall bool, domain, appIdentify string) (bool, string, string, string, string, string, string, string) {
	slog.Info("check order permission can install", "formula_name", formula.Name, "version", formula.Version, "consoleuid", consoleUid, "orderSn", orderSn, "isUpgrade", isUpgrade, "reinstall", reinstall, "domain", domain, "appIdentify", appIdentify)
	if formula.GoodsId <= 0 || formula.ConsoleUid == consoleUid {
		return true, formula.Name, orderSn, "", "", "", "", ""
	}

	ok, formulaIdentify, actualOrderSn, panelURL, panelDeviceSN, conflictReason, conflictDomain, conflictAppIdentify, err := w7.ZpkMarketSdk.CheckFormulaCanInstallOrUpgrade(formula.GoodsId, consoleUid, orderSn, isUpgrade, reinstall, domain, appIdentify)
	if err != nil {
		slog.Error("check formula can install or upgrade failed", "formula", formula.Name, "orderSn", orderSn, "err", err)
		return false, formula.Name, orderSn, "", "", "", "", ""
	}
	if formulaIdentify == "" {
		formulaIdentify = formula.Name
	}
	return ok, formulaIdentify, actualOrderSn, panelURL, panelDeviceSN, conflictReason, conflictDomain, conflictAppIdentify
}

func (l Order) GetFormulaCanUpgradeVersion(formula Formula, consoleUid int32, orderSn string) (string, bool, string, string, error) {
	slog.Info("check order permission can upgrade", "formula", formula, "consoleuid", consoleUid, "orderSn", orderSn)

	if formula.GoodsId <= 0 || formula.ConsoleUid == consoleUid {
		return "", false, formula.Name, orderSn, nil
	}

	version, ok, formulaIdentify, actualOrderSn, err := w7.ZpkMarketSdk.GetFormulaCanUpgradeVersion(formula.GoodsId, consoleUid, orderSn)
	if err != nil {
		return "", false, formula.Name, orderSn, err
	}
	if formulaIdentify == "" {
		formulaIdentify = formula.Name
	}

	_, err = strconv.Atoi(version)
	if err == nil {
		formulaID := formula.ID
		if formulaIdentify != formula.Name {
			targetFormula, _ := dao.Q.Formula.Where(dao.Q.Formula.Name.Eq(formulaIdentify)).First()
			if targetFormula != nil {
				formulaID = targetFormula.ID
			}
		}
		realVersion, err := dao.Q.Version.
			Where(dao.Q.Version.FormulaID.Eq(formulaID)).
			Where(dao.Q.Version.Name.Like(version + ".%")).
			Where(dao.Q.Version.PublishStatus.In(FormulaPublishStatusSuccess, 0)).
			Order(dao.Q.Version.ID.Desc()).First()
		if err != nil {
			return "", false, formulaIdentify, actualOrderSn, err
		}
		return realVersion.Name, false, formulaIdentify, actualOrderSn, nil
	}

	return version, ok, formulaIdentify, actualOrderSn, nil
}
