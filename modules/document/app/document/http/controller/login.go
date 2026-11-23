package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/modules/document/app/document/logic"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/zpk"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/controller"
)

type Login struct {
	controller.Abstract
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
		status := http.StatusServiceUnavailable
		if errors.Is(err, zpk.ErrUnauthenticated) {
			status = http.StatusUnauthorized
		}
		c.JsonResponseWithError(ctx, err, status)
		return
	}
	c.JsonResponseWithoutError(ctx, gin.H{"token": token})
}

func (c Login) CurrentUser(ctx *gin.Context) {
	c.JsonResponseWithoutError(ctx, actor(ctx).Principal)
}
