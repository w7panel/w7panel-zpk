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

const (
	Category = "document_category"
	Library  = "document_library"
	File     = "document_file"
)

var ErrDenied = errors.New("无操作权限")

type Principal struct {
	ID       int32  `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	Admin    bool   `json:"admin"`
}

type Resource struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type Grant struct {
	UserID   int32    `json:"user_id"`
	Resource Resource `json:"resource"`
	Actions  []string `json:"actions"`
}

type Decision struct {
	Allowed  bool `json:"allowed"`
	Explicit bool `json:"explicit"`
}

// Resources are ordered from the most specific resource to its parents.
type Service interface {
	Identity(context.Context, string) (Principal, error)
	Login(context.Context, string) (string, error)
	Check(context.Context, string, []Resource, string) (Decision, error)
	Replace(context.Context, string, []Grant) error
	Initialize(context.Context, string, []Grant) error
	CheckUser(context.Context, int32, []Resource, string) (Decision, error)
	Batch(context.Context, string, []Request) ([]Decision, error)
	BatchUser(context.Context, int32, []Request) ([]Decision, error)
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

func (c *Client) Check(ctx context.Context, token string, resources []Resource, action string) (Decision, error) {
	var decision Decision
	err := c.call(ctx, token, "/check", map[string]any{"resources": resources, "action": action}, &decision)
	return decision, err
}

func (c *Client) Replace(ctx context.Context, token string, grants []Grant) error {
	return c.call(ctx, token, "/replace", map[string]any{"grants": grants}, nil)
}

func (c *Client) CheckUser(ctx context.Context, userID int32, resources []Resource, action string) (Decision, error) {
	var decision Decision
	err := c.call(ctx, "", "/evaluate", map[string]any{"user_id": userID, "resources": resources, "action": action}, &decision)
	return decision, err
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

func (c *Client) Initialize(ctx context.Context, token string, grants []Grant) error {
	return c.call(ctx, token, "/initialize", map[string]interface{}{"grants": grants}, nil)
}

type Request struct {
	Resources []Resource `json:"resources"`
	Action    string     `json:"action"`
}

func (c *Client) Batch(ctx context.Context, token string, requests []Request) ([]Decision, error) {
	var decisions []Decision
	err := c.call(ctx, token, "/check-batch", map[string]interface{}{"requests": requests}, &decisions)
	if err == nil && len(decisions) != len(requests) {
		err = errors.New("权限服务响应数量无效")
	}
	return decisions, err
}

func (c *Client) BatchUser(ctx context.Context, userID int32, requests []Request) ([]Decision, error) {
	var decisions []Decision
	err := c.call(ctx, "", "/evaluate-batch", map[string]interface{}{"user_id": userID, "requests": requests}, &decisions)
	if err == nil && len(decisions) != len(requests) {
		err = errors.New("权限服务响应数量无效")
	}
	return decisions, err
}
