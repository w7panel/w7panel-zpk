package zpk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/w7panel/w7panel-zpk/common/service/w7/base"
)

type ZpkService struct {
	base.Base
}

type BaseInfoResp struct {
	LatestVersion string `json:"latest_version"`
}

type FormulaInfoResp struct {
	HelmURL     string `json:"helm_url"`
	FormulaType string `json:"formula_type"`
}

func (s ZpkService) GetRemoteFormulaInfo(ctx context.Context, infoURL string) (*FormulaInfoResp, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, infoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		var apiError base.ApiError
		if err := json.Unmarshal(body, &apiError); err == nil && apiError.ErrorMsg != "" {
			return nil, apiError
		}
		return nil, fmt.Errorf("获取制品信息返回 HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var result struct {
		Code  int             `json:"code"`
		Error string          `json:"error"`
		Data  FormulaInfoResp `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	if result.Code != 0 && result.Code != http.StatusOK {
		return nil, base.ApiError{Code: result.Code, ErrorMsg: result.Error}
	}
	if result.Data.HelmURL == "" {
		return nil, fmt.Errorf("制品信息缺少 helm_url")
	}
	return &result.Data, nil
}

func (s ZpkService) GetRemoteFormulaBaseInfo(formulaName string) (*BaseInfoResp, error) {
	infoUrl := s.BaseUrl + "/zpk/respo/open-api/formula/base-info"

	params := map[string]string{
		"identifie": formulaName,
	}
	paramsJson, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	req, err := http.NewRequest("POST", infoUrl, bytes.NewReader(paramsJson))
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-requested-with", "XMLHttpRequest")
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		var apiError base.ApiError
		err = json.Unmarshal(respBody, &apiError)
		if err != nil {
			return nil, err
		}
		return nil, apiError
	}

	type wrappedBaseInfoResp struct {
		Data BaseInfoResp `json:"data"`
	}
	var wrappedResp wrappedBaseInfoResp
	if err := json.Unmarshal(respBody, &wrappedResp); err == nil && wrappedResp.Data.LatestVersion != "" {
		return &wrappedResp.Data, nil
	}

	ret := BaseInfoResp{}
	err = json.Unmarshal(respBody, &ret)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}
