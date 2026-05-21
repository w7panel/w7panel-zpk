package controller

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/common/function"
	"github.com/w7panel/w7panel-zpk/common/service/w7"
)

type CloudAttach struct {
	Abstract
}

func (c CloudAttach) UploadImg(ctx *gin.Context) {
	type ParamsValidate struct {
		FileName string `form:"file_name" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	file, err := ctx.FormFile("file")
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	// 校验文件类型
	ext := filepath.Ext(file.Filename)
	if ext != ".jpg" && ext != ".png" && ext != ".jpeg" {
		c.JsonResponseWithServerError(ctx, errors.New("file type not support"))
		return
	}
	md5 := function.GetMd5(params.FileName + file.Filename)
	savePath := filepath.Join(os.TempDir(), "upload") + md5 + ".png"
	err = ctx.SaveUploadedFile(file, savePath)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	defer os.Remove(savePath)

	imgFile, err := os.Open(savePath)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	host := "console.w7.cc"
	ticket, err1 := w7.W7CloudAttach.GetJsTicketByHost(host)
	if err1 != nil && err1.IsError() {
		c.JsonResponseWithError(ctx, err1, 500)
		return
	}

	img, err := w7.W7CloudAttach.UploadImg(ticket, imgFile, params.FileName)
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}

	c.JsonResponseWithoutError(ctx, img)
}
