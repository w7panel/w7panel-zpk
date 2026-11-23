// Package zpk provides the document system's client for the shared ZPK user API.
package zpk

import (
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

func New(baseURL string) *Client {
	return &Client{BaseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"), HTTP: &http.Client{Timeout: 15 * time.Second}}
}

func (c *Client) CurrentUser(ctx context.Context, accessToken string) (User, error) {
	if c.BaseURL == "" {
		return User{}, errors.New("zpk.base_url 未配置")
	}
	if strings.TrimSpace(accessToken) == "" {
		return User{}, ErrUnauthenticated
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v2/api/user/current", nil)
	if err != nil {
		return User{}, err
	}
	req.Header.Set("X-W7Panel-Access-Token", accessToken)
	req.Header.Set("Accept", "application/json")
	response, err := c.HTTP.Do(req)
	if err != nil {
		return User{}, fmt.Errorf("ZPK 用户服务不可用: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode == 401 || response.StatusCode == 403 {
		return User{}, ErrUnauthenticated
	}
	if response.StatusCode != http.StatusOK {
		return User{}, fmt.Errorf("ZPK 用户服务返回 %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if err != nil {
		return User{}, err
	}
	if len(body) > 1<<20 {
		return User{}, errors.New("ZPK 用户服务响应过长")
	}
	var result struct {
		Data  User   `json:"data"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return User{}, errors.New("ZPK 用户服务响应无效")
	}
	if result.Error != "" || result.Data.ID <= 0 || result.Data.Username == "" || result.Data.Role == "" {
		return User{}, errors.New("ZPK 用户信息无效")
	}
	return result.Data, nil
}
