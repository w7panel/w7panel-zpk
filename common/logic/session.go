package logic

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	ranginesession "github.com/we7coreteam/w7-rangine-go/v2/src/http/session"
)

const ZpkTokenHeader = "X-Zpk-Token"

type Session struct {
	Logic
}

type UserSession struct {
	UserID     int32  `json:"user_id"`
	ConsoleUid int32  `json:"console_uid"`
	Username   string `json:"username"`
}

func (l Session) SaveUserInfo(ctx *gin.Context, user UserSession) (string, error) {
	if err := l.writeUserSession(ctx, user); err != nil {
		return "", err
	}
	token := l.responseSessionToken(ctx)
	if token == "" {
		return "", errors.New("session token is empty")
	}
	return token, nil
}

func (l Session) RefreshExpire(ctx *gin.Context) (string, error) {
	curSession := sessions.Default(ctx)
	curSession.Options(ranginesession.BuildOptions(facade.GetConfig()))
	if err := curSession.Save(); err != nil {
		return "", err
	}
	return l.responseSessionToken(ctx), nil
}

func (l Session) GetUserInfo(ctx *gin.Context, token string) (*UserSession, error) {
	l.injectRequestToken(ctx, token)

	curSession := sessions.Default(ctx)

	val := curSession.Get("user")
	if val == nil {
		return nil, nil
	}

	user := UserSession{}
	err := json.Unmarshal([]byte(val.(string)), &user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (l Session) writeUserSession(ctx *gin.Context, user UserSession) error {
	content, err := json.Marshal(user)
	if err != nil {
		return err
	}

	curSession := sessions.Default(ctx)
	curSession.Set("user", string(content))

	return curSession.Save()
}

func (l Session) responseSessionToken(ctx *gin.Context) string {
	sessionName := l.SessionName()
	for _, rawCookie := range ctx.Writer.Header().Values("Set-Cookie") {
		nameValue := strings.SplitN(rawCookie, ";", 2)[0]
		name, value, ok := strings.Cut(nameValue, "=")
		if ok && name == sessionName {
			return value
		}
	}
	return ""
}

func (l Session) injectRequestToken(ctx *gin.Context, token string) {
	token = strings.TrimSpace(token)
	if token == "" {
		return
	}
	ctx.Request.AddCookie(&http.Cookie{
		Name:  l.SessionName(),
		Value: token,
	})
}

func (l Session) SessionName() string {
	name := facade.GetConfig().GetString("session.name")
	if name == "" {
		name = "SESSION_ID"
	}
	return name
}
