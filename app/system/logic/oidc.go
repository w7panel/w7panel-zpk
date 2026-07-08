package logic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
)

const oidcUserInfoPath = "/panel-api/v1/oidc/userinfo"

type OIDC struct{}

type UserInfo struct {
	Subject    string
	Username   string
	Role       string
	ConsoleUID int32
	Claims     map[string]interface{}
}

func (OIDC) UserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(facade.GetConfig().GetString("system.oidc.base_url")), "/")
	if baseURL == "" {
		return nil, errors.New("system.oidc.base_url is required")
	}
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return nil, errors.New("access_token is required")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+oidcUserInfoPath, nil)
	if err != nil {
		return nil, err
	}
	authorization := accessToken
	if !strings.HasPrefix(strings.ToLower(authorization), "bearer ") {
		authorization = "Bearer " + authorization
	}
	req.Header.Set("Authorization", authorization)
	req.Header.Set("Accept", "application/json")

	oidcHTTPClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := oidcHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("userinfo request failed with status %d", resp.StatusCode)
	}

	claims := map[string]interface{}{}
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(&claims); err != nil {
		return nil, err
	}
	return userInfoFromClaims(claims)
}

func userInfoFromClaims(claims map[string]interface{}) (*UserInfo, error) {
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
		return nil, errors.New("userinfo role is required")
	}
	consoleUID := int32(0)
	if cloudUID, _ := claims["cloud_uid"].(string); cloudUID != "" {
		if value, err := strconv.ParseInt(cloudUID, 10, 32); err == nil {
			consoleUID = int32(value)
		}
	}
	return &UserInfo{
		Subject:    sub,
		Username:   username,
		Role:       role,
		ConsoleUID: consoleUID,
		Claims:     claims,
	}, nil
}
