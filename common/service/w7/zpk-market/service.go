package zpk_market

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/w7panel/w7panel-zpk/common/service/w7/base"
)

type ZpkMarketService struct {
	base.Base
}

type apiResponse[T any] struct {
	Code int    `json:"code"`
	Data T      `json:"data"`
	Err  string `json:"error"`
}

func (s ZpkMarketService) CheckToken(token, formulaIdentify string) error {
	return postSigned[any](s, "/zpk-market/formula/check-token", map[string]string{
		"token":            token,
		"formula_identify": formulaIdentify,
	}, nil)
}

func (s ZpkMarketService) UseOrder(orderSn, formulaVersion string, isUpgrade bool) error {
	return postSigned[any](s, "/zpk-market/order/use-order", map[string]string{
		"order_sn":        orderSn,
		"formula_version": formulaVersion,
		"is_upgrade":      strconv.FormatBool(isUpgrade),
	}, nil)
}

func (s ZpkMarketService) DiscardUsedOrder(orderSn string) error {
	return postSigned[any](s, "/zpk-market/order/discard-used-order", map[string]string{
		"order_sn": orderSn,
	}, nil)
}

func (s ZpkMarketService) CheckFormulaCanInstallOrUpgrade(goodsId, consoleUid int32, orderSn string, isUpgrade bool) (bool, error) {
	type result struct {
		CanInstallOrUpgrade bool `json:"can_install_or_upgrade"`
	}

	ret := result{}
	err := postSigned(s, "/zpk-market/order/check-formula-can-install-or-upgrade", map[string]string{
		"goods_id":    strconv.Itoa(int(goodsId)),
		"console_uid": strconv.Itoa(int(consoleUid)),
		"order_sn":    orderSn,
		"is_upgrade":  strconv.FormatBool(isUpgrade),
	}, &ret)
	if err != nil {
		return false, err
	}

	return ret.CanInstallOrUpgrade, nil
}

func (s ZpkMarketService) GetFormulaCanUpgradeVersion(goodsId, consoleUid int32, orderSn string) (string, bool, error) {
	type result struct {
		FormulaVersion string `json:"formula_version"`
		IsUpgrade      bool   `json:"is_upgrade"`
	}

	ret := result{}
	err := postSigned(s, "/zpk-market/order/get-formula-can-upgrade-version", map[string]string{
		"goods_id":    strconv.Itoa(int(goodsId)),
		"console_uid": strconv.Itoa(int(consoleUid)),
		"order_sn":    orderSn,
	}, &ret)
	if err != nil {
		return "", false, err
	}

	return ret.FormulaVersion, ret.IsUpgrade, nil
}

func postSigned[T any](s ZpkMarketService, path string, params map[string]string, result *T) error {
	targetUrl := s.BaseUrl + path
	convertSign, err := s.ConvertRequestSign(params, targetUrl)
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
