package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/docker/distribution/registry/auth/token"
	"github.com/docker/libtrust"
	"github.com/w7panel/w7panel-zpk/common/entity"
	"github.com/w7panel/w7panel-zpk/common/function"
	"github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/w7panel/w7panel-zpk/common/types/registry"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
)

const (
	VisibleTypePrivate         = 1
	VisibleTypePublicWrite     = 2
	VisibleTypePublicRead      = 4
	VisibleTypeFollowNamespace = 3
)

type PermissionScopeType string

const (
	ScopeRegistryType PermissionScopeType = "repository"
)

var PermissionPrivateKey libtrust.PrivateKey
var PermissionPublicKey libtrust.PublicKey

type RegistryPermission struct {
	logic.Logic
}

// CheckRegistryScopes 按原有镜像规则准备权限，每个 scope 只调用一次通用校验。
func (l RegistryPermission) CheckRegistryScopes(ctx context.Context, user *entity.RegistryUser, scopes []registry.PermissionScope) error {
	if user != nil && user.Type == logic.UserTypeSuperAdminForRegistry {
		return nil
	}
	for _, scope := range scopes {
		if scope.Type != string(ScopeRegistryType) {
			continue
		}
		resource, err := l.prepareScopeResource(scope)
		if err != nil {
			return err
		}
		decision, err := (ResourcePermission{}).CheckResource(ctx, user, resource, scope.Actions)
		if err != nil {
			return err
		}
		if !decision.Allowed {
			if user == nil {
				return errors.New("user not found")
			}
			return errors.New("invalid action")
		}
	}
	return nil
}

func (l RegistryPermission) prepareScopeResource(scope registry.PermissionScope) (ResourcePermissionResource, error) {
	repositoryName, namespace := logic.ParseRepositoryNameAndNamespace(scope.Name)
	resource := ResourcePermissionResource{Type: string(PermissionResourceTypeRepository), Value: scope.Name}
	repository, _ := (Repository{}).GetByNameAndNamespace(repositoryName, namespace)
	visibleType := VisibleTypeFollowNamespace
	if repository != nil {
		visibleType = int(repository.VisibleType)
	}
	if visibleType == VisibleTypeFollowNamespace {
		namespaceModel, _ := (Namespace{}).GetByName(namespace)
		if namespaceModel == nil {
			return resource, errors.New("namespace not found")
		}
		visibleType = int(namespaceModel.VisibleType)
	}
	publicActions := registryPublicActions(visibleType)
	publicAction := ""
	if slices.Contains(scope.Actions, string(PermissionActionTypePush)) {
		publicAction = string(PermissionActionTypePush)
	} else if slices.Contains(scope.Actions, string(PermissionActionTypePull)) {
		publicAction = string(PermissionActionTypePull)
	}
	if publicAction != "" && slices.Contains(publicActions, publicAction) {
		// 原镜像规则按 push、pull 优先级放行整个 scope，保留其混合动作行为。
		resource.PublicActions = scope.Actions
	}
	return resource, nil
}

func registryPublicActions(visibleType int) []string {
	switch visibleType {
	case VisibleTypePublicRead:
		return []string{string(PermissionActionTypePull)}
	case VisibleTypePublicWrite:
		return []string{string(PermissionActionTypePush), string(PermissionActionTypePull)}
	default:
		return nil
	}
}

func (l RegistryPermission) CreateToken(user *entity.RegistryUser, scopes []registry.PermissionScope) (string, error) {
	userName := ""
	if user != nil {
		userName = user.Username
	}
	now := time.Now()
	header := token.Header{
		Type:       "JWT",
		SigningAlg: "RS256",
		KeyID:      PermissionPublicKey.KeyID(),
	}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("failed to marshal header: %s", err)
	}

	claims := token.ClaimSet{
		Issuer:     facade.GetConfig().GetString("registry_cli.default.oauth.issuer"),
		Subject:    userName,
		Audience:   "docker-registry",
		NotBefore:  now.Unix() - 20,
		Expiration: now.Add(7 * 24 * time.Hour).Unix(),
		Access:     []*token.ResourceActions{},
	}
	for _, a := range scopes {
		ra := &token.ResourceActions{
			Type:    a.Type,
			Name:    a.Name,
			Actions: a.Actions,
		}
		if ra.Actions == nil {
			ra.Actions = []string{}
		}
		sort.Strings(ra.Actions)
		claims.Access = append(claims.Access, ra)
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("failed to marshal claims: %s", err)
	}

	payload := fmt.Sprintf("%s%s%s", function.JoseBase64UrlEncode(headerJSON), token.TokenSeparator, function.JoseBase64UrlEncode(claimsJSON))

	sig, _, err := PermissionPrivateKey.Sign(strings.NewReader(payload), 0)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%s%s", payload, token.TokenSeparator, function.JoseBase64UrlEncode(sig)), nil
}

func (l RegistryPermission) OnAddUserPermissionEvent(payload registry.AddUserPermissionPayload) {
	err := (ResourcePermission{}).AddUserPermission(payload.UserID, payload.ResourceValue, PermissionResourceType(payload.ResourceType), payload.Actions)
	slog.Info("OnAddUserPermissionEvent", "payload", payload, "err", err)
	if err != nil {
		return
	}
}

func (l RegistryPermission) OnDelUserPermissionEvent(payload registry.DelUserPermissionPayload) {
	err := (ResourcePermission{}).DelUserPermission(payload.UserID, payload.ResourceValue, PermissionResourceType(payload.ResourceType))
	slog.Info("OnDelUserPermissionEvent", "payload", payload, "err", err)
	if err != nil {
		return
	}

}

func (l RegistryPermission) OnClearUserPermissionEvent(payload registry.ClearUserPermissionPayload) {
	err := (ResourcePermission{}).ClearUserPermission(payload.UserID)
	slog.Info("onClearUserPermissionEvent", "payload", payload, "err", err)
	if err != nil {
		return
	}
}
