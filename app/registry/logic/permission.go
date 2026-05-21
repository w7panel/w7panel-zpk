package logic

import (
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
	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/app/registry/types"
	"github.com/w7panel/w7panel-zpk/common/accessor"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	"github.com/w7panel/w7panel-zpk/common/function"
	"github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
)

type PermissionResourceType string

var (
	PermissionResourceTypeNamespace  PermissionResourceType = "namespace"
	PermissionResourceTypeRepository PermissionResourceType = "repository"
)

type PermissionScopeType string

var (
	ScopeRegistryType PermissionScopeType = "repository"
)

type PermissionActionType string

var (
	PermissionActionTypePush PermissionActionType = "push"
	PermissionActionTypePull PermissionActionType = "pull"
)

var PermissionPrivateKey libtrust.PrivateKey
var PermissionPublicKey libtrust.PublicKey

type Permission struct {
	logic.Logic
}

func (l Permission) CheckUserScopes(user *entity.RegistryUser, scopes []types.PermissionScope) error {
	slog.Info("auth", "user", user, "scopes", scopes)

	if user != nil && user.Type == logic.UserTypeSuperAdminForRegistry {
		return nil
	}

	validateAction := func(reqAction []string, permissionAction []string) bool {
		for _, action := range reqAction {
			if !slices.Contains(permissionAction, action) {
				return false
			}
		}

		return true
	}

	for _, item := range scopes {
		if item.Type == string(ScopeRegistryType) {
			repositoryName, namespace := Repository{}.ParseRepositoryNameAndNamespace(item.Name)
			//直接验证仓库是不是公有
			registryModel, _ := Repository{}.GetByNameAndNamespace(repositoryName, namespace)
			if registryModel != nil {
				if slices.Contains(item.Actions, string(PermissionActionTypePush)) {
					if int(registryModel.VisibleType) == VisibleTypePublicWrite {
						continue
					}
				} else if slices.Contains(item.Actions, string(PermissionActionTypePull)) {
					if int(registryModel.VisibleType) == VisibleTypePublicRead || int(registryModel.VisibleType) == VisibleTypePublicWrite {
						continue
					}
				}
			}
			//再验证对应的namespace是不是公有
			if registryModel == nil || (int(registryModel.VisibleType) == VisibleTypeFollowNamespace) {
				namespaceModel, _ := Namespace{}.GetByName(namespace)
				if namespaceModel == nil {
					return errors.New("namespace not found")
				}
				if slices.Contains(item.Actions, string(PermissionActionTypePush)) {
					if int(namespaceModel.VisibleType) == VisibleTypePublicWrite {
						continue
					}
				} else if slices.Contains(item.Actions, string(PermissionActionTypePull)) {
					if int(namespaceModel.VisibleType) == VisibleTypePublicRead || int(namespaceModel.VisibleType) == VisibleTypePublicWrite {
						continue
					}
				}
			}

			if user == nil {
				return errors.New("user not found")
			}

			//再验证有没对应的权限
			registryPermission, _ := dao.Q.RegistryUserPermission.
				Where(dao.Q.RegistryUserPermission.UserID.Eq(user.ID)).
				Where(dao.Q.RegistryUserPermission.ResourceType.Eq(string(PermissionResourceTypeRepository))).
				Where(dao.Q.RegistryUserPermission.ResourceValue.Eq(item.Name)).First()
			if registryPermission != nil {
				if !validateAction(item.Actions, *registryPermission.Action) {
					return errors.New("invalid action")
				}
				continue
			}
			namespacePermission, _ := dao.Q.RegistryUserPermission.
				Where(dao.Q.RegistryUserPermission.UserID.Eq(user.ID)).
				Where(dao.Q.RegistryUserPermission.ResourceType.Eq(string(PermissionResourceTypeNamespace))).
				Where(dao.Q.RegistryUserPermission.ResourceValue.Eq(namespace)).First()
			if namespacePermission != nil {
				if !validateAction(item.Actions, *namespacePermission.Action) {
					return errors.New("invalid action")
				}
				continue
			}

			return errors.New("invalid action")
		} else {
			//暂不处理
		}
	}
	return nil
}

func (l Permission) CreateToken(user *entity.RegistryUser, scopes []types.PermissionScope) (string, error) {
	userName := ""
	expireDays := 999
	if user != nil {
		if user.ExpireDays <= 0 {
			expireDays = 999
		}
		userName = user.Username
	}
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
		NotBefore:  time.Now().Unix() - 20,
		Expiration: time.Now().Add(time.Duration(expireDays) * time.Second * 86400).Unix(),
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

func (l Permission) UserHasPermissionByResource(userId int, resourceValue string, resourceType PermissionResourceType) bool {
	exists, _ := dao.Q.RegistryUserPermission.
		Where(dao.Q.RegistryUserPermission.UserID.Eq(int32(userId))).
		Where(dao.Q.RegistryUserPermission.ResourceType.Eq(string(resourceType))).
		Where(dao.Q.RegistryUserPermission.ResourceValue.Eq(resourceValue)).First()

	return exists != nil
}

func (l Permission) DelUserPermission(tx *dao.Query, userId int) error {
	_, err := tx.RegistryUserPermission.
		Where(tx.RegistryUserPermission.UserID.Eq(int32(userId))).Delete()
	return err
}

func (l Permission) DelPermissionByResource(tx *dao.Query, resourceValue string, resourceType PermissionResourceType) error {
	_, err := tx.RegistryUserPermission.
		Where(tx.RegistryUserPermission.ResourceType.Eq(string(resourceType))).
		Where(tx.RegistryUserPermission.ResourceValue.Eq(resourceValue)).Delete()
	return err
}

func (l Permission) GetNamespacePermissions(namespace string) ([]*entity.RegistryUserPermission, error) {
	return dao.Q.RegistryUserPermission.
		Where(dao.Q.RegistryUserPermission.ResourceType.Eq(string(PermissionResourceTypeNamespace))).
		Where(dao.Q.RegistryUserPermission.ResourceValue.Eq(namespace)).
		Find()
}

func (l Permission) ReplaceNamespacePermissions(namespace string, permissions []types.NamespacePermissionItem) error {
	return dao.Q.Transaction(func(tx *dao.Query) error {
		_, err := tx.RegistryUserPermission.
			Where(tx.RegistryUserPermission.ResourceType.Eq(string(PermissionResourceTypeNamespace))).
			Where(tx.RegistryUserPermission.ResourceValue.Eq(namespace)).
			Delete()
		if err != nil {
			return err
		}

		if len(permissions) == 0 {
			return nil
		}

		userPermissions := make([]*entity.RegistryUserPermission, 0, len(permissions))
		for _, item := range permissions {
			userPermissions = append(userPermissions, &entity.RegistryUserPermission{
				UserID:        int32(item.UserID),
				ResourceValue: namespace,
				ResourceType:  string(PermissionResourceTypeNamespace),
				Action:        (*accessor.PermissionActionOption)(&item.Actions),
			})
		}

		return tx.RegistryUserPermission.CreateInBatches(userPermissions, 10)
	})
}

func (l Permission) IsCanOperate(ctx *gin.Context, curUid int32) bool {
	user := logic.User{}.GetUser(ctx)
	if user == nil || (logic.User{}.IsAdminUser(user)) || curUid == user.ID {
		return true
	}
	return false
}
