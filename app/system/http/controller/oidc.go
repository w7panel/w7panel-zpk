package controller

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	systemlogic "github.com/w7panel/w7panel-zpk/app/system/logic"
	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/controller"
)

type OIDC struct {
	controller.Abstract
}

func (c OIDC) LoginFromW7panel(ctx *gin.Context) {
	type ParamsValidate struct {
		AccessToken string `form:"access_token" json:"access_token" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	oidcLogic := systemlogic.OIDC{}

	info, err := oidcLogic.UserInfo(ctx.Request.Context(), params.AccessToken)
	slog.Info("oidc accesstoken login", "token", params.AccessToken, "info", info, "err", err)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	if info.Subject == "" || info.Username == "" {
		c.JsonResponseWithError(ctx, errors.New("用户信息异常"), http.StatusUnauthorized)
		return
	}

	user, err := commonlogic.User{}.GetOrCreatePanelUser(info.Subject, info.Username, info.Role)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	if user == nil {
		c.JsonResponseWithError(ctx, errors.New("用户不存在"), http.StatusUnauthorized)
		return
	}

	token, err := (commonlogic.Session{}).SaveUserInfo(ctx, commonlogic.UserSession{
		UserID:     user.ID,
		ConsoleUid: info.ConsoleUID,
		Username:   user.Username,
	})
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	c.JsonResponseWithoutError(ctx, map[string]interface{}{
		"token": token,
	})
}
