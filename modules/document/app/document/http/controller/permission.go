package controller

import "github.com/gin-gonic/gin"

type Permission struct{ Abstract }

func (c Permission) Set(ctx *gin.Context) {
	type ParamsValidate struct {
		Ref          string   `json:"ref" binding:"required"`
		Path         string   `json:"path"`
		TargetUserID int32    `json:"target_user_id" binding:"required"`
		Actions      []string `json:"actions"`
	}
	params := ParamsValidate{}
	if !c.Bind(ctx, &params) {
		return
	}
	c.Respond(ctx, nil, c.Logic.SetPermission(ctx.Request.Context(), actor(ctx), params.Ref, params.Path, params.TargetUserID, params.Actions))
}
