package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/modules/document/app/document/logic"
)

type File struct{ Abstract }

func (c File) List(ctx *gin.Context) {
	type ParamsValidate struct {
		Ref string `json:"ref" binding:"required,document_ref"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	result, e := logic.File{}.List(ctx.Request.Context(), currentUser(ctx), params.Ref)
	c.Respond(ctx, result, e)
}

func (c File) Read(ctx *gin.Context) {
	type ParamsValidate struct {
		Ref  string `json:"ref" binding:"required,document_ref"`
		Path string `json:"path" binding:"required,document_path"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	f, b, e := logic.File{}.Read(ctx.Request.Context(), currentUser(ctx), params.Ref, params.Path)
	if e != nil {
		c.Respond(ctx, nil, e)
		return
	}
	c.Respond(ctx, gin.H{"path": f.Path, "title": f.Title, "content": string(b)}, nil)
}

func (c File) Save(ctx *gin.Context) {
	type ParamsValidate struct {
		Ref     string `json:"ref" binding:"required,document_ref"`
		Path    string `json:"path" binding:"required,document_path"`
		Title   string `json:"title" binding:"maxbytes=255"`
		Content string `json:"content" binding:"maxbytes=4194304"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	f, e := logic.File{}.Save(ctx.Request.Context(), currentUser(ctx), params.Ref, params.Path, params.Title, []byte(params.Content))
	if e != nil {
		c.Respond(ctx, nil, e)
		return
	}
	c.Respond(ctx, gin.H{"path": f.Path}, nil)
}

func (c File) Rename(ctx *gin.Context) {
	type ParamsValidate struct {
		Ref   string `json:"ref" binding:"required,document_ref"`
		Path  string `json:"path" binding:"required,document_path"`
		Title string `json:"title" binding:"required,notblank,maxbytes=255"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	c.Respond(ctx, nil, logic.File{}.Rename(ctx.Request.Context(), currentUser(ctx), params.Ref, params.Path, params.Title))
}

func (c File) Delete(ctx *gin.Context) {
	type ParamsValidate struct {
		Ref  string `json:"ref" binding:"required,document_ref"`
		Path string `json:"path" binding:"required,document_path"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	c.Respond(ctx, nil, logic.File{}.Delete(ctx.Request.Context(), currentUser(ctx), params.Ref, params.Path))
}

func (c File) Order(ctx *gin.Context) {
	type ParamsValidate struct {
		Ref   string   `json:"ref" binding:"required,document_ref"`
		Paths []string `json:"paths" binding:"max=10000,unique,dive,document_path"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	c.Respond(ctx, nil, logic.File{}.Order(ctx.Request.Context(), currentUser(ctx), params.Ref, params.Paths))
}
