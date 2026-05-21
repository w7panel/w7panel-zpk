package middleware

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/url"
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/middleware"
)

type W7App struct {
	middleware.Abstract
}

func (m W7App) Process(ctx *gin.Context) {
	err := ctx.Request.ParseForm()
	if err != nil {
		m.JsonResponseWithServerError(ctx, err)
		ctx.Abort()
		return
	}

	sign := ctx.Request.FormValue("sign")
	params := make(map[string]string)
	for key, values := range ctx.Request.Form {
		params[key] = values[0]
	}
	slog.Info("app auth", "params", params)
	nsign := makeSign(facade.GetConfig().GetString("w7.secret"), params)
	if nsign != sign {
		//m.JsonResponseWithServerError(ctx, errors.New("签名错误"))
		//ctx.Abort()
		//return
	}

	ctx.Set("appid", params["appid"])
	ctx.Next()
}

func makeSign(secret string, params map[string]string) string {
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
