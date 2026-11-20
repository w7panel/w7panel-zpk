package controller

import (
	"io"
	"mime"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Download struct{ Abstract }

func (c Download) Export(ctx *gin.Context) {
	type ParamsValidate struct {
		Ref    string `json:"ref" binding:"required"`
		Format string `json:"format" binding:"required"`
		Full   bool   `json:"full"`
	}
	params := ParamsValidate{}
	if !c.Bind(ctx, &params) {
		return
	}
	f, e := c.Logic.Export(ctx.Request.Context(), actor(ctx), params.Ref, params.Format, params.Full, nil)
	if e != nil {
		c.Respond(ctx, nil, e)
		return
	}
	c.send(ctx, f, params.Format)
}

func (c Download) Public(ctx *gin.Context) {
	format := ctx.DefaultQuery("format", "zip")
	f, e := c.Logic.ExportLink(ctx.Request.Context(), ctx.Query("token"), format)
	if e != nil {
		c.Respond(ctx, nil, e)
		return
	}
	c.send(ctx, f, format)
}

func (c Download) CreateLink(ctx *gin.Context) {
	type ParamsValidate struct {
		Ref   string `json:"ref" binding:"required"`
		Hours int    `json:"hours" binding:"required,min=1,max=720"`
	}
	params := ParamsValidate{}
	if !c.Bind(ctx, &params) {
		return
	}
	r, e := c.Logic.CreateLink(ctx.Request.Context(), actor(ctx), params.Ref, params.Hours)
	c.Respond(ctx, r, e)
}

func (c Download) Links(ctx *gin.Context) {
	type ParamsValidate struct {
		Ref string `json:"ref" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Bind(ctx, &params) {
		return
	}
	r, e := c.Logic.Links(ctx.Request.Context(), actor(ctx), params.Ref)
	c.Respond(ctx, r, e)
}

func (c Download) RevokeLink(ctx *gin.Context) {
	type ParamsValidate struct {
		ID int32 `json:"id" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Bind(ctx, &params) {
		return
	}
	c.Respond(ctx, nil, c.Logic.RevokeLink(ctx.Request.Context(), actor(ctx), params.ID))
}

func (c Download) send(ctx *gin.Context, f *os.File, format string) {
	defer f.Close()
	defer os.Remove(f.Name())
	ext := "zip"
	kind := "application/zip"
	if format == "markdown" {
		ext = "md"
		kind = "text/markdown; charset=utf-8"
	}
	stat, e := f.Stat()
	if e != nil {
		c.Respond(ctx, nil, e)
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.Header("Referrer-Policy", "no-referrer")
	ctx.Header("X-Content-Type-Options", "nosniff")
	ctx.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": "documents." + ext}))
	ctx.Header("Content-Length", strconv.FormatInt(stat.Size(), 10))
	ctx.Header("Content-Type", kind)
	ctx.Status(200)
	_, _ = io.Copy(ctx.Writer, f)
}
