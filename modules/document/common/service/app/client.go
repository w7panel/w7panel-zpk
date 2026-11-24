// Package app implements cloudnoauth application signatures for service calls.
package app

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

var ErrSignature = errors.New("应用签名无效")
var ErrCredential = errors.New("应用签名凭据未初始化")

type Client struct {
	AppID  string
	Secret string
}

func New(appID, secret string) *Client {
	return &Client{AppID: appID, Secret: secret}
}

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

func (c *Client) Sign(request *http.Request) error {
	if c == nil || c.AppID == "" || c.Secret == "" {
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
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	params := map[string]string{
		"body":                string(body),
		"appid":               c.AppID,
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

func (c *Client) Verify(request *http.Request) error {
	if c == nil || c.AppID == "" || c.Secret == "" {
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
	if signature == "" || params["appid"] != c.AppID ||
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

func (s Client) getSign(params map[string]string, secret string) string {
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
