package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/modules/document/app/document/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/controller"
)

type Login struct {
	controller.Abstract
	Logic *logic.Login
}

func (c Login) FromPanel(ctx *gin.Context) {
	type ParamsValidate struct {
		AccessToken string `json:"access_token" form:"access_token" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	token, err := c.Logic.FromPanel(ctx.Request.Context(), params.AccessToken)
	if err != nil {
		c.JsonResponseWithError(ctx, err, 401)
		return
	}
	c.JsonResponseWithoutError(ctx, gin.H{"token": token})
}

func (c Login) CurrentUser(ctx *gin.Context) {
	c.JsonResponseWithoutError(ctx, actor(ctx).Principal)
}
