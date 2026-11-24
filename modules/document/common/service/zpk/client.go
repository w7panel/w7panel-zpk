// Package zpk provides the document system's client for the shared ZPK user and namespace APIs.
package zpk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var ErrUnauthenticated = errors.New("用户身份无效")

type User struct {
	ID       int32  `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	Admin    bool   `json:"admin"`
}

type Client struct {
	BaseURL string
	HTTP    *http.Client
}

type ResponseError struct {
	StatusCode int
	Message    string
}

func (e *ResponseError) Error() string { return e.Message }

func New(baseURL string) *Client {
	return &Client{BaseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"), HTTP: &http.Client{Timeout: 15 * time.Second}}
}

func (c *Client) CurrentUser(ctx context.Context, accessToken string) (User, error) {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return User{}, ErrUnauthenticated
	}
	var user User
	err := c.call(ctx, accessToken, "/v2/api/user/current", struct{}{}, &user)
	var responseError *ResponseError
	if errors.As(err, &responseError) && (responseError.StatusCode == http.StatusUnauthorized || responseError.StatusCode == http.StatusForbidden) {
		return User{}, ErrUnauthenticated
	}
	if err != nil {
		return User{}, err
	}
	if user.ID <= 0 || user.Username == "" || user.Role == "" {
		return User{}, errors.New("ZPK 用户信息无效")
	}
	return user, nil
}

func (c *Client) call(ctx context.Context, token, route string, payload, output any) error {
	if c.BaseURL == "" {
		return errors.New("zpk.base_url 未配置")
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+route, bytes.NewReader(data))
	if err != nil {
		return err
	}
	if token != "" {
		request.Header.Set("X-W7Panel-Access-Token", token)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := c.HTTP.Do(request)
	if err != nil {
		return fmt.Errorf("ZPK 服务不可用: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return &ResponseError{StatusCode: response.StatusCode, Message: fmt.Sprintf("ZPK 服务返回 %d", response.StatusCode)}
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if err != nil {
		return err
	}
	if len(body) > 1<<20 {
		return errors.New("ZPK 服务响应过长")
	}
	var result struct {
		Data  json.RawMessage `json:"data"`
		Error string          `json:"error"`
		Code  int             `json:"code"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("ZPK 服务响应无效（HTTP %d）", response.StatusCode)
	}
	if response.StatusCode != http.StatusOK || result.Error != "" || (result.Code != 0 && result.Code != 200) {
		if result.Error == "" {
			result.Error = fmt.Sprintf("ZPK 服务返回 %d", response.StatusCode)
		}
		return &ResponseError{StatusCode: response.StatusCode, Message: result.Error}
	}
	if output != nil {
		if len(result.Data) == 0 || string(result.Data) == "null" {
			return errors.New("ZPK 服务响应缺少数据")
		}
		return json.Unmarshal(result.Data, output)
	}
	var success string
	if err := json.Unmarshal(result.Data, &success); err != nil || success != "success" {
		return errors.New("ZPK 服务未确认操作成功")
	}
	return nil
}
