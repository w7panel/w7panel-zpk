package logic

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/common/service/session"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"golang.org/x/sync/singleflight"
)

const ZpkTokenHeader = session.TokenHeader

const sessionRefreshAtKey = "zpk_session_refresh_at"
const defaultSessionRefreshIntervalSeconds = 300

var sessionRefreshGroup singleflight.Group

type Session struct {
	Logic
}

type UserSession struct {
	UserID           int32  `json:"user_id"`
	ConsoleUid       int32  `json:"console_uid"`
	Username         string `json:"username"`
	PanelAccessToken string `json:"panel_access_token,omitempty"`
}

// PanelAccessToken is the original OIDC credential, never a ZPK session token.
func (Session) PanelAccessToken(ctx *gin.Context) string {
	return ctx.GetString("panel_access_token")
}

func (l Session) SaveUserInfo(ctx *gin.Context, user UserSession) (string, error) {
	if err := l.writeUserSession(ctx, user); err != nil {
		return "", err
	}
	token := ctx.Writer.Header().Get(ZpkTokenHeader)
	if token == "" {
		return "", errors.New("session token is empty")
	}
	return token, nil
}

func (l Session) GetUserInfo(ctx *gin.Context) (*UserSession, error) {
	curSession := sessions.Default(ctx)

	val := curSession.Get("user")
	if val == nil {
		return nil, nil
	}

	content, ok := val.(string)
	if !ok {
		return nil, errors.New("invalid session user")
	}
	user := UserSession{}
	err := json.Unmarshal([]byte(content), &user)
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
	curSession.Set(sessionRefreshAtKey, time.Now().Unix())

	return curSession.Save()
}

func (l Session) RefreshExpire(ctx *gin.Context) (string, error) {
	curSession := sessions.Default(ctx)
	now := time.Now().Unix()
	if lastRefreshAt := sessionRefreshAt(curSession.Get(sessionRefreshAtKey)); now-lastRefreshAt < defaultSessionRefreshIntervalSeconds {
		return "", nil
	}

	result, err, _ := sessionRefreshGroup.Do("session:"+curSession.ID(), func() (interface{}, error) {
		now := time.Now().Unix()
		if lastRefreshAt := sessionRefreshAt(curSession.Get(sessionRefreshAtKey)); now-lastRefreshAt < defaultSessionRefreshIntervalSeconds {
			return "", nil
		}

		curSession.Set(sessionRefreshAtKey, now)
		curSession.Options(sessions.Options{MaxAge: facade.GetConfig().GetInt("session.max_age")})
		if err := curSession.Save(); err != nil {
			return "", err
		}
		return ctx.Writer.Header().Get(ZpkTokenHeader), nil
	})
	if err != nil {
		return "", err
	}
	nextToken, _ := result.(string)
	return nextToken, nil
}

func sessionRefreshAt(value interface{}) int64 {
	v, ok := value.(int64)
	if !ok {
		return 0
	}
	return v
}
