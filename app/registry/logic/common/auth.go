package common

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

type PermissionScopeType string

const (
	ScopeRegistryType PermissionScopeType = "repository"
)

var PermissionPrivateKey libtrust.PrivateKey
var PermissionPublicKey libtrust.PublicKey

type Auth struct {
	logic.Logic
}

// CheckScopes 按原有镜像规则准备权限，每个 scope 只调用一次通用校验。
func (l Auth) CheckScopes(ctx context.Context, user *entity.RegistryUser, scopes []registry.PermissionScope) error {
	if user != nil && user.Type == logic.UserTypeSuperAdminForRegistry {
		return nil
	}
	for _, scope := range scopes {
		resource, err := l.prepareScopeResource(ctx, scope)
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

func (l Auth) prepareScopeResource(ctx context.Context, scope registry.PermissionScope) (ResourcePermissionResource, error) {
	resource, err := (Resource{}).Prepare(ctx, ResourcePermissionResource{Type: scope.Type, Value: scope.Name})
	if err != nil {
		return resource, err
	}
	publicActions := resource.PublicActions
	resource.PublicActions = nil
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

func (l Auth) CreateToken(user *entity.RegistryUser, scopes []registry.PermissionScope) (string, error) {
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

// VerifyToken validates the signed token and exact resource type, path and actions.
func (l Auth) VerifyToken(raw string, scope registry.PermissionScope) error {
	if PermissionPublicKey == nil {
		return errors.New("registry signing key is unavailable")
	}
	parsed, err := token.NewToken(raw)
	if err != nil {
		return err
	}
	if err := parsed.Verify(token.VerifyOptions{
		TrustedIssuers:    []string{facade.GetConfig().GetString("registry_cli.default.oauth.issuer")},
		AcceptedAudiences: []string{"docker-registry"},
		TrustedKeys:       map[string]libtrust.PublicKey{PermissionPublicKey.KeyID(): PermissionPublicKey},
	}); err != nil {
		return err
	}
	for _, access := range parsed.Claims.Access {
		if access.Type == scope.Type && access.Name == scope.Name && ValidatePermissionActions(scope.Actions, access.Actions) {
			return nil
		}
	}
	return errors.New("insufficient repository scope")
}
