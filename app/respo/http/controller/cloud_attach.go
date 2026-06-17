package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/common/service/w7/cloudapi"
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

	c.JsonResponseWithoutError(ctx, cloudapi.UploadAttachResp{
		Attach: cloudapi.AttachInfo{
			Path: "/home",
		},
	})
}
