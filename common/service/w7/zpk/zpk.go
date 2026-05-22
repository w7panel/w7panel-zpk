package zpk

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/w7panel/w7panel-zpk/common/service/w7/base"
)

type ZpkService struct {
	base.Base
}

type Formula struct {
	Identifie            string   `form:"identifie" json:"identifie" binding:"required"`
	LatestVersion        string   `form:"latest_version" json:"latest_version" binding:"required"`
	Title                string   `form:"title" json:"title" binding:"required"`
	Tags                 []string `form:"tags" json:"tags"`
	Description          string   `form:"description" json:"description"`
	RemoteFormulaInfoUrl string   `form:"remote_formula_info_url" json:"remote_formula_info_url" binding:"required"`
}

func (s ZpkService) PushToOfficialZpkStore(formula Formula) error {
	payload, err := json.Marshal(formula)
	if err != nil {
		return err
	}
	convertSign, err := s.ConvertRequestSign(map[string]string{
		"body": string(payload),
	}, s.BaseUrl)
	if err != nil {
		return err
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("POST", s.BaseUrl+"/zpk/respo/open-api/add-from-remote", bytes.NewReader(convertSign))
	if err != nil {
		return err
	}
	req.Header.Set("x-requested-with", "XMLHttpRequest")
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	statusCode := resp.StatusCode
	if statusCode != 200 {
		var apiError base.ApiError
		err = json.Unmarshal(respBody, &apiError)
		if err != nil {
			return err
		}

		if apiError.ErrorMsg == "" {
			apiError.ErrorMsg = string(respBody)
			apiError.Code = 500
		}

		return apiError
	}

	return nil
}

type InfoRequest struct {
	Identifie    string `json:"identifie"`
	Version      string `json:"version"`
	CName        string `json:"cname"`
	IsUpgrade    int32  `json:"is_upgrade"`
	CheckUpgrade int32  `json:"check_upgrade"`
	CurVersion   string `json:"cur_version"`
	OrderSn      string `json:"order_sn"`
	ConsoleUid   int32  `json:"console_uid"`
}

func (s ZpkService) GetRemoteFormulaInfo(remoteUrl string, infoReq InfoRequest) (interface{}, error) {
	urlInfo, err := url.Parse(remoteUrl)
	if err != nil {
		return nil, err
	}
	infoUrl := urlInfo.Scheme + "://" + urlInfo.Host + "/zpk/respo/open-api/formula/info"

	reqBody, err := json.Marshal(infoReq)
	if err != nil {
		return nil, err
	}
	convertSign, err := s.ConvertRequestSign(map[string]string{
		"body": string(reqBody),
	}, infoUrl)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	req, err := http.NewRequest("POST", infoUrl, bytes.NewReader(convertSign))
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

	ret := make(map[string]interface{})
	err = json.Unmarshal(respBody, &ret)
	if err != nil {
		return nil, err
	}

	return ret["data"], nil
}

type BaseInfoResp struct {
	LatestVersion string `json:"latest_version"`
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

func (s ZpkService) DownloadRemoteFormulaIcon(remoteUrl string, savePath string) error {
	iconUrl := strings.ReplaceAll(remoteUrl, "zpk/respo/info", "zpk/zip/icon")
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	req, err := http.NewRequest("GET", iconUrl, nil)
	if err != nil {
		return err
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

	err = os.WriteFile(savePath, respBody, 0644)
	if err != nil {
		return err
	}

	return nil
}

type OrderListRequest struct {
	Limit      int    `json:"limit"`
	Page       int    `json:"page"`
	Keyword    string `json:"keyword"`
	ConsoleUid int32  `json:"console_uid"`
}

func (s ZpkService) GetRemoteFormulaOrderList(remoteUrl string, listReq OrderListRequest) (interface{}, error) {
	listUrl := strings.ReplaceAll(remoteUrl, "zpk/respo/info", "zpk/respo/open-api/order/list")

	reqBody, err := json.Marshal(listReq)
	if err != nil {
		return nil, err
	}
	convertSign, err := s.ConvertRequestSign(map[string]string{
		"body": string(reqBody),
	}, listUrl)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	req, err := http.NewRequest("POST", listUrl, bytes.NewReader(convertSign))
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

	ret := make(map[string]interface{})
	err = json.Unmarshal(respBody, &ret)
	if err != nil {
		return nil, err
	}

	return ret["data"], nil
}

func (s ZpkService) NotifyFormulaAuditResult(identifie string, remoteFormulaInfoUrl string, auditStatus int, auditMark string) error {
	urlInfo, err := url.Parse(remoteFormulaInfoUrl)
	if err != nil {
		return err
	}

	notifyUrl := urlInfo.Scheme + "://" + urlInfo.Host + "/zpk/respo/open-api/audit/notify"
	reqBody, err := json.Marshal(map[string]interface{}{
		"identifie":    identifie,
		"audit_status": auditStatus,
		"audit_remark": auditMark,
	})
	if err != nil {
		return err
	}
	convertSign, err := s.ConvertRequestSign(map[string]string{
		"body": string(reqBody),
	}, notifyUrl)
	if err != nil {
		return err
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("POST", notifyUrl, bytes.NewReader(convertSign))
	if err != nil {
		return err
	}
	req.Header.Set("x-requested-with", "XMLHttpRequest")
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	statusCode := resp.StatusCode
	if statusCode != 200 {
		var apiError base.ApiError
		err = json.Unmarshal(respBody, &apiError)
		if err != nil {
			return err
		}

		if apiError.ErrorMsg == "" {
			apiError.ErrorMsg = string(respBody)
			apiError.Code = 500
		}

		return apiError
	}

	return nil
}
