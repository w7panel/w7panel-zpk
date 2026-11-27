package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

// NewDocument forwards browser requests; the document service authenticates them.
// Service-to-service OpenAPI calls continue to use DocumentSdk directly.
func NewDocument(baseURL string) gin.HandlerFunc {
	target, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil || (target.Scheme != "http" && target.Scheme != "https") || target.Host == "" || target.User != nil || target.Path != "" || target.RawQuery != "" || target.Fragment != "" {
		return func(ctx *gin.Context) {
			ctx.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "文档服务地址配置无效"})
		}
	}
	proxy := &httputil.ReverseProxy{
		Rewrite: func(request *httputil.ProxyRequest) {
			request.SetURL(target)
			request.SetXForwarded()
		},
		ErrorHandler: func(w http.ResponseWriter, request *http.Request, err error) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"error":"文档服务不可用"}`))
		},
	}
	return func(ctx *gin.Context) {
		if ctx.Request.URL.Path == "/doc/openapi" || strings.HasPrefix(ctx.Request.URL.Path, "/doc/openapi/") {
			ctx.AbortWithStatus(http.StatusNotFound)
			return
		}
		proxy.ServeHTTP(ctx.Writer, ctx.Request)
	}
}
