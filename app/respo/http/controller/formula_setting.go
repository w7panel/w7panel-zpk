package controller

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/common/accessor"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
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

	formula, err := c.getDepot().GetFormula(params.Identifie, "", logic2.User{}.GetUser(ctx))
	if err != nil {
		c.JsonResponseWithServerError(ctx, errors.New("制品不存在"))
		return
	}

	setting := accessor.FormulaSettingOption{}
	if formula.Setting != nil {
		setting = *formula.Setting
	}
	if setting.BaseInfo == nil {
		setting.BaseInfo = formula.GetBaseInfo()
		_, err = dao.Q.Formula.Where(dao.Q.Formula.ID.Eq(formula.ID)).Updates(entity.Formula{
			Title:   setting.BaseInfo.Name,
			Setting: &setting,
		})
		if err != nil {
			c.JsonResponseWithServerError(ctx, err)
			return
		}
	}

	c.JsonResponseWithoutError(ctx, &setting)
}

func (c FormulaSetting) Set(ctx *gin.Context) {
	type ParamsValidate struct {
		Identifie                     string                          `form:"identifie" json:"identifie" binding:"required"`
		SupportCrossUpgrade           *bool                           `form:"support_cross_upgrade" json:"support_cross_upgrade"`
		SupportAutoPublishToZpkMarket *bool                           `form:"support_auto_publish_to_zpk_market" json:"support_auto_publish_to_zpk_market"`
		EnableServicePackageFee       *bool                           `form:"enable_service_package_fee" json:"enable_service_package_fee"`
		BaseInfo                      *accessor.FormulaBaseInfoOption `json:"base_info"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	formula, err := c.getDepot().GetFormula(params.Identifie, "", logic2.User{}.GetUser(ctx))
	if err != nil {
		c.JsonResponseWithServerError(ctx, errors.New("制品不存在"))
		return
	}
	if formula.Setting == nil {
		formula.Setting = &accessor.FormulaSettingOption{}
	}

	if params.SupportCrossUpgrade != nil {
		formula.Setting.SupportCrossUpgrade = *params.SupportCrossUpgrade
	}
	if params.SupportAutoPublishToZpkMarket != nil {
		formula.Setting.SupportAutoPublishToZpkMarket = *params.SupportAutoPublishToZpkMarket
	}
	if params.EnableServicePackageFee != nil {
		formula.Setting.EnableServicePackageFee = *params.EnableServicePackageFee
	}
	if params.BaseInfo != nil {
		params.BaseInfo.Name = strings.TrimSpace(params.BaseInfo.Name)
		if params.BaseInfo.Name == "" {
			c.JsonResponseWithServerError(ctx, errors.New("制品名称不能为空"))
			return
		}
		if params.BaseInfo.Annotation == nil {
			params.BaseInfo.Annotation = map[string]interface{}{}
		}
		formula.Setting.BaseInfo = params.BaseInfo
	}

	update := entity.Formula{
		Setting: formula.Setting,
	}
	if params.BaseInfo != nil {
		update.Title = params.BaseInfo.Name
	}
	_, err = dao.Q.Formula.Where(dao.Q.Formula.ID.Eq(formula.ID)).Updates(update)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	c.JsonSuccessResponse(ctx)
}
