package ip

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/w7panel/w7panel-zpk/common/service/w7/base"
)

type GoodsService struct {
	base.Base
}

type SetGoodsSettingReq struct {
	GoodsId         int    `json:"goods_id"`
	PayNotifyUrl    string `json:"pay_notify_url"`
	RefundNotifyUrl string `json:"refund_notify_url"`
}

func (s GoodsService) SetOrderSetting(setGoodsSettingReq SetGoodsSettingReq) error {
	convertSign, err := s.ConvertRequestSign(map[string]string{
		"notify_appid":      s.Appid,
		"goods_id":          strconv.Itoa(setGoodsSettingReq.GoodsId),
		"pay_notify_url":    setGoodsSettingReq.PayNotifyUrl,
		"refund_notify_url": setGoodsSettingReq.RefundNotifyUrl,
	}, s.BaseUrl)
	if err != nil {
		return err
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("POST", s.BaseUrl+"/ddd-order/sdk/set-goods-setting", bytes.NewReader(convertSign))
	if err != nil {
		return err
	}
	if base.DefaultUserAgent != "" {
		req.Header.Set("User-Agent", base.DefaultUserAgent)
	}
	req.Header.Set("x-requested-with", "XMLHttpRequest")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
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
	if statusCode != 200 && statusCode != 201 {
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

type GoodsListItem struct {
	Id           int    `json:"id"`
	ConsoleUid   int64  `json:"user_id"`
	OnShelf      int    `json:"on_shelf"`
	AuditStatus  int    `json:"audit_status"`
	AuditMessage string `json:"audit_message"`
	CreatedAt    string `json:"created_at"`
	CategoryId   int    `json:"category_id"`
	ProductId    int    `json:"product_id"`
}

type GoodsBatchListReq struct {
	GoodsIds []int `json:"goods_ids"`
}

type GoodsBatchListResp struct {
	CurrentPage int             `json:"current_page"`
	List        []GoodsListItem `json:"data"`
	LastPage    int             `json:"last_page"`
	Total       int             `json:"total"`
}

func (s GoodsService) GoodsBatchList(listReq GoodsBatchListReq) ([]GoodsListItem, error) {
	reqBody, err := json.Marshal(listReq)
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

	req, err := http.NewRequest("POST", s.BaseUrl+"/ddd-order/sdk/goods-index-batch", bytes.NewReader(convertSign))
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
	if statusCode != 200 && statusCode != 201 {
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

	listResp := make([]GoodsListItem, 0)
	err = json.Unmarshal(respBody, &listResp)
	if err != nil {
		return nil, err
	}

	return listResp, nil
}
