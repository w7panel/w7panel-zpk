package devcenter

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/w7panel/w7panel-zpk/common/service/w7/base"
)

const W7ZpkGoodsCategoryId = 13

type PublishGoodsReq struct {
	ConsoleUid  int                      `json:"console_uid"`
	Id          int                      `json:"id"`
	Title       string                   `json:"title"`
	Description string                   `json:"description"`
	Summary     string                   `json:"summary"`
	GoodsType   int                      `json:"category_id"`
	Logo        string                   `json:"logo"`
	WindowLogo  string                   `json:"window_logo"`
	GoodsImgs   []map[string]string      `json:"window_slides"`
	RespoUrl    string                   `json:"respo_url"`
	UnitPrice   float64                  `json:"unit_price"`
	Water       int                      `json:"water"`
	Products    []map[string]interface{} `json:"products"`
	LabelIds    []int                    `json:"label_ids"`
	Enable      int                      `json:"enable"`
	AuditStatus int                      `json:"audit_status"`
	OnShelf     int                      `json:"on_shelf"`
	Extra       interface{}              `json:"extra"`
}

type PublishGoodsResp struct {
	Id int `json:"id"`
}

type PublishGoodsInfoReq struct {
	ConsoleUid int `json:"console_uid"`
	Id         int `json:"id"`
}

type GoodsProduct struct {
	Id    int     `json:"id"`
	Price float64 `json:"unit_price"`
}

type PublishGoodsInfo struct {
	Id            int                      `json:"id"`
	Title         string                   `json:"title"`
	Description   string                   `json:"description"`
	GoodsType     int                      `json:"category_id"`
	Logo          string                   `json:"logo"`
	WindowLogo    string                   `json:"window_logo"`
	GoodsImgs     []map[string]string      `json:"slides_arr"`
	Water         int                      `json:"water"`
	Products      []map[string]interface{} `json:"products"`
	ProductsInfo  []GoodsProduct
	Labels        []Label     `json:"labels"`
	Ext           interface{} `json:"ext"`
	AuditStatus   int         `json:"audit_status"`
	ServiceConfig struct {
		GiveMonth int `json:"give_month"`
	} `json:"service_config"`
	OnShelf int `json:"on_shelf"`
}

