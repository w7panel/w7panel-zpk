package zpk

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
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
