package devcenter

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/w7panel/w7panel-zpk/common/service/w7/base"
)

type NotApp struct {
	ConsoleUid  int32  `json:"console_uid"`
	Id          int    `json:"id"`
	Name        string `json:"name"`
	Title       string `json:"title"`
	Logo        string `json:"cdn_logo"`
	WindowLogo  string `json:"cdn_window_logo"`
	BranchId    int    `json:"branch_id"`
	Description string `json:"design_description"`
	GoodsId     int    `json:"goods_id"`
}

type NotAppBranch struct {
	Id int `json:"id"`
}

type NotAppVersionSupport struct {
	SupportType string `json:"support_type"`
}

type NotAppBranchVersion struct {
	Id           int                    `json:"id"`
	BranchId     int                    `json:"branch_id"`
	Price        float64                `json:"price"`
	Version      string                 `json:"version"`
	Description  string                 `json:"description"`
	Status       int                    `json:"status"`
	SupportTypes []NotAppVersionSupport `json:"version_support"`
}

type NotAppListReq struct {
	ConsoleUid int32 `json:"console_uid"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	NotAppType int   `json:"not_app_type"`
}

type NotAppListResp struct {
	CurrentPage int      `json:"current_page"`
	List        []NotApp `json:"data"`
	LastPage    int      `json:"last_page"`
	Total       int      `json:"total"`
}

type NotAppInfoReq struct {
	Id         int   `json:"id"`
	ConsoleUid int32 `json:"console_uid"`
}

type CreateNotAppReq struct {
	ConsoleUid    int32  `json:"console_uid"`
	Name          string `json:"name"`
	Description   string `json:"design_description"`
	Logo          string `json:"logo"`
	Title         string `json:"title"`
	NotAppType    int    `json:"notapp_type"`
	IsShortcut    int    `json:"is_shortcut"`
	IsFreeInstall int    `json:"is_free_install"`
	DisplayOrder  int    `json:"displayorder"`
}

type NotAppBranchInfoReq struct {
	ConsoleUid int32 `json:"console_uid"`
	AppId      int   `json:"id"`
	BranchId   int   `json:"branch_id"`
}

type NotAppServicePackage struct {
	Id           int  `json:"id"`
	Month        int  `json:"month"`
	Price        int  `json:"price"`
	IsEnable     int  `json:"enabled"`
	IsGift       bool `json:"is_gift"`
	DisplayOrder int  `json:"displayorder"`
}

type GetNotAppServicePackagesReq struct {
	ConsoleUid int32 `json:"console_uid"`
	AppId      int   `json:"id"`
}

type BranchPrice struct {
	Price     float64 `json:"price"`
	SpecPrice float64 `json:"spec_price"`
}

type NotAppBranchInfo struct {
	AppId    int          `json:"id"`
	BranchId int          `json:"branch_id"`
	Price    *BranchPrice `json:"branch_price"`
}

type NotAppBranchVersionPriceInfo struct {
	Id         int   `json:"id"`
	Price      int64 `json:"price"`
	Version    int64 `json:"version"`
	CreateTime int64 `json:"create_time"`
}

type GetNotAppBranchVersionPriceListReq struct {
	ConsoleUid int32 `json:"console_uid"`
	AppId      int   `json:"id"`
	BranchId   int   `json:"branch_id"`
}

type NotAppVersionListReq struct {
	ConsoleUid int32 `json:"console_uid"`
	AppId      int   `json:"id"`
	BranchId   int   `json:"branch_id"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
}

type NotAppVersionListResp struct {
	CurrentPage int                   `json:"current_page"`
	List        []NotAppBranchVersion `json:"data"`
	LastPage    int                   `json:"last_page"`
	Total       int                   `json:"total"`
}

type NotAppVersionAttachReq struct {
	ConsoleUid  int32  `json:"console_uid"`
	AppId       int    `json:"id"`
	BranchId    int    `json:"branch_id"`
	VersionId   int    `json:"version_id"`
	SupportType string `json:"support_type"`
}

type NotAppVersionAttachResp struct {
	VersionZipMd5 string `json:"version_zipfile"`
}

type NotAppService struct {
	base.Base
}

