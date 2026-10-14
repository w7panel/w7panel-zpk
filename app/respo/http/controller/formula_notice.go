package controller

import (
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	zpk_market "github.com/w7panel/w7panel-zpk/common/service/w7/zpk-market"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
)

func writeFormulaAccessError(ctx *gin.Context, goodsID int32, orderSN string, err error) bool {
	var accessErr *zpk_market.InstallAccessError
	if !errors.As(err, &accessErr) {
		return false
	}
	writeFormulaAccessNotice(ctx, goodsID, zpk_market.FormulaInstallCheckResult{OrderSn: orderSN}, accessErr.Reason, accessErr.Message)
	return true
}

func formulaAccessMarketURL(frontendURL, path string) string {
	target, err := url.Parse(strings.TrimSpace(frontendURL))
	if err != nil || (target.Scheme != "https" && target.Scheme != "http") || target.Hostname() == "" || target.User != nil {
		return ""
	}
	// The market uses createWebHashHistory, including when hosted under a prefix.
	target.Path = strings.TrimRight(target.Path, "/") + "/"
	target.RawPath, target.RawQuery, target.RawFragment = "", "", ""
	target.ForceQuery = false
	target.Fragment = path
	return target.String()
}

func formulaAccessNoticePayload(frontendURL string, goodsID int32, result zpk_market.FormulaInstallCheckResult, reason, message string) gin.H {
	title, action := "请先购买应用", "去购买"
	path := fmt.Sprintf("/site-detail/%d", goodsID)
	switch reason {
	case zpk_market.EntitlementStatusTrialExpired:
		title, action = "试用已到期", "去购买或续费"
	case zpk_market.AccessReasonVIPRequired:
		title, action, path = "需要开通 VIP", "去购买 VIP", "/vip"
	case zpk_market.AccessReasonVIPLevel:
		title, action, path = "需要升级 VIP", "去升级 VIP", "/vip"
	case zpk_market.AccessReasonVIPExpired:
		title, action, path = "VIP 已过期", "去续费 VIP", "/vip"
	case zpk_market.InstallConflictRefundPending:
		title, action, path = "订单退款处理中", "查看此订单", "/user-center/orders"
		if orderSN := strings.TrimSpace(result.OrderSn); orderSN != "" {
			path += "?" + url.Values{"order_sn": {orderSN}}.Encode()
		}
	}
	marketURL := formulaAccessMarketURL(frontendURL, path)
	content := `<div style="padding:16px"><p style="margin:0 0 24px;line-height:1.8">` + html.EscapeString(message) + `</p>`
	if reason == zpk_market.InstallConflictRefundPending {
		content += formulaNoticeField("订单号", result.OrderSn)
	}
	if marketURL != "" {
		content += `<a style="display:inline-block;padding:10px 20px;background-color:#165dff;color:#fff;text-decoration:none;border-radius:4px" href="` + html.EscapeString(marketURL) + `" target="_blank" rel="noopener noreferrer">` + html.EscapeString(action) + `</a>`
	} else {
		content += `<p>请前往制品市场查看授权或购买、续费。</p>`
	}
	content += `</div>`
	return gin.H{
		"code": "ZPK_HTML_NOTICE", "error": message,
		"data": gin.H{
			"title": title, "html": content, "reason": reason,
			"goods_id": goodsID, "order_sn": result.OrderSn,
			"trial_expire_at": result.TrialExpireAt,
			"restriction":     "application_operation", "market_url": marketURL,
		},
	}
}

