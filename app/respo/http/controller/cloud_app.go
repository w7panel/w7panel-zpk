package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/app/respo/logic"
	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/w7panel/w7panel-zpk/common/service/w7"
	"github.com/w7panel/w7panel-zpk/common/service/w7/devcenter"
)

type CloudApp struct {
	Abstract
}

func (c CloudApp) NotAppList(ctx *gin.Context) {
	type ParamsValidate struct {
		Page int `form:"page" json:"page"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	if params.Page == 0 {
		params.Page = 1
	}
	list, err := w7.DevCenterNotAppSdk.NotAppList(devcenter.NotAppListReq{
		ConsoleUid: logic2.User{}.GetConsoleUid(ctx),
		NotAppType: 1,
		Page:       params.Page,
		PageSize:   16,
	})
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	c.JsonResponseWithoutError(ctx, list)
}

func (c CloudApp) UnPackNotApp(ctx *gin.Context) {
	type ParamsValidate struct {
		Id int `form:"id" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	err := logic.CloudApp{}.UnpackNotAppToFormula(params.Id, logic2.User{}.GetUser(ctx), logic2.User{}.GetConsoleUid(ctx))
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	c.JsonSuccessResponse(ctx)
}
