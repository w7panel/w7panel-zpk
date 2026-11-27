package controller

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	formulalogic "github.com/w7panel/w7panel-zpk/app/respo/logic/formula"
	"github.com/w7panel/w7panel-zpk/app/respo/logic/zpkmarket"
	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/w7panel/w7panel-zpk/common/service/w7"
	zpk_market "github.com/w7panel/w7panel-zpk/common/service/w7/zpk-market"
	v1 "k8s.io/api/core/v1"
)

type formulaInstallDetailParams struct {
	Identifie      string `uri:"id" json:"identifie" binding:"required"`
	Version        string `uri:"version" json:"version"`
	CName          string `uri:"cid" json:"cname"`
	IsUpgrade      int32  `form:"is_upgrade" json:"is_upgrade"`
	CheckUpgrade   int32  `form:"check_upgrade" json:"check_upgrade"`
	CurVersion     string `form:"cur_version" json:"cur_version"`
	Token          string `form:"token" json:"token"`
	OrderSn        string `form:"order_sn" json:"order_sn"`
	ConsoleUid     int32  `form:"console_uid" json:"console_uid"`
	Domain         string `form:"domain" json:"domain"`
	AppIdentify    string `form:"app_identify" json:"app_identify"`
	Reinstall      bool   `form:"reinstall" json:"reinstall"`
	FullManifest   bool   `form:"full_manifest" json:"full_manifest"`
	RuntimeContext bool   `form:"runtime_context" json:"runtime_context"`
}

// authorizeFormulaInstallDetail returns false after writing an error or a runtime-context
// response. Only true allows Info to continue building the installation response.
func (c Formula) authorizeFormulaInstallDetail(ctx *gin.Context, params *formulaInstallDetailParams, formula *formulalogic.Formula, consoleUid int32) (zpk_market.FormulaUpgradeVersionResult, bool) {
	dynamicValues := map[string]interface{}{
		"order_status": "none",
		"license_type": "paid",
		"validate": map[string]interface{}{
			"valid":   true,
			"reason":  "not_required",
			"message": "当前制品无需订单授权",
		},
	}

	user := logic2.User{}.GetUser(ctx)
	isAdminUser := false
	userId := int32(0)
	if user != nil {
		isAdminUser = logic2.User{}.IsAdminUser(user)
		userId = user.ID
	}
	// 安装入口按凭据分流：普通仓库链接验证 pull；市场订单和后台安装 token 走市场验证。
	// params.Token 是市场后台审核 / 测试安装链接的 token，与 Authorization 中的 Registry Bearer token 不同。
	marketRequest := params.OrderSn != "" || params.Token != ""
	if marketRequest && formula.GoodsId <= 0 {
		c.JsonResponseWithError(ctx, errors.New("该制品未发布到制品市场"), http.StatusForbidden)
		return zpk_market.FormulaUpgradeVersionResult{}, false
	}
	if !marketRequest && !checkFormulaPullPermission(ctx, formula.Name) {
		return zpk_market.FormulaUpgradeVersionResult{}, false
	}
	// 源站制品所有者、管理员沿用管理豁免；其他市场访问按以下场景验证。
	if marketRequest && userId != formula.UserId && !isAdminUser {
		if params.Token != "" {
			// 市场管理后台商品列表生成临时安装链接，供管理员审核 / 测试安装，无需购买订单。
			// 由市场检查 token 的应用标识和有效期；此分支不再执行本地仓库 pull 和订单购买校验。
			if err := w7.ZpkMarketSdk.CheckToken(params.Token, formula.Name); err != nil {
				c.JsonResponseWithServerError(ctx, err)
				return zpk_market.FormulaUpgradeVersionResult{}, false
			}
		} else {
			if consoleUid <= 0 {
				c.JsonResponseWithError(ctx, errors.New("制品市场安装需要登录"), http.StatusUnauthorized)
				return zpk_market.FormulaUpgradeVersionResult{}, false
			}
			if params.RuntimeContext {
				// 已安装应用读取订单运行上下文（如授权状态、动态参数）；只返回运行信息，不授予安装权限。
				licenseInfo, err := zpkmarket.GetFormulaLicenseInfo(*formula, consoleUid, params.OrderSn)
				if err != nil {
					c.JsonResponseWithServerError(ctx, err)
					return zpk_market.FormulaUpgradeVersionResult{}, false
				}
				c.JsonResponseWithoutError(ctx, formulaRuntimeContextResponse(formula.Title, licenseInfo))
				return zpk_market.FormulaUpgradeVersionResult{}, false
			}
			return c.resolveFormulaInstallOrder(ctx, params, formula, consoleUid)
		}
	}
	if params.RuntimeContext {
		c.JsonResponseWithoutError(ctx, formulaRuntimeContextResponse(formula.Title, dynamicValues))
		return zpk_market.FormulaUpgradeVersionResult{}, false
	}
	return zpk_market.FormulaUpgradeVersionResult{}, true
}

