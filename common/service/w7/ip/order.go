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

type OrderService struct {
	base.Base
}

type CreateOrderReq struct {
	ConsoleUid int32   `json:"console_uid"`
	ProductId  int     `json:"product_id"`
	Quantity   int     `json:"quantity"`
	Price      float64 `json:"price"`
}

type CreateOrderResp struct {
	OrderId   int    `json:"order_id"`
	OrderSn   string `json:"order_sn"`
	QrcodeUrl string `json:"qrcode_url"`
	Ticket    string `json:"ticket"`
}

type CheckoutOrderReq struct {
	OrderId int    `json:"order_id"`
	Token   string `json:"token"`
}

type CheckoutOrderResp struct {
	PaymentId int `json:"payment_id"`
}

type GetOrderPayInfoReq struct {
	PaymentId int `json:"payment_id"`
}

type OrderPayInfo struct {
	QrcodeUrl string `json:"qrcode_url"`
	Ticket    string `json:"ticket"`
}

func (s OrderService) CreateOrder(createOrderReq CreateOrderReq) (*CreateOrderResp, error) {
	convertSign, err := s.ConvertRequestSign(map[string]string{
		"uid":        strconv.Itoa(int(createOrderReq.ConsoleUid)),
		"product_id": strconv.Itoa(createOrderReq.ProductId),
		"quantity":   strconv.Itoa(createOrderReq.Quantity),
		"price":      strconv.FormatFloat(createOrderReq.Price, 'f', 2, 64),
	}, s.BaseUrl)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("POST", s.BaseUrl+"/ddd-order/sdk/create-order-and-change-price", bytes.NewReader(convertSign))
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

	var createOrderResp *CreateOrderResp
	err = json.Unmarshal(respBody, &createOrderResp)
	if err != nil {
		return nil, err
	}

	return createOrderResp, nil
}

func (s OrderService) CheckoutOrder(checkoutOrderReq CheckoutOrderReq) (*CheckoutOrderResp, error) {
	params, err := json.Marshal(map[string]interface{}{
		"data": map[string]interface{}{
			"order_ids": []int{checkoutOrderReq.OrderId},
		},
	})
	if err != nil {
		return nil, err
	}
	convertSign, err := s.ConvertRequestSignByJson(map[string]string{
		"body": string(params),
	}, s.BaseUrl)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("POST", s.BaseUrl+"/ddd-order/share/command/backend/check_out", bytes.NewReader(convertSign))
	if err != nil {
		return nil, err
	}
	if base.DefaultUserAgent != "" {
		req.Header.Set("User-Agent", base.DefaultUserAgent)
	}
	req.Header.Set("Authorization", "Bearer "+checkoutOrderReq.Token)
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
		return nil, err
	}

	var checkoutOrderResp *CheckoutOrderResp
	err = json.Unmarshal(respBody, &checkoutOrderResp)
	if err != nil {
		return nil, err
	}

	return checkoutOrderResp, nil
}

func (s OrderService) GetOrderPayInfo(orderPayInfoReq GetOrderPayInfoReq) (*OrderPayInfo, error) {
	params, err := json.Marshal(map[string]interface{}{
		"data": map[string]interface{}{
			"payment_id": orderPayInfoReq.PaymentId,
		},
	})
	if err != nil {
		return nil, err
	}
	convertSign, err := s.ConvertRequestSignByJson(map[string]string{
		"body": string(params),
	}, s.BaseUrl)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("POST", s.BaseUrl+"/ddd-order/share/sdk/command/qrcode", bytes.NewReader(convertSign))
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
		return nil, err
	}

	var orderPayInfoResp *OrderPayInfo
	err = json.Unmarshal(respBody, &orderPayInfoResp)
	if err != nil {
		return nil, err
	}

	return orderPayInfoResp, nil
}

func (s OrderService) Login(consoleUid int) (string, error) {
	convertSign, err := s.ConvertRequestSign(map[string]string{
		"uid":  strconv.Itoa(consoleUid),
		"role": "customer",
	}, s.BaseUrl)
	if err != nil {
		return "", err
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("POST", s.BaseUrl+"/ddd-order/share/sdk/login", bytes.NewReader(convertSign))
	if err != nil {
		return "", err
	}
	if base.DefaultUserAgent != "" {
		req.Header.Set("User-Agent", base.DefaultUserAgent)
	}
	req.Header.Set("x-requested-with", "XMLHttpRequest")
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	statusCode := resp.StatusCode
	if statusCode != 200 && statusCode != 201 {
		var apiError base.ApiError
		err = json.Unmarshal(respBody, &apiError)
		if err != nil {
			return "", err
		}
		if apiError.ErrorMsg == "" {
			apiError.ErrorMsg = string(respBody)
			apiError.Code = 500
		}
		return "", apiError
	}

	type LoginResp struct {
		Token string `json:"token"`
	}

	var loginResp *LoginResp
	err = json.Unmarshal(respBody, &loginResp)
	if err != nil {
		return "", err
	}

	return loginResp.Token, nil
}