func writeFormulaAccessNotice(ctx *gin.Context, goodsID int32, result zpk_market.FormulaInstallCheckResult, reason, message string) {
	frontendURL := facade.GetConfig().GetString("setting.depot_market.frontend_url")
	if ctx.Query("notice_format") != "html" {
		if reason == zpk_market.EntitlementStatusTrialExpired {
			// Preserve the legacy code, message and metadata for older panels.
			marketURL := strings.TrimRight(frontendURL, "/")
			if marketURL != "" {
				marketURL = fmt.Sprintf("%s/site-detail/%d", marketURL, goodsID)
			}
			ctx.JSON(http.StatusForbidden, gin.H{
				"code": "ZPK_TRIAL_EXPIRED", "error": "免费试用已到期",
				"data": gin.H{
					"goods_id": goodsID, "order_sn": result.OrderSn,
					"trial_expire_at": result.TrialExpireAt,
					"restriction":     "application_operation", "market_url": marketURL,
				},
			})
			return
		}
		controller := Formula{}
		controller.JsonResponseWithError(ctx, errors.New(message), http.StatusInternalServerError)
		return
	}
	ctx.JSON(http.StatusForbidden, formulaAccessNoticePayload(
		frontendURL, goodsID, result, reason, message,
	))
}

func formulaOriginalPanelURL(panelURL, appIdentify string) string {
	target, err := url.Parse(strings.TrimSpace(panelURL))
	if err != nil || (target.Scheme != "https" && target.Scheme != "http") || target.Hostname() == "" || target.User != nil {
		return ""
	}
	path := strings.TrimRight(target.Path, "/")
	if !strings.HasSuffix(path, "/app/apps") {
		path += "/app/apps"
	}
	target.Path, target.RawPath, target.Fragment, target.RawFragment = path, "", "", ""
	query := target.Query()
	if appIdentify != "" {
		query.Set("uninstallApp", appIdentify)
	}
	target.RawQuery = query.Encode()
	return target.String()
}

type formulaConflictNoticeContext struct {
	AppTitle string
	OrderSN  string
	Domain   string
}

func formulaNoticeField(label, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = "未提供"
	}
	return `<p>` + html.EscapeString(label) + `：<strong>` + html.EscapeString(value) + `</strong></p>`
}

func writeFormulaInstallConflict(ctx *gin.Context, result zpk_market.FormulaInstallCheckResult, allowReinstall bool, info formulaConflictNoticeContext) {
	data := gin.H{
		"conflict_reason": result.ConflictReason,
		"domain":          result.ConflictDomain, "panel_url": result.PanelURL,
		"panel_device_sn": result.PanelDeviceSN, "app_identify": result.ConflictAppIdentify,
	}
	if ctx.Query("notice_format") == "html" {
		data["title"] = "制品安装绑定冲突"
		orderSN := result.OrderSn
		if strings.TrimSpace(orderSN) == "" {
			orderSN = info.OrderSN
		}
		metadata := formulaNoticeField("应用", info.AppTitle) + formulaNoticeField("订单号", orderSN)
		if result.ConflictReason == zpk_market.InstallConflictDomainMismatch {
			data["html"] = `<p>当前域名与订单原绑定域名不一致，请使用原绑定域名安装。</p>` + metadata + formulaNoticeField("当前安装域名", info.Domain) + formulaNoticeField("原绑定域名", result.ConflictDomain)
		} else {
			data["html"] = `<p>该订单已有应用安装记录，请先在原面板卸载原应用后重试。</p>` + metadata + formulaNoticeField("已绑定应用标识", result.ConflictAppIdentify) + formulaNoticeField("原面板地址", result.PanelURL)
			if target := formulaOriginalPanelURL(result.PanelURL, result.ConflictAppIdentify); target != "" {
				data["html"] = data["html"].(string) + `<p><a style="display:inline-block;padding:8px 16px;background-color:#165dff;color:#fff;text-decoration:none;border-radius:4px" href="` + html.EscapeString(target) + `" target="_blank" rel="noopener noreferrer">前往原面板卸载</a></p>`
			}
			if allowReinstall {
				data["html"] = data["html"].(string) + `<p style="color:#f53f3f">强制清除并安装会覆盖旧应用的安装记录，可能导致老应用状态丢失、无法继续升级，请谨慎操作。</p>`
				data["html"] = data["html"].(string) + `<button data-panel-action="reinstall" style="padding:8px 16px;background-color:#f53f3f;color:#fff;border:0;border-radius:4px">强制清除并安装</button>`
			}
		}
	}
	// Keep the original conflict protocol so older panels can still handle it.
	ctx.JSON(http.StatusConflict, gin.H{"code": http.StatusConflict, "error": "制品安装绑定冲突", "data": data})
}
