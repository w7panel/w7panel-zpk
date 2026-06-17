package controller

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/app/respo/logic"
	"github.com/w7panel/w7panel-zpk/common/accessor"
	"github.com/w7panel/w7panel-zpk/common/dao"
)

type FormulaSetting struct {
	Abstract
}

func (c FormulaSetting) Get(ctx *gin.Context) {
	type ParamsValidate struct {
		Identifie string `form:"identifie" json:"identifie" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	formula := logic.GetFormulaByName(params.Identifie)
	if formula == nil {
		c.JsonResponseWithServerError(ctx, errors.New("制品不存在"))
		return
	}

	setting := formula.Setting
	if setting == nil {
		setting = &accessor.FormulaSettingOption{}
	}

	c.JsonResponseWithoutError(ctx, setting)
}

func (c FormulaSetting) Set(ctx *gin.Context) {
	type ParamsValidate struct {
		Identifie                     string `form:"identifie" json:"identifie" binding:"required"`
		SupportCrossUpgrade           bool   `form:"support_cross_upgrade" json:"support_cross_upgrade"`
		SupportAutoPublishToZpkMarket bool   `json:"support_auto_publish_to_zpk_market"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	formula := logic.GetFormulaByName(params.Identifie)
	if formula == nil {
		c.JsonResponseWithServerError(ctx, errors.New("制品不存在"))
		return
	}

	formula.Setting.SupportCrossUpgrade = params.SupportCrossUpgrade
	formula.Setting.SupportAutoPublishToZpkMarket = params.SupportAutoPublishToZpkMarket

	_, err := dao.Q.Formula.Where(dao.Q.Formula.ID.Eq(formula.ID)).Update(dao.Q.Formula.Setting, formula.Setting)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	c.JsonSuccessResponse(ctx)
}
