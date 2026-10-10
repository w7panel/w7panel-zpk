package ip

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/w7panel/w7panel-zpk/common/service/w7/base"
)

type GoodsService struct {
	base.Base
	NotifyBaseUrl string
}

type SetGoodsSettingReq struct {
	GoodsId         int    `json:"goods_id"`
	Appid           string `json:"origin_appid"`
	ConsoleUid      int32  `json:"user_id"`
	PayNotifyUrl    string `json:"pay_notify_url"`
	ReturnNotifyUrl string `json:"return_notify_url"`
}

func (s GoodsService) SetOrderSetting(setGoodsSettingReq SetGoodsSettingReq) error {
	if setGoodsSettingReq.GoodsId <= 0 || setGoodsSettingReq.Appid == "" || setGoodsSettingReq.ConsoleUid <= 0 {
		return errors.New("商品、通知应用和发布用户不能为空")
	}
	notifyBaseUrl := strings.TrimRight(s.NotifyBaseUrl, "/")
	if notifyBaseUrl == "" {
		notifyBaseUrl = "http://console.w7.cc"
	}
	convertSign, err := s.ConvertRequestSign(map[string]string{
		"origin_appid":      setGoodsSettingReq.Appid,
		"goods_id":          strconv.Itoa(setGoodsSettingReq.GoodsId),
		"user_id":           strconv.FormatInt(int64(setGoodsSettingReq.ConsoleUid), 10),
		"pay_notify_url":    setGoodsSettingReq.PayNotifyUrl,
		"return_notify_url": setGoodsSettingReq.ReturnNotifyUrl,
	}, notifyBaseUrl)
	if err != nil {
		return err
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest(http.MethodPut, notifyBaseUrl+"/api/thirdparty-pay/pay-goods-ip/modify-notify-url", bytes.NewReader(convertSign))
	if err != nil {
		return err
	}
	if base.DefaultUserAgent != "" {
		req.Header.Set("User-Agent", base.DefaultUserAgent)
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

	var apiError base.ApiError
	if json.Unmarshal(respBody, &apiError) == nil && apiError.ErrorMsg != "" {
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
