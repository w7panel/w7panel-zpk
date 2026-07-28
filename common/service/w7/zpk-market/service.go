package zpk_market

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/w7panel/w7panel-zpk/common/service/w7/base"
)

type ZpkMarketService struct {
	base.Base
}

const (
	InstallConflictDomainMismatch    = "domain_mismatch"
	InstallConflictAppIdentifyExists = "app_identify_exists"
)

type apiResponse[T any] struct {
	Code int    `json:"code"`
	Data T      `json:"data"`
	Err  string `json:"error"`
}

func (s ZpkMarketService) CheckToken(token, formulaIdentify string) error {
	return postSigned[any](s, "/zpk-market/formula/check-token", map[string]interface{}{
		"token":            token,
		"formula_identify": formulaIdentify,
	}, nil)
}

func (s ZpkMarketService) UseOrder(consoleUid int32, orderSn, formulaVersion string, isUpgrade, reinstall bool, panelDeviceSN, panelURL, appIdentify, domain string) error {
	return postSigned[any](s, "/zpk-market/order/use-order", map[string]interface{}{
		"order_sn":        orderSn,
		"formula_version": formulaVersion,
		"is_upgrade":      isUpgrade,
		"reinstall":       reinstall,
		"console_uid":     consoleUid,
		"panel_device_sn": panelDeviceSN,
		"panel_url":       panelURL,
		"app_identify":    appIdentify,
		"domain":          domain,
	}, nil)
}

func (s ZpkMarketService) DiscardUsedOrder(consoleUid int32, orderSn string) error {
	return postSigned[any](s, "/zpk-market/order/discard-used-order", map[string]interface{}{
		"order_sn":    orderSn,
		"console_uid": consoleUid,
	}, nil)
}

func (s ZpkMarketService) CheckFormulaCanInstallOrUpgrade(goodsId, consoleUid int32, orderSn string, isUpgrade, reinstall bool, domain, appIdentify string) (bool, string, string, string, string, string, error) {
	type result struct {
		CanInstallOrUpgrade bool   `json:"can_install_or_upgrade"`
		PanelURL            string `json:"panel_url"`
		PanelDeviceSN       string `json:"panel_device_sn"`
		ConflictReason      string `json:"conflict_reason"`
		Domain              string `json:"domain"`
		AppIdentify         string `json:"app_identify"`
	}

	ret := result{}
	err := postSigned(s, "/zpk-market/order/check-formula-can-install-or-upgrade", map[string]interface{}{
		"goods_id":     goodsId,
		"console_uid":  consoleUid,
		"order_sn":     orderSn,
		"is_upgrade":   isUpgrade,
		"reinstall":    reinstall,
		"domain":       domain,
		"app_identify": appIdentify,
	}, &ret)
	if err != nil {
		return false, "", "", "", "", "", err
	}

	return ret.CanInstallOrUpgrade, ret.PanelURL, ret.PanelDeviceSN, ret.ConflictReason, ret.Domain, ret.AppIdentify, nil
}

func (s ZpkMarketService) GetFormulaCanUpgradeVersion(goodsId, consoleUid int32, orderSn string) (string, bool, string, string, error) {
	type result struct {
		FormulaVersion  string `json:"formula_version"`
		IsUpgrade       bool   `json:"is_upgrade"`
		FormulaIdentify string `json:"formula_identify"`
		OrderSn         string `json:"order_sn"`
	}

	ret := result{}
	err := postSigned(s, "/zpk-market/order/get-formula-can-upgrade-version", map[string]interface{}{
		"goods_id":    goodsId,
		"console_uid": consoleUid,
		"order_sn":    orderSn,
	}, &ret)
	if err != nil {
		return "", false, "", "", err
	}

	return ret.FormulaVersion, ret.IsUpgrade, ret.FormulaIdentify, ret.OrderSn, nil
}

func postSigned[T any](s ZpkMarketService, path string, params map[string]interface{}, result *T) error {
	targetUrl := s.BaseUrl + path
	payload, err := json.Marshal(params)
	if err != nil {
		return err
	}
	convertSign, err := s.ConvertRequestSign(map[string]string{
		"body": string(payload),
	}, targetUrl)
	if err != nil {
		return err
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	req, err := http.NewRequest("POST", targetUrl, bytes.NewReader(convertSign))
	if err != nil {
		return err
	}
	req.Header.Set("x-requested-with", "XMLHttpRequest")
	req.Header.Set("Content-Type", "application/json")
	if base.DefaultUserAgent != "" {
		req.Header.Set("User-Agent", base.DefaultUserAgent)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		apiError := base.ApiError{}
		if err := json.Unmarshal(respBody, &apiError); err != nil {
			return err
		}
		if apiError.ErrorMsg == "" {
			apiError.ErrorMsg = string(respBody)
			apiError.Code = resp.StatusCode
		}
		return apiError
	}

	wrappedResp := apiResponse[T]{}
	if err := json.Unmarshal(respBody, &wrappedResp); err != nil {
		return err
	}
	if wrappedResp.Code >= http.StatusBadRequest || wrappedResp.Err != "" {
		if wrappedResp.Err == "" {
			wrappedResp.Err = "zpk market api error"
		}
		return errors.New(wrappedResp.Err)
	}
	if result != nil {
		*result = wrappedResp.Data
	}

	return nil
}
