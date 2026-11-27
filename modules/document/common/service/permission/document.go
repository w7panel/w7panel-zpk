package permission

import (
	"context"
	"errors"
	"fmt"

	"github.com/w7panel/w7panel-zpk/modules/document/common/entity"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/zpk"
)

const (
	VisibleTypePrivate    = 1 // 不公开，由用户授权决定。
	VisibleTypePublicRead = 4 // 匿名可读取。
)

var ErrDenied = errors.New("无操作权限")

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

func LibraryResource(library entity.DocumentLibrary) Resource {
	return Resource{Type: "document_library", Value: library.Category + "/" + library.Identifie}
}

func (c *Client) Check(ctx context.Context, userID int32, resource Resource, action string) (bool, error) {
	var decision *Decision
	err := c.call(ctx, "/resource/check", struct {
		UserID   int32    `json:"user_id"`
		Resource Resource `json:"resource"`
		Action   string   `json:"action"`
	}{userID, resource, action}, &decision)
	if err != nil {
		return false, err
	}
	if decision == nil {
		return false, errors.New("权限服务响应无效")
	}
	return decision.Allowed, nil
}

// SetResource creates or updates a resource without replacing user grants.
func (c *Client) SetResource(ctx context.Context, resource Resource, userID, visibleType int32) error {
	return c.call(ctx, "/resource/set", map[string]any{"resource": resource, "user_id": userID, "visible_type": visibleType}, nil)
}

func (c *Client) AddOwner(ctx context.Context, library entity.DocumentLibrary) error {
	return c.call(ctx, "/permission/add", Grant{UserID: library.UserID, Resource: LibraryResource(library), Actions: []string{"push", "pull"}}, nil)
}

func (c *Client) ClearLibrary(ctx context.Context, library entity.DocumentLibrary) error {
	return c.call(ctx, "/resource/delete", map[string]any{"resource": LibraryResource(library)}, nil)
}

func (c *Client) CanRead(ctx context.Context, user zpk.User, library entity.DocumentLibrary) (bool, error) {
	return c.Check(ctx, user.ID, LibraryResource(library), "pull")
}

func (c *Client) CanManage(ctx context.Context, user zpk.User, library entity.DocumentLibrary) (bool, error) {
	return c.Check(ctx, user.ID, LibraryResource(library), "push")
}

func Require(allowed bool, err error) error {
	if err != nil {
		return fmt.Errorf("权限验证失败: %w", err)
	}
	if !allowed {
		return ErrDenied
	}
	return nil
}
