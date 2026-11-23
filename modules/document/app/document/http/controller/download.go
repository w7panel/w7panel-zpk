package controller

import (
	"io"
	"mime"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/modules/document/app/document/logic"
)

type Download struct{ Abstract }

func (c Download) Export(ctx *gin.Context) {
	f, e := logic.Document{}.Export(ctx.Request.Context(), actor(ctx), ctx.Query("ref"))
	if e != nil {
		c.Respond(ctx, nil, e)
		return
	}
	c.send(ctx, f)
}

func (c Download) send(ctx *gin.Context, f *os.File) {
	defer f.Close()
	defer os.Remove(f.Name())
	stat, e := f.Stat()
	if e != nil {
		c.Respond(ctx, nil, e)
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.Header("Referrer-Policy", "no-referrer")
	ctx.Header("X-Content-Type-Options", "nosniff")
	ctx.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": "documents.zip"}))
	ctx.Header("Content-Length", strconv.FormatInt(stat.Size(), 10))
	ctx.Header("Content-Type", "application/zip")
	ctx.Status(200)
	_, _ = io.Copy(ctx.Writer, f)
}
