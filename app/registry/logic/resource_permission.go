package logic

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"

	"github.com/w7panel/w7panel-zpk/common/accessor"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	userlogic "github.com/w7panel/w7panel-zpk/common/logic"
)

type ResourcePermissionResource struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type ResourcePermissionGrant struct {
	UserID   int32                      `json:"user_id"`
	Resource ResourcePermissionResource `json:"resource"`
	Actions  []string                   `json:"actions"`
}

type ResourcePermissionDecision struct {
	Allowed  bool `json:"allowed"`
	Explicit bool `json:"explicit"`
}

type ResourcePermission struct{}

var resourcePermissionMu sync.Mutex
var resourcePermissionActions = []string{"read", "write", "create", "delete", "manage_permissions", "push", "pull"}

func validResourcePermissionResource(resource ResourcePermissionResource) bool {
	return slices.Contains([]string{"namespace", "repository", "document_category", "document_library", "document_file"}, resource.Type) && resource.Value != "" && len(resource.Value) <= 512 && !strings.ContainsAny(resource.Value, "\x00\r\n")
}

func (l ResourcePermission) Check(ctx context.Context, user *entity.RegistryUser, resources []ResourcePermissionResource, action string) (ResourcePermissionDecision, error) {
	if user == nil || !slices.Contains(resourcePermissionActions, action) || len(resources) == 0 || len(resources) > 3 {
		return ResourcePermissionDecision{}, errors.New("无效权限请求")
	}
	for _, resource := range resources {
		if !validResourcePermissionResource(resource) {
			return ResourcePermissionDecision{}, errors.New("无效资源类型或标识")
		}
	}
	if (userlogic.User{}).IsAdminUser(user) {
		return ResourcePermissionDecision{Allowed: true}, nil
	}
	query := dao.Q.RegistryUserPermission
	for _, resource := range resources {
		permissions, err := query.WithContext(ctx).Where(query.ResourceType.Eq(resource.Type), query.ResourceValue.Eq(resource.Value)).Find()
		if err != nil {
			return ResourcePermissionDecision{}, err
		}
		if len(permissions) == 0 {
			continue
		}
		for _, item := range permissions {
			if (item.UserID == user.ID || item.UserID == 0) && item.Action != nil && slices.Contains(*item.Action, action) {
				return ResourcePermissionDecision{Allowed: true, Explicit: true}, nil
			}
		}
		return ResourcePermissionDecision{Explicit: true}, nil
	}
	return ResourcePermissionDecision{}, nil
}

func (l ResourcePermission) Replace(ctx context.Context, grants []ResourcePermissionGrant) error {
	if len(grants) == 0 || len(grants) > 500 {
		return errors.New("无效授权请求")
	}
	for _, grant := range grants {
		if grant.UserID < 0 || !validResourcePermissionResource(grant.Resource) {
			return errors.New("无效授权资源")
		}
		for _, action := range grant.Actions {
			if !slices.Contains(resourcePermissionActions, action) {
				return errors.New("无效授权动作")
			}
		}
	}
	resourcePermissionMu.Lock()
	defer resourcePermissionMu.Unlock()
	return dao.Q.Transaction(func(tx *dao.Query) error {
		query := tx.RegistryUserPermission
		for _, grant := range grants {
			if _, err := query.WithContext(ctx).Where(query.UserID.Eq(grant.UserID), query.ResourceType.Eq(grant.Resource.Type), query.ResourceValue.Eq(grant.Resource.Value)).Delete(); err != nil {
				return err
			}
			actions := accessor.PermissionActionOption(grant.Actions)
			if err := query.WithContext(ctx).Create(&entity.RegistryUserPermission{UserID: grant.UserID, ResourceType: grant.Resource.Type, ResourceValue: grant.Resource.Value, Action: &actions}); err != nil {
				return err
			}
		}
		return nil
	})
}

