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

	"github.com/w7panel/w7panel-zpk/common/service/w7/base"
)

type Client struct {
	base.Base
	HTTP *http.Client
}
type ResponseError struct {
	StatusCode int
	Message    string
}

func (e *ResponseError) Error() string {
	return fmt.Sprintf("权限服务返回 %d: %s", e.StatusCode, e.Message)
}
func IsNotFound(err error) bool {
	var response *ResponseError
	return errors.As(err, &response) && response.StatusCode == http.StatusNotFound
}

type Resource struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}
type ResourceInfo struct {
	VisibleType  int32  `json:"visible_type"`
	ResourceType string `json:"resource_type"`
	UserID       int32  `json:"user_id"`
}
type Namespace struct {
	ID          int32  `json:"id"`
	Name        string `json:"name"`
	VisibleType int32  `json:"visible_type"`
}

type Grant struct {
	UserID   int32    `json:"user_id"`
	Resource Resource `json:"resource"`
	Actions  []string `json:"actions"`
}

func New(serverURL, appID, secret string) *Client {
	return &Client{Base: base.Base{BaseUrl: strings.TrimRight(serverURL, "/"), Appid: appID, Secret: secret}, HTTP: &http.Client{Timeout: 15 * time.Second}}
}
func (c *Client) call(ctx context.Context, route string, body, output any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseUrl+"/zpk/v2/openapi"+route, bytes.NewReader(data))
	if err != nil {
		return err
	}
	if err := c.Sign(request); err != nil {
		return err
	}
	response, err := c.HTTP.Do(request)
	if err != nil {
		return fmt.Errorf("权限服务不可用: %w", err)
	}
	defer response.Body.Close()
	var result struct {
		Data  json.RawMessage `json:"data"`
		Error string          `json:"error"`
		Code  int             `json:"code"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&result); err != nil {
		return err
	}
	if response.StatusCode != http.StatusOK {
		return &ResponseError{StatusCode: response.StatusCode, Message: result.Error}
	}
	if result.Error != "" || (result.Code != 0 && result.Code != 200) {
		return fmt.Errorf("权限服务请求失败 (%d): %s", response.StatusCode, result.Error)
	}
	if output != nil {
		return json.Unmarshal(result.Data, output)
	}
	var success string
	if err := json.Unmarshal(result.Data, &success); err != nil || success != "success" {
		return errors.New("权限服务未确认设置成功")
	}
	return nil
}
func (c *Client) Add(ctx context.Context, grant Grant) error {
	return c.call(ctx, "/permission/add", grant, nil)
}
func (c *Client) Delete(ctx context.Context, userID int32, resource Resource) error {
	return c.call(ctx, "/permission/delete", map[string]any{"user_id": userID, "resource": resource}, nil)
}
func (c *Client) Clear(ctx context.Context, userID int32) error {
	return c.call(ctx, "/permission/clear", map[string]any{"user_id": userID}, nil)
}

func (c *Client) Check(ctx context.Context, userID int32, resource Resource, action, token string) (bool, error) {
	var decision *struct {
		Allowed bool `json:"allowed"`
	}
	err := c.call(ctx, "/resource/check", map[string]any{"resource": resource, "user_id": userID, "action": action, "token": token}, &decision)
	if err != nil {
		return false, err
	}
	if decision == nil {
		return false, errors.New("权限服务响应无效")
	}
	return decision.Allowed, nil
}

func (c *Client) GetResource(ctx context.Context, resource Resource) (ResourceInfo, error) {
	var result ResourceInfo
	err := c.call(ctx, "/resource/get", map[string]any{"resource": resource}, &result)
	return result, err
}
func (c *Client) SetResource(ctx context.Context, resource Resource, userID, visibleType int32) error {
	return c.call(ctx, "/resource/set", map[string]any{"resource": resource, "user_id": userID, "visible_type": visibleType}, nil)
}
func (c *Client) GetNamespace(ctx context.Context, name string) (Namespace, error) {
	var result Namespace
	err := c.call(ctx, "/namespace/get", map[string]any{"name": name}, &result)
	return result, err
}
func (c *Client) EnsureNamespace(ctx context.Context, name string) error {
	_, err := c.GetNamespace(ctx, name)
	if err == nil {
		return nil
	}
	if !IsNotFound(err) {
		return err
	}
	return c.call(ctx, "/namespace/create", map[string]any{"name": name, "visible_type": 1, "visible": false}, nil)
}
