package logic

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/zpk"
)

const TokenHeader = "X-Document-Token"
const SessionName = "DOCUMENT_SESSION_ID"

type Session struct{}

func (Session) SaveUserInfo(ctx *gin.Context, principal zpk.User) (string, error) {
	content, err := json.Marshal(principal)
	if err != nil {
		return "", err
	}
	current := sessions.Default(ctx)
	current.Set("user", string(content))
	if err = current.Save(); err != nil {
		return "", err
	}
	for _, cookie := range (&http.Response{Header: ctx.Writer.Header()}).Cookies() {
		if cookie.Name == SessionName {
			return cookie.Value, nil
		}
	}
	return "", errors.New("文档会话凭据为空")
}

func (Session) GetUserInfo(ctx *gin.Context, token string) (zpk.User, error) {
	// Header credentials take precedence over any browser cookie.
	cookies := ctx.Request.Cookies()
	ctx.Request.Header.Del("Cookie")
	for _, cookie := range cookies {
		if cookie.Name != SessionName {
			ctx.Request.AddCookie(cookie)
		}
	}
	ctx.Request.AddCookie(&http.Cookie{Name: SessionName, Value: strings.TrimSpace(token)})
	value, ok := sessions.Default(ctx).Get("user").(string)
	if !ok {
		return zpk.User{}, zpk.ErrUnauthenticated
	}
	var principal zpk.User
	if err := json.Unmarshal([]byte(value), &principal); err != nil {
		return zpk.User{}, zpk.ErrUnauthenticated
	}
	if principal.ID <= 0 {
		return zpk.User{}, zpk.ErrUnauthenticated
	}
	return principal, nil
}