func (s NotAppService) NotAppList(listReq NotAppListReq) (*NotAppListResp, error) {
	convertSign, err := s.ConvertRequestSign(map[string]string{
		"not_app_type": strconv.Itoa(listReq.NotAppType),
		"page":         strconv.Itoa(listReq.Page),
		"page_size":    strconv.Itoa(listReq.PageSize),
	}, s.BaseUrl)
	if err != nil {
		return nil, err
	}
	queryParams := map[string]string{}
	err = json.Unmarshal(convertSign, &queryParams)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	for key, val := range queryParams {
		params.Add(key, val)
	}
	// 构建完整URL
	requestUrl, _ := url.Parse(s.BaseUrl + "/v2/api/notapp/list")
	requestUrl.RawQuery = params.Encode() // 自动编码特殊字符

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", requestUrl.String(), nil)
	if err != nil {
		return nil, err
	}
	if base.DefaultUserAgent != "" {
		req.Header.Set("User-Agent", base.DefaultUserAgent)
	}
	req.Header.Set("x-requested-with", "XMLHttpRequest")
	req.Header.Set("uid", strconv.Itoa(int(listReq.ConsoleUid)))
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	statusCode := resp.StatusCode
	if statusCode != 200 {
		var apiError base.ApiError
		err = json.Unmarshal(respBody, &apiError)
		if err != nil {
			return nil, err
		}
		return nil, apiError
	}

	var listResp *NotAppListResp
	err = json.Unmarshal(respBody, &listResp)
	if err != nil {
		return nil, err
	}

	return listResp, nil
}

func (s NotAppService) GetNotAppInfo(notAppQueryReq NotAppInfoReq) (*NotApp, error) {
	convertSign, err := s.ConvertRequestSign(map[string]string{}, s.BaseUrl)
	if err != nil {
		return nil, err
	}
	queryParams := map[string]string{}
	err = json.Unmarshal(convertSign, &queryParams)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	for key, val := range queryParams {
		params.Add(key, val)
	}
	// 构建完整URL
	requestUrl, _ := url.Parse(s.BaseUrl + "/v2/api/notapp/" + strconv.Itoa(notAppQueryReq.Id) + "/query")
	requestUrl.RawQuery = params.Encode() // 自动编码特殊字符

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", requestUrl.String(), nil)
	if err != nil {
		return nil, err
	}
	if base.DefaultUserAgent != "" {
		req.Header.Set("User-Agent", base.DefaultUserAgent)
	}
	req.Header.Set("x-requested-with", "XMLHttpRequest")
	req.Header.Set("uid", strconv.Itoa(int(notAppQueryReq.ConsoleUid)))
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	statusCode := resp.StatusCode
	if statusCode != 200 {
		var apiError base.ApiError
		err = json.Unmarshal(respBody, &apiError)
		if err != nil {
			return nil, err
		}
		return nil, apiError
	}

	type QueryResp struct {
		Store struct {
			Id          int    `json:"id"`
			Name        string `json:"name"`
			Title       string `json:"title"`
			Logo        string `json:"cdn_logo"`
			WindowLogo  string `json:"cdn_window_logo"`
			Description string `json:"design_description"`
		}
		Goods []struct {
			GoodsId int `json:"goods_id"`
		}
	}

	var queryResp QueryResp
	err = json.Unmarshal(respBody, &queryResp)
	if err != nil {
		return nil, err
	}

	goodsId := 0
	if len(queryResp.Goods) > 0 {
		goodsId = queryResp.Goods[0].GoodsId
	}

	return &NotApp{
		Id:          queryResp.Store.Id,
		Name:        queryResp.Store.Name,
		Title:       queryResp.Store.Title,
		Logo:        queryResp.Store.Logo,
		WindowLogo:  queryResp.Store.WindowLogo,
		Description: queryResp.Store.Description,
		GoodsId:     goodsId,
	}, nil
}

func (s NotAppService) GetNotAppBranch(notAppInfo NotApp) (*NotAppBranch, error) {
	convertSign, err := s.ConvertRequestSign(map[string]string{}, s.BaseUrl)
	if err != nil {
		return nil, err
	}
	queryParams := map[string]string{}
	err = json.Unmarshal(convertSign, &queryParams)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	for key, val := range queryParams {
		params.Add(key, val)
	}
	// 构建完整URL
	requestUrl, _ := url.Parse(s.BaseUrl + "/v2/api/notapp/" + strconv.Itoa(notAppInfo.Id) + "/branch")
	requestUrl.RawQuery = params.Encode() // 自动编码特殊字符

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", requestUrl.String(), nil)
	if err != nil {
		return nil, err
	}
	if base.DefaultUserAgent != "" {
		req.Header.Set("User-Agent", base.DefaultUserAgent)
	}
	req.Header.Set("x-requested-with", "XMLHttpRequest")
	req.Header.Set("uid", strconv.Itoa(int(notAppInfo.ConsoleUid)))
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	statusCode := resp.StatusCode
	if statusCode != 200 {
		var apiError base.ApiError
		err = json.Unmarshal(respBody, &apiError)
		if err != nil {
			return nil, err
		}
		return nil, apiError
	}

	var branchResp []NotAppBranch
	err = json.Unmarshal(respBody, &branchResp)
	if err != nil {
		return nil, err
	}
	if len(branchResp) == 0 {
		return nil, errors.New("branch 为空")
	}

	return &branchResp[0], nil
}