type GoodsLabelsReq struct {
	Title    string `json:"title"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}

type Label struct {
	Id         int    `json:"id"`
	Title      string `json:"title"`
	GoodsCount int    `json:"goods_count"`
	CreatedAt  string `json:"created_at"`
}

type GoodsLabelsResp struct {
	CurrentPage int     `json:"current_page"`
	List        []Label `json:"data"`
	LastPage    int     `json:"last_page"`
	Total       int     `json:"total"`
}

type AddLabelReq struct {
	Title string `json:"title"`
}

type ChangeGoodsStatusReq struct {
	ConsoleUid int `json:"console_uid"`
	GoodsId    int `json:"goods_id"`
	Status     int `json:"on_shelf"`
}

type GoodsService struct {
	base.Base
}

func (s GoodsService) PublishGoods(publishGoodsReq PublishGoodsReq) (*PublishGoodsResp, error) {
	reqBody, err := json.Marshal(publishGoodsReq)
	if err != nil {
		return nil, err
	}
	convertSign, err := s.ConvertRequestSignByJson(map[string]string{
		"body": string(reqBody),
	}, s.BaseUrl)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("POST", s.BaseUrl+"/v2/api/publish-goods/publish", bytes.NewReader(convertSign))
	if err != nil {
		return nil, err
	}
	if base.DefaultUserAgent != "" {
		req.Header.Set("User-Agent", base.DefaultUserAgent)
	}
	req.Header.Set("x-requested-with", "XMLHttpRequest")
	req.Header.Set("uid", strconv.Itoa(int(publishGoodsReq.ConsoleUid)))
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

	statusCode := resp.StatusCode
	if statusCode != 200 {
		slog.Error("发布商品", "req", publishGoodsReq, "resp", string(respBody))
		var apiError base.ApiError
		err = json.Unmarshal(respBody, &apiError)
		if err != nil {
			return nil, err
		}

		if apiError.ErrorMsg == "" {
			apiError.ErrorMsg = string(respBody)
			apiError.Code = 500
		}

		return nil, apiError
	}

	var publishResp *PublishGoodsResp
	err = json.Unmarshal(respBody, &publishResp)
	if err != nil {
		return nil, err
	}

	return publishResp, nil
}

func (s GoodsService) PublishGoodsInfo(publishGoodsInfoReq PublishGoodsInfoReq) (*PublishGoodsInfo, error) {
	convertSign, err := s.ConvertRequestSign(map[string]string{}, s.BaseUrl)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("POST", s.BaseUrl+"/v2/api/publish-goods/detail/"+strconv.Itoa(publishGoodsInfoReq.Id), bytes.NewReader(convertSign))
	if err != nil {
		return nil, err
	}
	if base.DefaultUserAgent != "" {
		req.Header.Set("User-Agent", base.DefaultUserAgent)
	}
	req.Header.Set("x-requested-with", "XMLHttpRequest")
	req.Header.Set("uid", strconv.Itoa(int(publishGoodsInfoReq.ConsoleUid)))
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

	statusCode := resp.StatusCode
	if statusCode != 200 {
		var apiError base.ApiError
		err = json.Unmarshal(respBody, &apiError)
		if err != nil {
			return nil, err
		}
		return nil, apiError
	}

	var publishResp *PublishGoodsInfo
	err = json.Unmarshal(respBody, &publishResp)
	if err != nil {
		return nil, err
	}
	if publishResp.Products != nil {
		publishResp.ProductsInfo = make([]GoodsProduct, 0)
		for _, item := range publishResp.Products {
			price, exists := item["unit_price"]
			if !exists {
				continue
			}
			id, exists := item["id"]
			if !exists {
				continue
			}

			publishResp.ProductsInfo = append(publishResp.ProductsInfo, GoodsProduct{
				Price: price.(float64),
				Id:    int(id.(float64)),
			})
		}
	}

	return publishResp, nil
}

func (s GoodsService) ChangeGoodsStatus(changeGoodsStatusReq ChangeGoodsStatusReq) error {
	convertSign, err := s.ConvertRequestSign(map[string]string{
		"goods_id": strconv.Itoa(changeGoodsStatusReq.GoodsId),
		"on_shelf": strconv.Itoa(changeGoodsStatusReq.Status),
	}, s.BaseUrl)
	if err != nil {
		return err
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("POST", s.BaseUrl+"/v2/api/publish-goods/change-status", bytes.NewReader(convertSign))
	if err != nil {
		return err
	}
	if base.DefaultUserAgent != "" {
		req.Header.Set("User-Agent", base.DefaultUserAgent)
	}
	req.Header.Set("x-requested-with", "XMLHttpRequest")
	req.Header.Set("uid", strconv.Itoa(int(changeGoodsStatusReq.ConsoleUid)))
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

func (s GoodsService) GoodsLabels(goodsLabelReq GoodsLabelsReq) (*GoodsLabelsResp, error) {
	convertSign, err := s.ConvertRequestSign(map[string]string{
		"title":     goodsLabelReq.Title,
		"page":      strconv.Itoa(goodsLabelReq.Page),
		"page_size": strconv.Itoa(goodsLabelReq.PageSize),
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
	requestUrl, _ := url.Parse(s.BaseUrl + "/v2/api/publish-goods/label")
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

		if apiError.ErrorMsg == "" {
			apiError.ErrorMsg = string(respBody)
			apiError.Code = 500
		}
		return nil, apiError
	}

	var labelsResp *GoodsLabelsResp
	err = json.Unmarshal(respBody, &labelsResp)
	if err != nil {
		return nil, err
	}

	return labelsResp, nil
}

func (s GoodsService) AddLabel(addLabelReq AddLabelReq) (*Label, error) {
	convertSign, err := s.ConvertRequestSign(map[string]string{
		"title": addLabelReq.Title,
	}, s.BaseUrl)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	req, err := http.NewRequest("POST", s.BaseUrl+"/v2/api/publish-goods/add-label", bytes.NewReader(convertSign))
	if err != nil {
		return nil, err
	}
	if base.DefaultUserAgent != "" {
		req.Header.Set("User-Agent", base.DefaultUserAgent)
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

	statusCode := resp.StatusCode
	if statusCode != 200 {
		var apiError base.ApiError
		err = json.Unmarshal(respBody, &apiError)
		if err != nil {
			return nil, err
		}

		if apiError.ErrorMsg == "" {
			apiError.ErrorMsg = string(respBody)
			apiError.Code = 500
		}
		return nil, apiError
	}

	var label *Label
	err = json.Unmarshal(respBody, &label)
	if err != nil {
		return nil, err
	}

	return label, nil
}
