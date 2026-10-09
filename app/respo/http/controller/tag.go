package controller

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/app/respo/logic"
	"github.com/w7panel/w7panel-zpk/common/dao"
	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
)

type Category struct {
	Abstract
}

func (self Category) Save(ctx *gin.Context) {
	type ParamsValidate struct {
		Identifie string `form:"identifie" binding:"required"`
		Name      string `form:"name"`
	}

	params := ParamsValidate{}
	if !self.Validate(ctx, &params) {
		return
	}

	user := logic2.User{}.GetUser(ctx)
	if user == nil {
		self.JsonResponseWithError(ctx, errors.New("用户信息异常"), 401)
		return
	}
	formulaQuery := dao.Q.Formula.Where(dao.Q.Formula.Name.Eq(params.Identifie))
	if !(logic2.User{}).IsAdminUser(user) {
		formulaQuery = formulaQuery.Where(dao.Q.Formula.UserID.Eq(user.ID))
	}
	formula, err := formulaQuery.First()
	if err != nil || formula == nil {
		self.JsonResponseWithError(ctx, errors.New("制品不存在或无权操作"), 404)
		return
	}

	if _, err := (logic.Category{}).SaveFormulaCategory(formula.ID, params.Name); err != nil {
		self.JsonResponseWithServerError(ctx, err)
		return
	}

	self.JsonSuccessResponse(ctx)
	return
}

func (self Category) List(ctx *gin.Context) {
	type ParamsValidate struct {
		Limit int `form:"limit" binding:"omitempty"`
	}

	params := ParamsValidate{}
	if !self.Validate(ctx, &params) {
		return
	}

	categories, err := (logic.Category{}).List(params.Limit)
	if err != nil {
		self.JsonResponseWithServerError(ctx, err)
		return
	}
	self.JsonResponseWithoutError(ctx, gin.H{
		"list": categories,
	})
	return
}
