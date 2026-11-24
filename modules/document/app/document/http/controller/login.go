package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/modules/document/app/document/logic"
)

type Login struct {
	Abstract
}

func (c Login) FromPanel(ctx *gin.Context) {
	type ParamsValidate struct {
		AccessToken string `json:"access_token" form:"access_token" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	token, err := logic.Login{}.FromPanel(ctx, params.AccessToken)
	if err != nil {
		c.Respond(ctx, nil, err)
		return
	}
	c.JsonResponseWithoutError(ctx, gin.H{"token": token})
}

func (c Login) CurrentUser(ctx *gin.Context) {
	c.JsonResponseWithoutError(ctx, currentUser(ctx))
}
