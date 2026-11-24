package controller

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/app/respo/logic"
	"github.com/w7panel/w7panel-zpk/app/respo/logic/formula"
	userlogic "github.com/w7panel/w7panel-zpk/common/logic"
	documentsdk "github.com/w7panel/w7panel-zpk/common/service/w7/document"
)

// FormulaDocument adapts artifact requests to the independent document service.
type FormulaDocument struct{ Abstract }

func (c FormulaDocument) Files(ctx *gin.Context) {
	type ParamsValidate struct {
		Identifie string `form:"identifie" json:"identifie" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	documents := logic.FormulaDocument{}
	token := (userlogic.Session{}).PanelAccessToken(ctx)
	reference, err := documents.Ensure(ctx.Request.Context(), token, params.Identifie, (userlogic.User{}).GetUser(ctx))
	if err != nil {
		c.documentError(ctx, err)
		return
	}
	files, err := documents.Files(ctx.Request.Context(), token, reference)
	if err != nil {
		c.documentError(ctx, err)
		return
	}
	c.JsonResponseWithoutError(ctx, gin.H{"list": files})
}

func (c FormulaDocument) Save(ctx *gin.Context) {
	type ParamsValidate struct {
		Identifie string `form:"identifie" json:"identifie" binding:"required"`
		Filename  string `form:"filename" json:"filename" binding:"required"`
		Content   string `form:"content" json:"content"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	filename, err := formula.NormalizeFormulaFilePath(params.Filename)
	if err != nil {
		c.JsonResponseWithError(ctx, err, 400)
		return
	}
	documents := logic.FormulaDocument{}
	token := (userlogic.Session{}).PanelAccessToken(ctx)
	reference, err := documents.Ensure(ctx.Request.Context(), token, params.Identifie, (userlogic.User{}).GetUser(ctx))
	if err == nil {
		err = documents.Save(ctx.Request.Context(), token, reference, filename, sanitizeFormulaAttachContent(params.Content))
	}
	if err != nil {
		c.documentError(ctx, err)
		return
	}
	c.JsonSuccessResponse(ctx)
}

func (c FormulaDocument) documentError(ctx *gin.Context, err error) {
	status := 503
	var responseError *documentsdk.ResponseError
	if errors.As(err, &responseError) {
		status = responseError.StatusCode
	}
	c.JsonResponseWithError(ctx, err, status)
}
