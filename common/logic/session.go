package logic

import (
	"encoding/json"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type Session struct {
	Logic
}

type UserSession struct {
	UserID     int32  `json:"user_id"`
	ConsoleUid int32  `json:"console_uid"`
	Username   string `json:"username"`
}

func (l Session) WriteUserSession(ctx *gin.Context, user UserSession) error {
	content, err := json.Marshal(user)
	if err != nil {
		return err
	}

	curSession := sessions.Default(ctx)
	curSession.Set("user", string(content))
	return curSession.Save()
}

func (l Session) ReadUserSession(ctx *gin.Context) (*UserSession, error) {
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
