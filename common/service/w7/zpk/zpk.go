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
