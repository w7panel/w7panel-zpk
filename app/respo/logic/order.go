package logic

import (
	"log/slog"
	"strconv"

	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/service/w7"
	zpk_market "github.com/w7panel/w7panel-zpk/common/service/w7/zpk-market"
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

func (l Order) CheckFormulaCanInstallOrUpgrade(formula Formula, consoleUid int32, orderSn string, isUpgrade, reinstall bool, domain, appIdentify string) zpk_market.FormulaInstallCheckResult {
	slog.Info("check formula install permission",
		"formula_identify", formula.Name,
		"formula_version", formula.Version,
		"console_uid", consoleUid,
		"order_sn", orderSn,
		"is_upgrade", isUpgrade,
		"reinstall", reinstall,
		"domain", domain,
		"app_identify", appIdentify,
	)
	if formula.GoodsId <= 0 || formula.ConsoleUid == consoleUid {
		return zpk_market.FormulaInstallCheckResult{CanInstallOrUpgrade: true}
	}

	ret, err := w7.ZpkMarketSdk.CheckFormulaCanInstallOrUpgrade(formula.GoodsId, consoleUid, orderSn, isUpgrade, reinstall, domain, appIdentify)
	slog.Info("formula install permission checked",
		"install_result", ret,
		"err", err,
	)
	if err != nil {
		return zpk_market.FormulaInstallCheckResult{}
	}
	return ret
}

func (l Order) GetFormulaCanUpgradeVersion(formula Formula, consoleUid int32, orderSn string) (zpk_market.FormulaUpgradeVersionResult, error) {
	slog.Info("check formula upgrade permission",
		"formula_identify", formula.Name,
		"console_uid", consoleUid,
		"order_sn", orderSn,
	)

	if formula.GoodsId <= 0 || formula.ConsoleUid == consoleUid {
		return zpk_market.FormulaUpgradeVersionResult{
			FormulaIdentify: formula.Name,
			OrderSn:         orderSn,
		}, nil
	}

	response, err := w7.ZpkMarketSdk.GetFormulaCanUpgradeVersion(formula.GoodsId, consoleUid, orderSn)
	slog.Info("formula upgrade permission checked",
		"upgrade_result", response,
		"err", err,
	)
	if err != nil {
		return zpk_market.FormulaUpgradeVersionResult{
			FormulaIdentify: formula.Name,
			OrderSn:         orderSn,
		}, err
	}
	if response.FormulaIdentify == "" {
		response.FormulaIdentify = formula.Name
	}

	_, err = strconv.Atoi(response.Version)
	if err == nil {
		formulaID := formula.ID
		if response.FormulaIdentify != formula.Name {
			targetFormula, _ := dao.Q.Formula.Where(dao.Q.Formula.Name.Eq(response.FormulaIdentify)).First()
			if targetFormula != nil {
				formulaID = targetFormula.ID
			}
		}
		realVersion, err := dao.Q.Version.
			Where(dao.Q.Version.FormulaID.Eq(formulaID)).
			Where(dao.Q.Version.Name.Like(response.Version + ".%")).
			Where(dao.Q.Version.PublishStatus.In(FormulaPublishStatusSuccess, 0)).
			Order(dao.Q.Version.ID.Desc()).First()
		if err != nil {
			return response, err
		}
		response.Version = realVersion.Name
		response.FormulaExpire = false
	}

	return response, nil
}
