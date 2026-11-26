package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/modules/document/app/document/logic"
	"github.com/w7panel/w7panel-zpk/modules/document/common/entity"
)

type Library struct{ Abstract }

func (c Library) Info(ctx *gin.Context) {
	type ParamsValidate struct {
		Ref string `json:"ref" binding:"required,document_ref"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	library, err := logic.Library{}.Info(ctx.Request.Context(), currentUser(ctx), params.Ref)
	c.Respond(ctx, library, err)
}

func (c Library) List(ctx *gin.Context) {
	type ParamsValidate struct {
		Category string `json:"category" binding:"omitempty,document_identifie"`
		Page     int    `json:"page" binding:"min=1"`
		Limit    int    `json:"limit" binding:"min=1,max=100"`
	}
	params := ParamsValidate{Page: 1, Limit: 20}
	if !c.Validate(ctx, &params) {
		return
	}
	result, e := logic.Library{}.List(ctx.Request.Context(), currentUser(ctx), params.Category, params.Page, params.Limit)
	c.Respond(ctx, result, e)
}

func (c Library) Create(ctx *gin.Context) {
	type FileParamsValidate struct {
		Path      string `json:"path" binding:"required,document_path"`
		Title     string `json:"title" binding:"maxbytes=255"`
		Content   string `json:"content" binding:"maxbytes=4194304"`
		Protected int32  `json:"protected" binding:"oneof=0 1"`
	}
	type ParamsValidate struct {
		Ref         string               `json:"ref" binding:"required,document_ref"`
		Name        string               `json:"name" binding:"required,notblank,maxbytes=255"`
		Description string               `json:"description" binding:"maxbytes=4096"`
		Public      int32                `json:"public" binding:"oneof=0 1"`
		OwnerName   string               `json:"owner_name" binding:"maxbytes=255"`
		OwnerID     int32                `json:"owner_id" binding:"min=0"`
		Protected   int32                `json:"protected" binding:"oneof=0 1"`
		Files       []FileParamsValidate `json:"files" binding:"max=10000,unique=Path,dive"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	category, identifie := logic.SplitRef(params.Ref)
	library := entity.DocumentLibrary{Category: category, Identifie: identifie, Name: params.Name, Description: params.Description, Public: params.Public, UserID: params.OwnerID, Username: params.OwnerName, Protected: params.Protected}
	files := make([]logic.CreateFile, 0, len(params.Files))
	for _, file := range params.Files {
		files = append(files, logic.CreateFile{Path: file.Path, Title: file.Title, Content: file.Content, Protected: file.Protected})
	}
	result, e := logic.Library{}.Create(ctx.Request.Context(), currentUser(ctx), library, files)
	c.Respond(ctx, result, e)
}

func (c Library) Edit(ctx *gin.Context) {
	type ParamsValidate struct {
		Ref         string `json:"ref" binding:"required,document_ref"`
		Name        string `json:"name" binding:"required,notblank,maxbytes=255"`
		Description string `json:"description" binding:"maxbytes=4096"`
		Public      *int32 `json:"public" binding:"omitempty,oneof=0 1"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	c.Respond(ctx, nil, logic.Library{}.Edit(ctx.Request.Context(), currentUser(ctx), params.Ref, params.Name, params.Description, params.Public))
}

func (c Library) Delete(ctx *gin.Context) {
	type ParamsValidate struct {
		Ref string `json:"ref" binding:"required,document_ref"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	c.Respond(ctx, nil, logic.Library{}.Delete(ctx.Request.Context(), currentUser(ctx), params.Ref))
}
