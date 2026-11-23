package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/modules/document/app/document/logic"
)

type File struct{ Abstract }

func (c File) List(ctx *gin.Context) {
	type ParamsValidate struct {
		Ref string `json:"ref" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Bind(ctx, &params) {
		return
	}
	result, e := logic.Document{}.Files(ctx.Request.Context(), actor(ctx), params.Ref)
	c.Respond(ctx, result, e)
}

func (c File) Read(ctx *gin.Context) {
	type ParamsValidate struct {
		Ref  string `json:"ref" binding:"required"`
		Path string `json:"path" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Bind(ctx, &params) {
		return
	}
	f, b, e := logic.Document{}.Read(ctx.Request.Context(), actor(ctx), params.Ref, params.Path)
	if e != nil {
		c.Respond(ctx, nil, e)
		return
	}
	c.Respond(ctx, gin.H{"path": f.Path, "title": f.Title, "revision": f.Revision, "content": string(b)}, nil)
}

func (c File) Save(ctx *gin.Context) {
	type ParamsValidate struct {
		Ref      string `json:"ref" binding:"required"`
		Path     string `json:"path" binding:"required"`
		Title    string `json:"title"`
		Content  string `json:"content"`
		Revision int32  `json:"revision"`
	}
	params := ParamsValidate{}
	if !c.Bind(ctx, &params) {
		return
	}
	f, e := logic.Document{}.Save(ctx.Request.Context(), actor(ctx), params.Ref, params.Path, params.Title, []byte(params.Content), params.Revision)
	if e != nil {
		c.Respond(ctx, nil, e)
		return
	}
	c.Respond(ctx, gin.H{"path": f.Path, "revision": f.Revision}, nil)
}

func (c File) Rename(ctx *gin.Context) {
	type ParamsValidate struct {
		Ref      string `json:"ref" binding:"required"`
		Path     string `json:"path" binding:"required"`
		Title    string `json:"title" binding:"required"`
		Revision int32  `json:"revision" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Bind(ctx, &params) {
		return
	}
	c.Respond(ctx, nil, logic.Document{}.Rename(ctx.Request.Context(), actor(ctx), params.Ref, params.Path, params.Title, params.Revision))
}

func (c File) Delete(ctx *gin.Context) {
	type ParamsValidate struct {
		Ref      string `json:"ref" binding:"required"`
		Path     string `json:"path" binding:"required"`
		Revision int32  `json:"revision" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Bind(ctx, &params) {
		return
	}
	c.Respond(ctx, nil, logic.Document{}.DeleteFile(ctx.Request.Context(), actor(ctx), params.Ref, params.Path, params.Revision))
}

func (c File) Order(ctx *gin.Context) {
	type ParamsValidate struct {
		Ref      string   `json:"ref" binding:"required"`
		Paths    []string `json:"paths"`
		Revision int32    `json:"revision" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Bind(ctx, &params) {
		return
	}
	c.Respond(ctx, nil, logic.Document{}.Order(ctx.Request.Context(), actor(ctx), params.Ref, params.Paths, params.Revision))
}