func (s NotAppService) GetNotAppBranchInfo(notAppBranchInfo NotAppBranchInfoReq) (*NotAppBranchInfo, error) {
	convertSign, err := s.ConvertRequestSign(map[string]string{}, s.BaseUrl)
	if err != nil {
		return nil, err
	}
	queryParams := map[string]string{}
	err = json.Unmarshal(convertSign, &queryParams)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	for key, val := range queryParams {
		params.Add(key, val)
	}
	// 构建完整URL
	requestUrl, _ := url.Parse(s.BaseUrl + "/v2/api/notapp/" + strconv.Itoa(notAppBranchInfo.AppId) + "/branch/" + strconv.Itoa(notAppBranchInfo.BranchId) + "/query")
	requestUrl.RawQuery = params.Encode() // 自动编码特殊字符

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", requestUrl.String(), nil)
	if err != nil {
		return nil, err
	}
	if base.DefaultUserAgent != "" {
		req.Header.Set("User-Agent", base.DefaultUserAgent)
	}
	req.Header.Set("x-requested-with", "XMLHttpRequest")
	req.Header.Set("uid", strconv.Itoa(int(notAppBranchInfo.ConsoleUid)))
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	statusCode := resp.StatusCode
	if statusCode != 200 {
		var apiError base.ApiError
		err = json.Unmarshal(respBody, &apiError)
		if err != nil {
			return nil, err
		}
		return nil, apiError
	}

	type InfoResp struct {
		BranchDetail struct {
			Detail NotAppBranchInfo `json:"detail"`
		} `json:"branch_detail"`
	}

	var branchResp InfoResp
	err = json.Unmarshal(respBody, &branchResp)
	if err != nil {
		return nil, err
	}
	branchResp.BranchDetail.Detail.AppId = notAppBranchInfo.AppId
	branchResp.BranchDetail.Detail.BranchId = notAppBranchInfo.BranchId

	return &branchResp.BranchDetail.Detail, nil
}

func (s NotAppService) GetNotAppServicePackages(servicePackagesReq GetNotAppServicePackagesReq) ([]NotAppServicePackage, error) {
	convertSign, err := s.ConvertRequestSign(map[string]string{}, s.BaseUrl)
	if err != nil {
		return nil, err
	}
	queryParams := map[string]string{}
	err = json.Unmarshal(convertSign, &queryParams)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	for key, val := range queryParams {
		params.Add(key, val)
	}
	// 构建完整URL
	requestUrl, _ := url.Parse(s.BaseUrl + "/v2/api/notapp/" + strconv.Itoa(servicePackagesReq.AppId) + "/service-package/list")
	requestUrl.RawQuery = params.Encode() // 自动编码特殊字符

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", requestUrl.String(), nil)
	if err != nil {
		return nil, err
	}
	if base.DefaultUserAgent != "" {
		req.Header.Set("User-Agent", base.DefaultUserAgent)
	}
	req.Header.Set("x-requested-with", "XMLHttpRequest")
	req.Header.Set("uid", strconv.Itoa(int(servicePackagesReq.ConsoleUid)))
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	statusCode := resp.StatusCode
	if statusCode != 200 {
		var apiError base.ApiError
		err = json.Unmarshal(respBody, &apiError)
		if err != nil {
			return nil, err
		}
		return nil, apiError
	}

	type InfoResp struct {
		Data []NotAppServicePackage `json:"data"`
	}

	var branchResp InfoResp
	err = json.Unmarshal(respBody, &branchResp)
	if err != nil {
		return nil, err
	}

	return branchResp.Data, nil
}

