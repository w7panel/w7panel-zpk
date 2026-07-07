package logic

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"golang.org/x/oauth2"
)

const defaultOIDCScope = "openid profile"
const oidcCallbackPath = "/zpk/system/oidc/callback"

type OIDC struct{}

type OIDCConfig struct {
	BaseURL                 string
	Issuer                  string
	ClientID                string
	ClientSecret            string
	RedirectURI             string
	Scopes                  []string
	TokenEndpointAuthMethod string
	SuccessRedirect         string
	AutoCreateUser          bool
	DefaultRole             string
}

type UserInfo struct {
	Subject    string
	Username   string
	Role       string
	ConsoleUID int32
	Claims     map[string]interface{}
}

func (OIDC) Config() OIDCConfig {
	baseURL := strings.TrimRight(strings.TrimSpace(facade.GetConfig().GetString("system.oidc.base_url")), "/")
	scope := strings.TrimSpace(facade.GetConfig().GetString("system.oidc.scope"))
	if scope == "" {
		scope = defaultOIDCScope
	}
	clientID := strings.TrimSpace(facade.GetConfig().GetString("system.oidc.client_id"))
	if clientID == "" {
		clientID = "default"
	}
	tokenEndpointAuthMethod := strings.TrimSpace(facade.GetConfig().GetString("system.oidc.token_endpoint_auth_method"))
	if tokenEndpointAuthMethod == "" {
		tokenEndpointAuthMethod = "none"
	}
	defaultRole := strings.TrimSpace(facade.GetConfig().GetString("system.oidc.default_role"))
	if defaultRole == "" {
		defaultRole = logic.UserRoleUser
	}
	issuer := strings.TrimRight(strings.TrimSpace(facade.GetConfig().GetString("system.oidc.issuer")), "/")
	if issuer == "" && baseURL != "" {
		issuer = baseURL + "/panel-api/v1/oidc"
	}
	redirectURI := strings.TrimSpace(facade.GetConfig().GetString("system.oidc.redirect_uri"))
	if redirectURI == "" {
		serviceBaseURL := depotExternalBaseURL(facade.GetConfig().GetString("system.depot.external_domain"))
		if serviceBaseURL != "" {
			redirectURI = serviceBaseURL + oidcCallbackPath
		}
	}
	return OIDCConfig{
		BaseURL:                 baseURL,
		Issuer:                  issuer,
		ClientID:                clientID,
		ClientSecret:            facade.GetConfig().GetString("system.oidc.client_secret"),
		RedirectURI:             redirectURI,
		Scopes:                  strings.Fields(scope),
		TokenEndpointAuthMethod: tokenEndpointAuthMethod,
		SuccessRedirect:         facade.GetConfig().GetString("system.oidc.success_redirect"),
		AutoCreateUser:          facade.GetConfig().GetBool("system.oidc.auto_create_user"),
		DefaultRole:             defaultRole,
	}
}

func depotExternalBaseURL(domain string) string {
	domain = strings.TrimRight(strings.TrimSpace(domain), "/")
	if domain == "" {
		return ""
	}
	if strings.HasPrefix(domain, "http://") || strings.HasPrefix(domain, "https://") {
		return domain
	}
	return "https://" + domain
}

func (o OIDC) LoginURL(ctx context.Context, cfg OIDCConfig, redirectURI, state, nonce string) (string, error) {
	provider, err := oidc.NewProvider(ctx, cfg.Issuer)
	if err != nil {
		return "", err
	}
	oauthConfig := o.oauthConfig(provider, cfg, redirectURI)
	return oauthConfig.AuthCodeURL(state, oidc.Nonce(nonce)), nil
}

func (o OIDC) ExchangeCode(ctx context.Context, cfg OIDCConfig, redirectURI, code, nonce string) (*UserInfo, error) {
	provider, err := oidc.NewProvider(ctx, cfg.Issuer)
	if err != nil {
		return nil, err
	}
	oauthConfig := o.oauthConfig(provider, cfg, redirectURI)
	oauthToken, err := oauthConfig.Exchange(ctx, code)
	if err != nil {
		return nil, err
	}
	rawIDToken, ok := oauthToken.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return nil, errors.New("oidc token response missing id_token")
	}
	idToken, err := provider.Verifier(&oidc.Config{ClientID: cfg.ClientID}).Verify(ctx, rawIDToken)
	if err != nil {
		return nil, err
	}
	if nonce != "" && idToken.Nonce != nonce {
		return nil, errors.New("id_token nonce mismatch")
	}
	claims := map[string]interface{}{}
	if err := idToken.Claims(&claims); err != nil {
		return nil, err
	}
	return userInfoFromClaims(claims, cfg.DefaultRole), nil
}

func (o OIDC) oauthConfig(provider *oidc.Provider, cfg OIDCConfig, redirectURI string) oauth2.Config {
	endpoint := provider.Endpoint()
	switch cfg.TokenEndpointAuthMethod {
	case "client_secret_basic":
		endpoint.AuthStyle = oauth2.AuthStyleInHeader
	case "client_secret_post":
		endpoint.AuthStyle = oauth2.AuthStyleInParams
	case "none":
		endpoint.AuthStyle = oauth2.AuthStyleInParams
		cfg.ClientSecret = ""
	}
	return oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     endpoint,
		RedirectURL:  redirectURI,
		Scopes:       cfg.Scopes,
	}
}

func userInfoFromClaims(claims map[string]interface{}, defaultRole string) *UserInfo {
	sub, _ := claims["sub"].(string)
	username, _ := claims["preferred_username"].(string)
	if username == "" {
		username, _ = claims["username"].(string)
	}
	if username == "" {
		username, _ = claims["name"].(string)
	}
	if username == "" {
		username = sub
	}
	role, _ := claims["role"].(string)
	if isFounder, ok := claims["is_founder"].(bool); ok && isFounder {
		role = logic.UserRoleFounder
	}
	if role == "" {
		role = defaultRole
	}
	consoleUID := int32(0)
	if cloudUID, _ := claims["cloud_uid"].(string); cloudUID != "" {
		if value, err := strconv.Atoi(cloudUID); err == nil {
			consoleUID = int32(value)
		}
	}
	return &UserInfo{
		Subject:    sub,
		Username:   username,
		Role:       role,
		ConsoleUID: consoleUID,
		Claims:     claims,
	}
}
