package controller

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	systemlogic "github.com/w7panel/w7panel-zpk/app/system/logic"
	"github.com/w7panel/w7panel-zpk/common/function"
	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/controller"
)

type OIDC struct {
	controller.Abstract
}

func (c OIDC) Login(ctx *gin.Context) {
	type ParamsValidate struct {
		RedirectURL string `form:"redirect_url" binding:"omitempty"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	nonce, err := function.RandomOIDCToken()
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	state, err := function.MakeOIDCState(nonce, params.RedirectURL, oidcStateSecret(), 10*time.Minute)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	oidcLogic := systemlogic.OIDC{}
	cfg := oidcLogic.Config()
	loginURL, err := oidcLogic.LoginURL(ctx.Request.Context(), cfg, cfg.RedirectURI, state, nonce)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	ctx.Redirect(http.StatusFound, loginURL)
}

func (c OIDC) Callback(ctx *gin.Context) {
	oidcLogic := systemlogic.OIDC{}
	cfg := oidcLogic.Config()
	if errMsg := ctx.Query("error"); errMsg != "" {
		c.JsonResponseWithError(ctx, errors.New(errMsg), http.StatusBadRequest)
		return
	}
	code := ctx.Query("code")
	if code == "" {
		c.JsonResponseWithError(ctx, errors.New("code is required"), http.StatusBadRequest)
		return
	}
	state, err := function.VerifyOIDCState(ctx.Query("state"), oidcStateSecret())
	if err != nil {
		c.JsonResponseWithError(ctx, err, http.StatusBadRequest)
		return
	}

	info, err := oidcLogic.ExchangeCode(ctx.Request.Context(), cfg, cfg.RedirectURI, code, state.Nonce)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	user, err := commonlogic.User{}.GetOrCreatePanelUser(info.Subject, info.Username, info.Role, cfg.AutoCreateUser)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	if user == nil {
		c.JsonResponseWithError(ctx, errors.New("用户不存在"), http.StatusUnauthorized)
		return
	}
	if err := (commonlogic.Session{}).WriteUserSession(ctx, commonlogic.UserSession{
		UserID:     user.ID,
		ConsoleUid: info.ConsoleUID,
		Username:   user.Username,
	}); err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	ctx.Set("console_uid", info.ConsoleUID)
	ctx.Set("user", user)

	success := state.RedirectURL
	if success == "" {
		success = cfg.SuccessRedirect
	}
	if success == "" {
		success = "/"
	}
	ctx.Redirect(http.StatusFound, success)
}

func oidcStateSecret() string {
	return facade.GetConfig().GetString("setting.secret")
}
