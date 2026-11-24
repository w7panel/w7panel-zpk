package zpk

import (
	"context"
	"errors"
	"net/http"
)

type Namespace struct {
	ID      int32  `json:"id"`
	Name    string `json:"name"`
	Visible bool   `json:"visible"`
}

func (c *Client) GetNamespace(ctx context.Context, name string) (Namespace, error) {
	var namespace Namespace
	err := c.call(ctx, "", "/v2/openapi/namespace/get", map[string]any{"name": name}, &namespace)
	if err == nil && (namespace.ID <= 0 || namespace.Name != name) {
		err = errors.New("ZPK 命名空间信息无效")
	}
	return namespace, err
}

// EnsureNamespace registers a hidden namespace, preserving an existing namespace's settings.
func (c *Client) EnsureNamespace(ctx context.Context, name string) error {
	_, err := c.GetNamespace(ctx, name)
	if err == nil {
		return nil
	}
	var responseError *ResponseError
	if !errors.As(err, &responseError) || responseError.StatusCode != http.StatusNotFound {
		return err
	}
	return c.call(ctx, "", "/v2/openapi/namespace/create", map[string]any{"name": name, "visible": false, "visible_type": 1}, nil)
}

func (c *Client) RenameNamespace(ctx context.Context, id int32, name string) error {
	return c.call(ctx, "", "/v2/openapi/namespace/rename", map[string]any{"id": id, "name": name}, nil)
}

func (c *Client) DeleteNamespace(ctx context.Context, id int32) error {
	return c.call(ctx, "", "/v2/openapi/namespace/delete", map[string]any{"id": id}, nil)
}