// Initialize replaces all grants on newly allocated resource identities, clearing grants left by deleted files.
func (l ResourcePermission) Initialize(ctx context.Context, grants []ResourcePermissionGrant) error {
	if len(grants) == 0 || len(grants) > 500 {
		return errors.New("无效授权请求")
	}
	for _, grant := range grants {
		if !validResourcePermissionResource(grant.Resource) || grant.UserID < 0 {
			return errors.New("无效资源")
		}
		for _, action := range grant.Actions {
			if !slices.Contains(resourcePermissionActions, action) {
				return errors.New("无效操作")
			}
		}
	}
	resourcePermissionMu.Lock()
	defer resourcePermissionMu.Unlock()
	return dao.Q.Transaction(func(tx *dao.Query) error {
		query := tx.RegistryUserPermission
		cleared := map[ResourcePermissionResource]bool{}
		for _, grant := range grants {
			if !cleared[grant.Resource] {
				if _, err := query.WithContext(ctx).Where(query.ResourceType.Eq(grant.Resource.Type), query.ResourceValue.Eq(grant.Resource.Value)).Delete(); err != nil {
					return err
				}
				cleared[grant.Resource] = true
			}
			actions := accessor.PermissionActionOption(grant.Actions)
			if err := query.WithContext(ctx).Create(&entity.RegistryUserPermission{UserID: grant.UserID, ResourceType: grant.Resource.Type, ResourceValue: grant.Resource.Value, Action: &actions}); err != nil {
				return err
			}
		}
		return nil
	})
}

type ResourcePermissionRequest struct {
	Resources []ResourcePermissionResource `json:"resources"`
	Action    string                       `json:"action"`
}

// CheckBatch loads the grants once, so listing a library does not issue a query per file/action.
func (l ResourcePermission) CheckBatch(ctx context.Context, user *entity.RegistryUser, requests []ResourcePermissionRequest) ([]ResourcePermissionDecision, error) {
	if user == nil || len(requests) == 0 || len(requests) > 500 {
		return nil, errors.New("无效批量权限请求")
	}
	types := []string{}
	values := []string{}
	seenTypes := map[string]bool{}
	seenValues := map[string]bool{}
	for _, request := range requests {
		if !slices.Contains(resourcePermissionActions, request.Action) || len(request.Resources) == 0 || len(request.Resources) > 3 {
			return nil, errors.New("无效权限动作或资源")
		}
		for _, resource := range request.Resources {
			if !validResourcePermissionResource(resource) {
				return nil, errors.New("无效权限资源")
			}
			if !seenTypes[resource.Type] {
				types = append(types, resource.Type)
				seenTypes[resource.Type] = true
			}
			if !seenValues[resource.Value] {
				values = append(values, resource.Value)
				seenValues[resource.Value] = true
			}
		}
	}
	decisions := make([]ResourcePermissionDecision, len(requests))
	if (userlogic.User{}).IsAdminUser(user) {
		for index := range decisions {
			decisions[index].Allowed = true
		}
		return decisions, nil
	}
	query := dao.Q.RegistryUserPermission
	rows, err := query.WithContext(ctx).Where(query.ResourceType.In(types...), query.ResourceValue.In(values...)).Find()
	if err != nil {
		return nil, err
	}
	grants := map[ResourcePermissionResource][]*entity.RegistryUserPermission{}
	for _, row := range rows {
		resource := ResourcePermissionResource{Type: row.ResourceType, Value: row.ResourceValue}
		grants[resource] = append(grants[resource], row)
	}
	for index, request := range requests {
		for _, resource := range request.Resources {
			entries := grants[resource]
			if len(entries) == 0 {
				continue
			}
			decisions[index].Explicit = true
			for _, entry := range entries {
				if (entry.UserID == user.ID || entry.UserID == 0) && entry.Action != nil && slices.Contains(*entry.Action, request.Action) {
					decisions[index].Allowed = true
					break
				}
			}
			break
		}
	}
	return decisions, nil
}
