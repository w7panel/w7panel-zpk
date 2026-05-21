package middleware

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/middleware"
)

type W7AppV2 struct {
	middleware.Abstract
}

func (m W7AppV2) Process(ctx *gin.Context) {
	var jsonParams map[string]interface{}
	if err := ctx.ShouldBindJSON(&jsonParams); err != nil {
		slog.Error("parse json failed", "error", err)
		m.JsonResponseWithServerError(ctx, errors.New("无效的 JSON 数据"))
		ctx.Abort()
		return
	}
	sign, ok := jsonParams["sign"].(string)
	if !ok {
		m.JsonResponseWithServerError(ctx, errors.New("缺少 sign 参数"))
		ctx.Abort()
		return
	}
	body, ok := jsonParams["body"].(string)
	if !ok {
		m.JsonResponseWithServerError(ctx, errors.New("缺少 body 参数"))
		ctx.Abort()
		return
	}

	params := make(map[string]string, len(jsonParams))
	for k, v := range jsonParams {
		if v == nil {
			params[k] = ""
		} else {
			params[k] = v.(string)
		}
	}

	slog.Info("app auth", "params", params)
	nsign := makeSignV2(facade.GetConfig().GetString("zpk.secret"), params)
	if nsign != sign {
		m.JsonResponseWithServerError(ctx, errors.New("签名错误"))
		ctx.Abort()
		return
	}

	ctx.Request.PostForm = nil
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("appid", params["appid"])
	ctx.Request.Body = io.NopCloser(bytes.NewBuffer([]byte(body)))
	ctx.Next()
}

func makeSignV2(secret string, params map[string]string) string {
	_, ok := params["sign"]
	if ok {
		delete(params, "sign")
	}

	var keys []string
	signStr := ""
	for s, _ := range params {
		if s == "sign" {
			continue
		}
		keys = append(keys, s)
	}
	sort.Strings(keys)
	for i, k := range keys {
		signStr += fmt.Sprintf("%s=%s", url.QueryEscape(k), url.QueryEscape(params[k]))
		if i < len(keys)-1 {
			signStr += "&"
		}
	}

	signStr += secret
	md5Ctx := md5.New()
	md5Ctx.Write([]byte(signStr))
	return hex.EncodeToString(md5Ctx.Sum(nil))
}
