package logic

import (
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/w7panel/w7panel-zpk/common/service/w7"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
)

const ZpkTokenHeader = "X-Zpk-Token"
const zpkTokenIssuer = "w7panel-zpk"

var ErrInvalidToken = errors.New("token 错误")

type Login struct{}

type LoginUser struct {
	UserID     int32  `json:"user_id"`
	ConsoleUid int32  `json:"console_uid"`
	Username   string `json:"username"`
}

type loginClaims struct {
	LoginUser
	jwt.RegisteredClaims
}

// PanelAccessToken is supplied by the panel, never stored in the login JWT.
func (Login) PanelAccessToken(ctx *gin.Context) string {
	return ctx.GetHeader("X-W7Panel-Access-Token")
}

func loginSigningKey() ([]byte, error) {
	if w7.AppSdk == nil || w7.AppSdk.Appid == "" || w7.AppSdk.Secret == "" {
		return nil, errors.New("登录签名凭据未初始化")
	}
	key := hmac.New(sha256.New, []byte(w7.AppSdk.Secret))
	key.Write([]byte(zpkTokenIssuer + ":login:v1"))
	return key.Sum(nil), nil
}

func (Login) SaveUserInfo(ctx *gin.Context, user LoginUser) (string, error) {
	key, err := loginSigningKey()
	if err != nil {
		return "", err
	}
	ttl := facade.GetConfig().GetInt("auth.token_ttl_seconds")
	if ttl <= 0 {
		return "", errors.New("登录 token 有效期未配置")
	}
	now := time.Now()
	claims := loginClaims{LoginUser: user, RegisteredClaims: jwt.RegisteredClaims{
		Issuer: zpkTokenIssuer, Audience: jwt.ClaimStrings{w7.AppSdk.Appid},
		IssuedAt: jwt.NewNumericDate(now), NotBefore: jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(ttl) * time.Second)),
	}}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(key)
	if err != nil {
		return "", err
	}
	ctx.Header(ZpkTokenHeader, token)
	return token, nil
}

func (Login) GetUserInfo(ctx *gin.Context) (*LoginUser, error) {
	key, err := loginSigningKey()
	if err != nil {
		return nil, err
	}
	claims := &loginClaims{}
	token, err := jwt.ParseWithClaims(ctx.GetHeader(ZpkTokenHeader), claims, func(*jwt.Token) (interface{}, error) {
		return key, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer(zpkTokenIssuer),
		jwt.WithAudience(w7.AppSdk.Appid), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
	if err != nil || !token.Valid || claims.UserID <= 0 || claims.IssuedAt == nil || claims.NotBefore == nil {
		return nil, ErrInvalidToken
	}
	ctx.Set("zpk_login_claims", claims)
	return &claims.LoginUser, nil
}

func (l Login) RefreshExpire(ctx *gin.Context) (string, error) {
	value, _ := ctx.Get("zpk_login_claims")
	claims := value.(*loginClaims)
	// Renew only in the latter half of the token lifetime; no server-side state.
	if time.Until(claims.ExpiresAt.Time) > claims.ExpiresAt.Sub(claims.IssuedAt.Time)/2 {
		return "", nil
	}
	return l.SaveUserInfo(ctx, claims.LoginUser)
}
