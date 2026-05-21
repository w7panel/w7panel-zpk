package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/middleware"
)

type Cors struct {
	middleware.Abstract
}

func (m Cors) Process(ctx *gin.Context) {
	ctx.Header("Access-Control-Allow-Origin", "*")
	ctx.Header("Access-Control-Allow-Headers", m.getAllowHeader())
	ctx.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	ctx.Header("Access-Control-Expose-Headers", m.getAllowHeader())
	ctx.Header("Access-Control-Allow-Credentials", "false")
	if ctx.Request.Method == "OPTIONS" {
		ctx.AbortWithStatus(http.StatusNoContent)
		return
	}
	ctx.Next()
}

func (m Cors) isAllow(ctx *gin.Context) (string, bool) {
	host := ctx.Request.Header.Get("origin")
	if host == "" {
		host = ctx.Request.Header.Get("referer")
	}
	if host == "" {
		return "", false
	}
	if true {
		return host, true
	}
	allowUrl := []string{
		"https://console.w7.cc",
		"http://console.w7.cc",
		"http://172.16.1.13:8084",
		"http://172.16.1.13",
		"http://172.16.1.13:8085",
		"http://devtool.w7.com",
	}
	for _, value := range allowUrl {
		if value == host {
			return host, true
		}
	}
	return "", false
}

func (m Cors) getAllowHeader() string {
	allowHeader := []string{
		"Content-Length",
		"Content-Type",
		"X-Auth-Token",
		"Origin",
		"Authorization",
		"X-Requested-With",
		"x-requested-with",
		"x-xsrf-token",
		"x-csrf-token",
		"x-w7-from",
		"access-token",
		"Api-Version",
		"Access-Control-Allow-Origin",
		"Access-Control-Allow-Headers",
		"Access-Control-Allow-Methods",
		"authority",
		"uid",
		"uuid",
		"X-W7Panel-Token",
	}
	return strings.Join(allowHeader, ",")
}
