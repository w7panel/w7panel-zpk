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

	"github.com/w7panel/w7panel-zpk/modules/document/common/service/app"
)

type ResponseError struct {
	StatusCode int
}

func (e *ResponseError) Error() string { return fmt.Sprintf("权限服务返回 %d", e.StatusCode) }

type Client struct {
	ServerURL string
	HTTP      *http.Client
	Signer    *app.Client
}

func New(serverURL string, signer *app.Client) *Client {
	return &Client{ServerURL: strings.TrimRight(strings.TrimSpace(serverURL), "/"), HTTP: &http.Client{Timeout: 15 * time.Second}, Signer: signer}
}

func (c *Client) call(ctx context.Context, path string, payload interface{}, output interface{}) error {
	if c.ServerURL == "" {
		return errors.New("permission.server_url 未配置")
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.ServerURL+"/zpk/v2/openapi"+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	if err := c.Signer.Sign(request); err != nil {
		return err
	}
	response, err := c.HTTP.Do(request)
	if err != nil {
		return fmt.Errorf("权限服务不可用: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return &ResponseError{StatusCode: response.StatusCode}
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if err != nil {
		return err
	}
	if len(body) > 1<<20 {
		return errors.New("权限服务响应过长")
	}
	var result struct {
		Data  json.RawMessage `json:"data"`
		Error string          `json:"error"`
		Code  int             `json:"code"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return errors.New("权限服务响应无效")
	}
	if result.Error != "" || (result.Code != 0 && result.Code != 200) {
		return errors.New("权限服务请求失败")
	}
	if output != nil {
		if len(result.Data) == 0 {
			return errors.New("权限服务响应缺少数据")
		}
		return json.Unmarshal(result.Data, output)
	}
	var success string
	if err := json.Unmarshal(result.Data, &success); err != nil || success != "success" {
		return errors.New("权限服务未确认设置成功")
	}
	return nil
}
