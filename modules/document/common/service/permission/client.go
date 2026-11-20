// Package permission is the document system's SDK for the external authorization service.
package permission

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

var ErrDenied = errors.New("无操作权限")

type Principal struct {
	ID       int32  `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	Admin    bool   `json:"admin"`
}

// Service delegates login and identity to the existing system.
type Service interface {
	Identity(context.Context, string) (Principal, error)
	Login(context.Context, string) (string, error)
}

type Client struct {
	BaseURL string
	HTTP    *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTP:    &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Client) call(ctx context.Context, token, route string, body any, out any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+route, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Zpk-Token", token)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("权限服务不可用: %w", err)
	}
	defer resp.Body.Close()
	var result struct {
		Data  json.RawMessage `json:"data"`
		Error string          `json:"error"`
	}
	if err = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&result); err != nil {
		return fmt.Errorf("权限服务响应无效: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("权限服务拒绝请求 (%d): %s", resp.StatusCode, result.Error)
	}
	if out != nil {
		return json.Unmarshal(result.Data, out)
	}
	return nil
}

func (c *Client) Identity(ctx context.Context, token string) (Principal, error) {
	var principal Principal
	err := c.call(ctx, token, "/identity", struct{}{}, &principal)
	if err == nil && principal.ID <= 0 {
		err = ErrDenied
	}
	return principal, err
}

func (c *Client) Login(ctx context.Context, accessToken string) (string, error) {
	var result struct {
		Token string `json:"token"`
	}
	err := c.call(ctx, "", "/login", map[string]interface{}{"access_token": accessToken}, &result)
	if err == nil && result.Token == "" {
		return "", ErrDenied
	}
	return result.Token, err
}
