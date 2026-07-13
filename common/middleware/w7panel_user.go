package middleware

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/middleware"
)

type W7PanelUser struct {
	middleware.Abstract
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
		ctx.Next()
		return
	}

	slog.Info("auth with X-W7Panel-Token", "token", panelToken)
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

	consoleUid := 0
	if payload.Info.Account.Name != "" {
		if payload.Aud == nil || len(payload.Aud) < 3 {
			payload.Aud = []interface{}{"", "", ""}
			//m.JsonResponseWithError(ctx, errors.New("token 格式错误"), 401)
			//ctx.Abort()
			//return
		}

		val := payload.Aud[2]
		tuid, ok := val.(string)
		if !ok {
			m.JsonResponseWithError(ctx, errors.New("token 格式错误，uid角色信息"), 401)
			ctx.Abort()
			return
		}

		consoleUid, _ = strconv.Atoi(tuid)
	} else {
		var consolePayload map[string]interface{}
		err = json.Unmarshal(decodeString, &consolePayload)
		if err == nil {
			if _, exist := consolePayload["user_id"]; exist {
				consoleUid = int(consolePayload["user_id"].(float64))
			}
		}
	}
	ctx.Set("console_uid", int32(consoleUid))

	ctx.Next()
}
