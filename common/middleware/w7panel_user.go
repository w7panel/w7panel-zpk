package middleware

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/middleware"
)

type W7PanelUser struct {
	middleware.Abstract
	CanSkip          bool
	NoAutoCreateUser bool
}

type W7PanelUserPayload struct {
	Uid        string `json:"uid"`
	Name       string `json:"name"`
	Role       string `json:"role"`
	ConsoleUid int    `json:"console_uid"`
}

func (m W7PanelUser) Process(ctx *gin.Context) {
	panelToken := ctx.Request.Header.Get("X-W7Panel-Token")
	if panelToken == "" {
		if m.CanSkip {
			ctx.Next()
			return
		}

		m.JsonResponseWithError(ctx, errors.New("面板 token 错误"), 401)
		ctx.Abort()
		return
	}

	infoPayloads := strings.Split(panelToken, ".")
	if len(infoPayloads) != 3 {
		m.JsonResponseWithError(ctx, errors.New("token 格式错误"), 401)
		ctx.Abort()
		return
	}
	decodeString, err := base64.RawStdEncoding.DecodeString(infoPayloads[1])
	if err != nil {
		m.JsonResponseWithError(ctx, errors.New("token 格式错误"), 401)
		ctx.Abort()
		return
	}

	type Payload struct {
		Aud  []interface{} `json:"aud"`
		Exp  int64         `json:"exp"`
		Info struct {
			Account W7PanelUserPayload `json:"serviceaccount"`
		} `json:"kubernetes.io"`
	}
	payload := Payload{}
	err = json.Unmarshal(decodeString, &payload)
	if err != nil {
		m.JsonResponseWithError(ctx, errors.New("token 格式错误"), 401)
		ctx.Abort()
		return
	}
	if err = validatePanelTokenExpiry(payload.Exp, time.Now()); err != nil {
		m.JsonResponseWithError(ctx, err, 401)
		ctx.Abort()
		return
	}

	consoleUid := 0
	w7PanelUserName := ""
	w7PanelUserId := ""
	w7PanelUserRole := ""
	if payload.Info.Account.Name != "" {
		if payload.Aud == nil || len(payload.Aud) < 3 {
			payload.Aud = []interface{}{"", founderRole, ""}
			//m.JsonResponseWithError(ctx, errors.New("token 格式错误"), 401)
			//ctx.Abort()
			//return
		}

		val := payload.Aud[1]
		role, ok := val.(string)
		if !ok {
			m.JsonResponseWithError(ctx, errors.New("token 格式错误，缺失角色信息"), 401)
			ctx.Abort()
			return
		}
		val = payload.Aud[2]
		tuid, ok := val.(string)
		if !ok {
			m.JsonResponseWithError(ctx, errors.New("token 格式错误，uid角色信息"), 401)
			ctx.Abort()
			return
		}

		w7PanelUserRole = role
		consoleUid, _ = strconv.Atoi(tuid)
		w7PanelUserName = strings.Trim(payload.Info.Account.Name, "")
		w7PanelUserId = payload.Info.Account.Uid
	} else {
		var consolePayload map[string]interface{}
		err = json.Unmarshal(decodeString, &consolePayload)
		if err == nil {
			if _, exist := consolePayload["user_id"]; exist {
				consoleUid = int(consolePayload["user_id"].(float64))
				w7PanelUserName = "admin"
				w7PanelUserRole = founderRole
				w7PanelUserId = strconv.Itoa(consoleUid)
			}
		}
	}
	if w7PanelUserName == "" || w7PanelUserId == "" {
		m.JsonResponseWithServerError(ctx, errors.New("token格式错误， 缺失用户信息"))
		ctx.Abort()
		return
	}
	ctx.Set("console_uid", int32(consoleUid))

	user, err := getOrCreateUser(w7PanelUserId, w7PanelUserName, w7PanelUserRole, !m.NoAutoCreateUser)

	sessionUser := logic.UserSession{
		ConsoleUid: int32(consoleUid),
	}
	if user != nil {
		sessionUser.UserID = user.ID
		sessionUser.Username = user.Username
	}
	if err1 := (logic.Session{}.WriteUserSession(ctx, sessionUser)); err1 != nil {
		m.JsonResponseWithServerError(ctx, err1)
		ctx.Abort()
		return
	}

	if user == nil && m.NoAutoCreateUser {
		ctx.Next()
		return
	}

	if err != nil {
		m.JsonResponseWithError(ctx, err, 401)
		ctx.Abort()
		return
	}
	if user == nil {
		m.JsonResponseWithError(ctx, errors.New("用户不存在"), 401)
		ctx.Abort()
		return
	}

	ctx.Set("user", user)
	ctx.Next()
}

func validatePanelTokenExpiry(exp int64, now time.Time) error {
	if exp <= 0 {
		return errors.New("token 缺少过期时间")
	}
	if now.Unix() >= exp {
		return errors.New("token 已过期")
	}
	return nil
}
