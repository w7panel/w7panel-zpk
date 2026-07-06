package base

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"
)

var DefaultUserAgent = ""

type ApiError struct {
	ErrorMsg string `json:"error"`
	Code     int    `json:"code"`
}

func (ve ApiError) Error() string {
	return ve.ErrorMsg
}

type Base struct {
	BaseUrl string
	Appid   string
	Secret  string
}

func (s Base) ConvertRequestSignByJson(params map[string]string, targetServerUrl string) ([]byte, error) {
	slog.Info("ConvertRequestSignByJson", "params", params, "targetServerUrl", targetServerUrl)
	url, err := url.Parse(targetServerUrl)
	if err != nil {
		return nil, err
	}
	params["appid"] = s.Appid
	params["nonce"] = s.makeRandStr(16)
	params["timestamp"] = strconv.FormatInt(time.Now().Unix(), 10)
	params["sign"] = s.getSign(params, s.Secret)
	params["target_host"] = url.Host

	paramsJson, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	paramsReader := bytes.NewReader(paramsJson)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	req, err := http.NewRequest("POST", "http://api.w7.cc/util/app/convert-sign-with-json-body", paramsReader)
	if err != nil {
		return nil, err
	}
	if DefaultUserAgent != "" {
		req.Header.Set("User-Agent", DefaultUserAgent)
	}
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
		var apiError ApiError
		err = json.Unmarshal(respBody, &apiError)
		if err != nil {
			return nil, err
		}
		return nil, apiError
	}

	return respBody, nil
}

func (s Base) ConvertRequestSign(params map[string]string, targetServerUrl string) ([]byte, error) {
	slog.Info("ConvertRequestSign", "params", params, "targetServerUrl", targetServerUrl)

	url, err := url.Parse(targetServerUrl)
	if err != nil {
		return nil, err
	}
	params["appid"] = s.Appid
	params["nonce"] = s.makeRandStr(16)
	params["timestamp"] = strconv.FormatInt(time.Now().Unix(), 10)
	params["sign"] = s.getSign(params, s.Secret)
	params["target_host"] = url.Host

	paramsJson, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	paramsReader := bytes.NewReader(paramsJson)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	req, err := http.NewRequest("POST", "http://api.w7.cc/util/app/convert-sign", paramsReader)
	if err != nil {
		return nil, err
	}
	if DefaultUserAgent != "" {
		req.Header.Set("User-Agent", DefaultUserAgent)
	}
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
		var apiError ApiError
		err = json.Unmarshal(respBody, &apiError)
		if err != nil {
			return nil, err
		}
		return nil, apiError
	}

	return respBody, nil
}

func (s Base) getSign(params map[string]string, secret string) string {
	_, ok := params["sign"]
	if ok {
		delete(params, "sign")
	}

	var keys []string
	signStr := ""
	for s, _ := range params {
		if s == "sign" {
			continue
		}
		keys = append(keys, s)
	}
	sort.Strings(keys)
	for i, k := range keys {
		signStr += fmt.Sprintf("%s=%s", url.QueryEscape(k), url.QueryEscape(params[k]))
		if i < len(keys)-1 {
			signStr += "&"
		}
	}

	signStr += secret
	md5Ctx := md5.New()
	md5Ctx.Write([]byte(signStr))
	return hex.EncodeToString(md5Ctx.Sum(nil))
}

func (s Base) makeRandStr(length int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	bytes := []byte(str)
	result := []byte{}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < length; i++ {
		result = append(result, bytes[r.Intn(len(bytes))])
	}

	return string(result)
}
