package logic

import (
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/zpk"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
)

const TokenHeader = "X-Document-Token"
const documentTokenIssuer = "w7panel-document"

type Login struct{}

type loginClaims struct {
	User zpk.User `json:"user"`
	jwt.RegisteredClaims
}

func loginSigningKey() ([]byte, error) {
	if service.AppSdk == nil || service.AppSdk.AppID == "" || service.AppSdk.Secret == "" {
		return nil, errors.New("文档登录签名凭据未初始化")
	}
	key := hmac.New(sha256.New, []byte(service.AppSdk.Secret))
	key.Write([]byte(documentTokenIssuer + ":login:v1"))
	return key.Sum(nil), nil
}

func (Login) SaveUserInfo(ctx *gin.Context, user zpk.User) (string, error) {
	key, err := loginSigningKey()
	if err != nil {
		return "", err
	}
	ttl := facade.GetConfig().GetInt("auth.token_ttl_seconds")
	if ttl <= 0 {
		return "", errors.New("文档 token 有效期未配置")
	}
	now := time.Now()
	claims := loginClaims{User: user, RegisteredClaims: jwt.RegisteredClaims{
		Issuer: documentTokenIssuer, Audience: jwt.ClaimStrings{service.AppSdk.AppID},
		IssuedAt: jwt.NewNumericDate(now), NotBefore: jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(ttl) * time.Second)),
	}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(key)
}

func (Login) GetUserInfo(ctx *gin.Context) (zpk.User, error) {
	key, err := loginSigningKey()
	if err != nil {
		return zpk.User{}, err
	}
	claims := &loginClaims{}
	token, err := jwt.ParseWithClaims(ctx.GetHeader(TokenHeader), claims, func(*jwt.Token) (interface{}, error) {
		return key, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer(documentTokenIssuer),
		jwt.WithAudience(service.AppSdk.AppID), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
	if err != nil || !token.Valid || claims.User.ID <= 0 || claims.IssuedAt == nil || claims.NotBefore == nil {
		return zpk.User{}, zpk.ErrUnauthenticated
	}
	return claims.User, nil
}

func (l Login) FromPanel(ctx *gin.Context, accessToken string) (string, error) {
	principal, err := service.ZPKSdk.CurrentUser(ctx.Request.Context(), accessToken)
	if err != nil {
		return "", err
	}
	return l.SaveUserInfo(ctx, principal)
}
