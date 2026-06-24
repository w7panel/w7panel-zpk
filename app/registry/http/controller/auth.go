package controller

import (
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/app/registry/logic"
	"github.com/w7panel/w7panel-zpk/common/entity"
	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/w7panel/w7panel-zpk/common/types/registry"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/controller"
)

var (
	scopeRegex = regexp.MustCompile(`([a-z0-9]+)(\([a-z0-9]+\))?`)
)

func parseScope(scope string) (string, string, error) {
	parts := scopeRegex.FindStringSubmatch(scope)
	if parts == nil {
		return "", "", fmt.Errorf("malformed scope request")
	}

	switch len(parts) {
	case 3:
		return parts[1], "", nil
	case 4:
		return parts[1], parts[3], nil
	default:
		return "", "", fmt.Errorf("malformed scope request")
	}
}

type Auth struct {
	controller.Abstract
}

func (c Auth) Auth(ctx *gin.Context) {
	scopes, err := c.matchScope(ctx)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	user, err := c.validateUser(ctx)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	err = logic.Permission{}.CheckUserScopes(user, scopes)
	if err != nil {
		c.JsonResponseWithError(ctx, err, 401)
		return
	}

	for _, scope := range scopes {
		if scope.Type == string(logic.ScopeRegistryType) {
			facade.GetEvent().Publish(registry.RegistryRepositoryPrepareEvent, registry.RegistryRepositoryPayLoad{
				User:  user,
				Scope: scope,
			})
		}
	}

	token, err := logic.Permission{}.CreateToken(user, scopes)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	ctx.JSON(200, gin.H{
		"token":        token,
		"access_token": token,
	})
}

func (c Auth) validateUser(ctx *gin.Context) (*entity.RegistryUser, error) {
	username, password, haveBasicAuth := ctx.Request.BasicAuth()
	if haveBasicAuth {

	} else if ctx.Request.Method == "POST" {
		username = ctx.Request.FormValue("username")
		password = ctx.Request.FormValue("password")
	}

	slog.Info("auth req", "username", username)

	account := ctx.Request.FormValue("account")
	if account == "" {
		account = username
	} else if haveBasicAuth && account != username {
		return nil, fmt.Errorf("user and account are not the same (%q vs %q)", username, account)
	}

	if password == "" {
		return nil, nil
	}

	user, err := logic2.User{}.GetByUsername(username)
	if err != nil {
		return nil, err
	}
	if (user.Password != logic2.User{}.MakeUserPassword(password)) {
		return nil, errors.New("username or password error")
	}
	if user.ExpireDays > 0 {
		if time.Now().After(user.CreatedAt.Add(time.Duration(user.ExpireDays) * 24 * time.Hour)) {
			return nil, errors.New("user is expired")
		}
	}

	return user, nil
}

func (c Auth) matchScope(ctx *gin.Context) ([]registry.PermissionScope, error) {
	var scopes = make([]registry.PermissionScope, 0)
	if ctx.Request.FormValue("scope") != "" {
		for _, scopeValue := range ctx.Request.Form["scope"] {
			for _, scopeStr := range strings.Split(scopeValue, " ") {
				parts := strings.Split(scopeStr, ":")
				var scope registry.PermissionScope

				scopeType, scopeClass, err := parseScope(parts[0])
				if err != nil {
					return nil, err
				}

				switch len(parts) {
				case 3:
					scope = registry.PermissionScope{
						Type:    scopeType,
						Class:   scopeClass,
						Name:    parts[1],
						Actions: strings.Split(parts[2], ","),
					}
				case 4:
					scope = registry.PermissionScope{
						Type:    scopeType,
						Class:   scopeClass,
						Name:    parts[1] + ":" + parts[2],
						Actions: strings.Split(parts[3], ","),
					}
				default:
					return nil, fmt.Errorf("invalid scope: %q", scopeStr)
				}
				sort.Strings(scope.Actions)
				scopes = append(scopes, scope)
			}
		}
	}

	return scopes, nil
}
