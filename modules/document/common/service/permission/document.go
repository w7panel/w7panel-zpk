package permission

import (
	"context"
	"errors"
	"fmt"

	"github.com/w7panel/w7panel-zpk/modules/document/common/entity"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/zpk"
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
type Record struct {
	ID            int32    `json:"id"`
	UserID        int32    `json:"user_id"`
	ResourceType  string   `json:"resource_type"`
	ResourceValue string   `json:"resource_value"`
	Action        []string `json:"action"`
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
	err := c.call(ctx, "/check", struct {
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

func (c *Client) Get(ctx context.Context, resource Resource) ([]Record, error) {
	var records []Record
	err := c.call(ctx, "/get", struct {
		Resource Resource `json:"resource"`
	}{resource}, &records)
	return records, err
}

func (c *Client) Set(ctx context.Context, grants []Grant) error {
	return c.call(ctx, "/set", struct {
		Grants []Grant `json:"grants"`
	}{grants}, nil)
}

func (c *Client) SetLibrary(ctx context.Context, library entity.DocumentLibrary) error {
	resource := LibraryResource(library)
	grants := []Grant{{UserID: library.UserID, Resource: resource, Actions: []string{"push", "pull"}}}
	if library.Public == 1 {
		grants = append(grants, Grant{UserID: 0, Resource: resource, Actions: []string{"pull"}})
	}
	return c.Set(ctx, grants)
}

func (c *Client) ClearLibrary(ctx context.Context, library entity.DocumentLibrary) error {
	return c.Set(ctx, []Grant{{UserID: library.UserID, Resource: LibraryResource(library), Actions: []string{}}})
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