func (c Formula) resolveFormulaInstallOrder(ctx *gin.Context, params *formulaInstallDetailParams, formula *formulalogic.Formula, consoleUid int32) (zpk_market.FormulaUpgradeVersionResult, bool) {
	// 普通购买用户从市场安装 / 升级：按用户、订单、商品、域名和应用标识验证安装资格与绑定。
	checkResult, checkErr := zpkmarket.CheckFormulaCanInstallOrUpgrade(*formula, consoleUid, params.OrderSn, params.IsUpgrade > 0, params.Reinstall, params.Domain, params.AppIdentify)
	if checkErr != nil {
		if writeFormulaAccessError(ctx, formula.GoodsId, params.OrderSn, checkErr) {
			return zpk_market.FormulaUpgradeVersionResult{}, false
		}
		c.JsonResponseWithError(ctx, checkErr, 500)
		return zpk_market.FormulaUpgradeVersionResult{}, false
	}
	if checkResult.EntitlementStatus == zpk_market.EntitlementStatusTrialExpired {
		writeFormulaAccessNotice(ctx, formula.GoodsId, checkResult, zpk_market.EntitlementStatusTrialExpired, "免费试用已到期，请购买或续费后继续使用")
		return zpk_market.FormulaUpgradeVersionResult{}, false
	}
	if !checkResult.CanInstallOrUpgrade {
		switch checkResult.ConflictReason {
		case zpk_market.InstallConflictDomainMismatch, zpk_market.InstallConflictAppIdentifyExists:
			writeFormulaInstallConflict(ctx, checkResult, params.IsUpgrade <= 0, formulaConflictNoticeContext{AppTitle: formula.Title, OrderSN: params.OrderSn, Domain: params.Domain})
		case zpk_market.InstallConflictRefundPending:
			writeFormulaAccessNotice(ctx, formula.GoodsId, checkResult, zpk_market.InstallConflictRefundPending, "订单退款处理中，暂不能安装或升级，请前往制品市场查看订单")
		default:
			writeFormulaAccessNotice(ctx, formula.GoodsId, checkResult, zpk_market.AccessReasonPurchaseRequired, "请先购买后再安装")
		}
		return zpk_market.FormulaUpgradeVersionResult{}, false
	}
	if checkResult.OrderSn != "" {
		params.OrderSn = checkResult.OrderSn
	}
	// 安装资格通过后，再解析订单允许的版本及绑定制品，支持版本限制和跨制品升级。
	upgradeResult, checkErr := zpkmarket.GetFormulaCanUpgradeVersion(*formula, consoleUid, params.OrderSn)
	slog.Info("formula upgrade result resolved",
		"upgrade_result", upgradeResult,
		"err", checkErr,
	)
	if checkErr != nil {
		if writeFormulaAccessError(ctx, formula.GoodsId, params.OrderSn, checkErr) {
			return zpk_market.FormulaUpgradeVersionResult{}, false
		}
		c.JsonResponseWithError(ctx, checkErr, 500)
		return zpk_market.FormulaUpgradeVersionResult{}, false
	}
	if upgradeResult.OrderSn != "" {
		params.OrderSn = upgradeResult.OrderSn
	}
	return upgradeResult, true
}

type formulaInstallInfo struct {
	Ingress     []logic2.Ingress     `json:"ingress"`
	Name        string               `json:"name"`
	Title       string               `json:"title"`
	Required    bool                 `json:"required"`
	StartParams []logic2.StartParams `json:"start_params"`
	Ports       []v1.ContainerPort   `json:"ports"`
	Volumes     []v1.Volume          `json:"volumes"`
}

func buildFormulaInstallInfos(formula *formulalogic.Formula, childName string) []formulaInstallInfo {
	installFormulas := make([]formulaInstallInfo, 0)
	if childName == "" {
		formulaRequiredMap := make(map[string]bool)
		for _, item := range formula.Manifest.Platform.Depends {
			formulaRequiredMap[item.Identifie] = item.Required
		}
		for index, item := range formula.AllManifest {
			if item == nil {
				continue
			}
			startParams := buildFormulaInstallStartParams(item, formula.AllManifest)
			volumes := item.Platform.Volumes
			if volumes == nil {
				volumes = make([]v1.Volume, 0)
			}
			installFormulas = append(installFormulas, formulaInstallInfo{
				Ingress:     installFormulaIngress(item.Platform),
				Name:        item.Application.Identifie,
				Title:       item.Application.Name,
				Required:    index == 0 || formulaRequiredMap[item.Application.Identifie],
				StartParams: startParams,
				Ports:       installFormulaPorts(item.Platform),
				Volumes:     volumes,
			})
		}
	}

	return installFormulas
}

func buildFormulaInstallStartParams(item *logic2.Manifest, manifests []*logic2.Manifest) []logic2.StartParams {
	startParams := formulalogic.ResolveStartParamAppReferencesCopy(
		item.Platform.StartParams,
		manifests,
		item.Application.Identifie,
	)
	hasStorageDevice := item.Platform.Storage != nil && item.Platform.Storage.Mode == logic2.StorageDevice
	if !hasStorageDevice {
		startParams = ensurePVCNameStartParam(startParams, item.Platform.Volumes)
	}
	// TODO: 移除旧安装端通过 values_text 读取候选项的临时兼容。
	for index := range startParams {
		if len(startParams[index].Options) > 0 {
			startParams[index].ValuesText = strings.Join(startParams[index].Options, "|")
		}
	}
	if startParams == nil {
		startParams = make([]logic2.StartParams, 0)
	}
	return startParams
}
