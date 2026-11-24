// Package document is the ZPK caller's SDK. The document module never imports this package.
package document

import (
	"bytes"
	"context"
	"encoding/json"
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
	return fmt.Sprintf("文档服务返回 %d: %s", e.StatusCode, e.Message)
}

func New(baseURL, appID, secret string) *Client {
	return &Client{
		Base: base.Base{BaseUrl: strings.TrimRight(baseURL, "/"), Appid: appID, Secret: secret},
		HTTP: &http.Client{Timeout: 30 * time.Second},
	}
}

type User struct {
	ID    int32 `json:"id"`
	Admin bool  `json:"admin"`
}

func (c *Client) CurrentUser(ctx context.Context, token string) (User, error) {
	var user User
	err := c.call(ctx, token, "/user", struct{}{}, &user)
	return user, err
}

func (c *Client) call(ctx context.Context, token, route string, body any, out any) error {
	if strings.TrimSpace(token) == "" {
		return &ResponseError{StatusCode: http.StatusUnauthorized, Message: "token 错误，请通过面板重新登录"}
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseUrl+"/zpk/document/openapi"+route, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-W7Panel-Access-Token", token)
	if err := c.Base.Sign(req); err != nil {
		return err
	}
	response, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("文档服务不可用: %w", err)
	}
	defer response.Body.Close()
	var result struct {
		Data  json.RawMessage `json:"data"`
		Error string          `json:"error"`
	}
	if err = json.NewDecoder(io.LimitReader(response.Body, 8<<20)).Decode(&result); err != nil {
		return err
	}
	if response.StatusCode != http.StatusOK {
		return &ResponseError{StatusCode: response.StatusCode, Message: result.Error}
	}
	if out != nil {
		return json.Unmarshal(result.Data, out)
	}
	return nil
}

type CreateLibrary struct {
	Ref         string        `json:"ref"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	OwnerID     int32         `json:"owner_id"`
	Public      int32         `json:"public"`
	Protected   int32         `json:"protected"`
	Files       []FileContent `json:"files"`
}

type Library struct {
	UserID    int32 `json:"user_id"`
	Protected int32 `json:"protected"`
}

func (c *Client) Create(ctx context.Context, token string, params CreateLibrary) error {
	return c.call(ctx, token, "/library/create", params, nil)
}

func (c *Client) Info(ctx context.Context, token, ref string) (Library, error) {
	var library Library
	err := c.call(ctx, token, "/library/info", map[string]string{"ref": ref}, &library)
	return library, err
}

func (c *Client) Save(ctx context.Context, token, ref, path, title, content string) error {
	return c.call(ctx, token, "/file/save", map[string]any{"ref": ref, "path": path, "title": title, "content": content}, nil)
}

type File struct {
	Path      string `json:"path"`
	Title     string `json:"title"`
	Protected int32  `json:"protected"`
}

type FileContent struct {
	File
	Content string `json:"content"`
}

type FileTree struct {
	Files []File `json:"files"`
}

func (c *Client) Tree(ctx context.Context, token, reference string) (FileTree, error) {
	var result FileTree
	err := c.call(ctx, token, "/file/list", map[string]interface{}{"ref": reference}, &result)
	return result, err
}

func (c *Client) Order(ctx context.Context, token, reference string, paths []string) error {
	return c.call(ctx, token, "/file/order", map[string]interface{}{"ref": reference, "paths": paths}, nil)
}

func (c *Client) Read(ctx context.Context, token, reference, path string) (FileContent, error) {
	var result FileContent
	err := c.call(ctx, token, "/file/read", map[string]any{"ref": reference, "path": path}, &result)
	return result, err
}

func (c *Client) Delete(ctx context.Context, token, reference, path string) error {
	return c.call(ctx, token, "/file/delete", map[string]any{"ref": reference, "path": path}, nil)
}
