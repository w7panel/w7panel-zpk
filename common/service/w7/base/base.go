package base

import (
	"bytes"
	"context"
	"crypto/md5"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
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
	return s.ConvertRequestSignWithContext(context.Background(), params, targetServerUrl)
}

func (s Base) ConvertRequestSignWithContext(ctx context.Context, params map[string]string, targetServerUrl string) ([]byte, error) {
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
	req, err := http.NewRequestWithContext(ctx, "POST", "http://api.w7.cc/util/app/convert-sign", paramsReader)
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
	delete(params, "sign")
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, url.QueryEscape(key)+"="+url.QueryEscape(params[key]))
	}
	digest := md5.Sum([]byte(strings.Join(parts, "&") + secret))
	return hex.EncodeToString(digest[:])
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

var ErrSignature = errors.New("应用签名无效")
var ErrCredential = errors.New("应用签名凭据未初始化")

func readBody(request *http.Request) (map[string]string, error) {
	if request.Body == nil {
		return nil, ErrSignature
	}
	body, err := io.ReadAll(request.Body)
	if err != nil {
		return nil, err
	}
	request.Body.Close()
	request.Body = io.NopCloser(bytes.NewReader(body))
	decoder := json.NewDecoder(bytes.NewReader(body))
	var data map[string]string
	if err := decoder.Decode(&data); err != nil || data == nil {
		return nil, ErrSignature
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return nil, ErrSignature
	}
	return data, nil
}

func tokenDigest(request *http.Request) string {
	digest := sha256.Sum256([]byte(request.Header.Get("X-W7Panel-Access-Token") + "\x00" + request.Header.Get("X-Document-Token")))
	return hex.EncodeToString(digest[:])
}

func (c *Base) Sign(request *http.Request) error {
	if c == nil || c.Appid == "" || c.Secret == "" {
		return ErrCredential
	}
	if request.Body == nil {
		return ErrSignature
	}
	body, err := io.ReadAll(request.Body)
	if err != nil {
		return err
	}
	request.Body.Close()
	request.Body = io.NopCloser(bytes.NewReader(body))
	if !json.Valid(body) {
		return ErrSignature
	}
	nonce := make([]byte, 16)
	if _, err := cryptorand.Read(nonce); err != nil {
		return err
	}
	params := map[string]string{
		"body":                string(body),
		"appid":               c.Appid,
		"timestamp":           strconv.FormatInt(time.Now().Unix(), 10),
		"nonce":               hex.EncodeToString(nonce),
		"request_method":      request.Method,
		"request_uri":         request.URL.RequestURI(),
		"access_token_sha256": tokenDigest(request),
	}
	params["sign"] = c.getSign(params, c.Secret)
	body, err = json.Marshal(params)
	if err != nil {
		return err
	}
	request.Body.Close()
	request.Body = io.NopCloser(bytes.NewReader(body))
	request.ContentLength = int64(len(body))
	request.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(body)), nil }
	request.Header.Set("Content-Type", "application/json")
	return nil
}

func (c *Base) Verify(request *http.Request) error {
	if c == nil || c.Appid == "" || c.Secret == "" {
		return ErrCredential
	}
	if request.Method != http.MethodPost || !strings.HasPrefix(strings.ToLower(request.Header.Get("Content-Type")), "application/json") {
		return ErrSignature
	}
	params, err := readBody(request)
	if err != nil {
		return ErrSignature
	}
	signature := params["sign"]
	if signature == "" || params["appid"] != c.Appid ||
		params["request_method"] != request.Method || params["request_uri"] != request.URL.RequestURI() ||
		params["access_token_sha256"] != tokenDigest(request) || len(params["nonce"]) != 32 {
		return ErrSignature
	}
	timestamp, err := strconv.ParseInt(params["timestamp"], 10, 64)
	now := time.Now().Unix()
	if err != nil || timestamp < now-300 || timestamp > now+300 {
		return ErrSignature
	}
	expected := c.getSign(params, c.Secret)
	if subtle.ConstantTimeCompare([]byte(signature), []byte(expected)) != 1 || !json.Valid([]byte(params["body"])) {
		return ErrSignature
	}
	request.Body.Close()
	request.Body = io.NopCloser(strings.NewReader(params["body"]))
	request.ContentLength = int64(len(params["body"]))
	request.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(strings.NewReader(params["body"])), nil }
	return nil
}
