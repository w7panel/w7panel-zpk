package permission

import (
	"context"
	"errors"
	"net/http"
)

type Namespace struct {
	ID   int32  `json:"id"`
	Name string `json:"name"`
}

func (c *Client) GetNamespace(ctx context.Context, name string) (Namespace, error) {
	var namespace Namespace
	err := c.call(ctx, "/namespace/get", map[string]any{"name": name}, &namespace)
	if err == nil && (namespace.ID <= 0 || namespace.Name != name) {
		err = errors.New("权限服务命名空间信息无效")
	}
	return namespace, err
}

// EnsureNamespace registers a private namespace, preserving an existing namespace's settings.
func (c *Client) EnsureNamespace(ctx context.Context, name string) error {
	_, err := c.GetNamespace(ctx, name)
	if err == nil {
		return nil
	}
	var responseError *ResponseError
	if !errors.As(err, &responseError) || responseError.StatusCode != http.StatusNotFound {
		return err
	}
	return c.call(ctx, "/namespace/create", map[string]any{"name": name, "visible_type": 1}, nil)
}