func (s NotAppService) GetNotAppBranchVersionPriceList(getNotAppBranchVersionPriceList GetNotAppBranchVersionPriceListReq) ([]NotAppBranchVersionPriceInfo, error) {
	convertSign, err := s.ConvertRequestSign(map[string]string{}, s.BaseUrl)
	if err != nil {
		return nil, err
	}
	queryParams := map[string]string{}
	err = json.Unmarshal(convertSign, &queryParams)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	for key, val := range queryParams {
		params.Add(key, val)
	}
	// 构建完整URL
	requestUrl, _ := url.Parse(s.BaseUrl + "/v2/api/notapp/" + strconv.Itoa(getNotAppBranchVersionPriceList.AppId) + "/branch/" + strconv.Itoa(getNotAppBranchVersionPriceList.BranchId) + "/upgrade/price/list")
	requestUrl.RawQuery = params.Encode() // 自动编码特殊字符

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", requestUrl.String(), nil)
	if err != nil {
		return nil, err
	}
	if base.DefaultUserAgent != "" {
		req.Header.Set("User-Agent", base.DefaultUserAgent)
	}
	req.Header.Set("x-requested-with", "XMLHttpRequest")
	req.Header.Set("uid", strconv.Itoa(int(getNotAppBranchVersionPriceList.ConsoleUid)))
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	statusCode := resp.StatusCode
	if statusCode != 200 {
		var apiError base.ApiError
		err = json.Unmarshal(respBody, &apiError)
		if err != nil {
			return nil, err
		}
		return nil, apiError
	}

	var listResp []NotAppBranchVersionPriceInfo
	err = json.Unmarshal(respBody, &listResp)
	if err != nil {
		return nil, err
	}

	return listResp, nil
}

func (s NotAppService) NotAppBranchVersionList(versionListReq NotAppVersionListReq) (*NotAppVersionListResp, error) {
	convertSign, err := s.ConvertRequestSign(map[string]string{
		"page":      strconv.Itoa(versionListReq.Page),
		"page_size": strconv.Itoa(versionListReq.PageSize),
	}, s.BaseUrl)
	if err != nil {
		return nil, err
	}
	queryParams := map[string]string{}
	err = json.Unmarshal(convertSign, &queryParams)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	for key, val := range queryParams {
		params.Add(key, val)
	}
	// 构建完整URL
	requestUrl, _ := url.Parse(s.BaseUrl + fmt.Sprintf("/v2/api/notapp/%s/branch/%s/version/list", strconv.Itoa(versionListReq.AppId), strconv.Itoa(versionListReq.BranchId)))
	requestUrl.RawQuery = params.Encode() // 自动编码特殊字符

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", requestUrl.String(), nil)
	if err != nil {
		return nil, err
	}
	if base.DefaultUserAgent != "" {
		req.Header.Set("User-Agent", base.DefaultUserAgent)
	}
	req.Header.Set("x-requested-with", "XMLHttpRequest")
	req.Header.Set("uid", strconv.Itoa(int(versionListReq.ConsoleUid)))
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	statusCode := resp.StatusCode
	if statusCode != 200 {
		var apiError base.ApiError
		err = json.Unmarshal(respBody, &apiError)
		if err != nil {
			return nil, err
		}
		return nil, apiError
	}

	var listResp *NotAppVersionListResp
	err = json.Unmarshal(respBody, &listResp)
	if err != nil {
		return nil, err
	}

	return listResp, nil
}

func (s NotAppService) NotAppBranchVersionAttach(versionAttachReq NotAppVersionAttachReq) (*NotAppVersionAttachResp, error) {
	convertSign, err := s.ConvertRequestSign(map[string]string{
		"support_type": versionAttachReq.SupportType,
	}, s.BaseUrl)
	if err != nil {
		return nil, err
	}
	queryParams := map[string]string{}
	err = json.Unmarshal(convertSign, &queryParams)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	for key, val := range queryParams {
		params.Add(key, val)
	}
	// 构建完整URL
	requestUrl, _ := url.Parse(s.BaseUrl + fmt.Sprintf("/v2/api/notapp/%s/branch/%s/version/%s/version-file", strconv.Itoa(versionAttachReq.AppId), strconv.Itoa(versionAttachReq.BranchId), strconv.Itoa(versionAttachReq.VersionId)))
	requestUrl.RawQuery = params.Encode() // 自动编码特殊字符

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", requestUrl.String(), nil)
	if err != nil {
		return nil, err
	}
	if base.DefaultUserAgent != "" {
		req.Header.Set("User-Agent", base.DefaultUserAgent)
	}
	req.Header.Set("x-requested-with", "XMLHttpRequest")
	req.Header.Set("uid", strconv.Itoa(int(versionAttachReq.ConsoleUid)))
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	statusCode := resp.StatusCode
	if statusCode != 200 {
		var apiError base.ApiError
		err = json.Unmarshal(respBody, &apiError)
		if err != nil {
			return nil, err
		}
		return nil, apiError
	}

	var attachResp *NotAppVersionAttachResp
	err = json.Unmarshal(respBody, &attachResp)
	if err != nil {
		return nil, err
	}

	return attachResp, nil
}
