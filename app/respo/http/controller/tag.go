package controller

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
)

type Tag struct {
	Abstract
}

func (self Tag) Add(ctx *gin.Context) {
	type ParamsValidate struct {
		Identifie string `form:"identifie" binding:"required"`
		Name      string `form:"name" binding:"required"`
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

	tag, _ := dao.Q.Tag.Where(dao.Q.Tag.Name.Eq(params.Name)).First()
	if tag == nil {
		tag = &entity.Tag{
			Name: params.Name,
		}
		if err := dao.Q.Tag.Create(tag); err != nil {
			self.JsonResponseWithServerError(ctx, err)
			return
		}
	}

	formulaTag, _ := dao.TagFormula.Where(dao.TagFormula.Where(dao.TagFormula.TagID.Eq(tag.ID),
		dao.TagFormula.FormulaID.Eq(formula.ID)),
	).First()

	if formulaTag == nil {
		if err := dao.TagFormula.Create(&entity.TagFormula{
			FormulaID: formula.ID,
			TagID:     tag.ID,
		}); err != nil {
			self.JsonResponseWithServerError(ctx, err)
			return
		}
	}

	self.JsonSuccessResponse(ctx)
	return
}

func (self Tag) Delete(ctx *gin.Context) {
	type ParamsValidate struct {
		TagId     int `form:"tag_id" binding:"required"`
		FormulaId int `form:"formula_id" binding:"required"`
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
	formulaQuery := dao.Q.Formula.Where(dao.Q.Formula.ID.Eq(int32(params.FormulaId)))
	if !(logic2.User{}).IsAdminUser(user) {
		formulaQuery = formulaQuery.Where(dao.Q.Formula.UserID.Eq(user.ID))
	}
	formula, err := formulaQuery.First()
	if err != nil || formula == nil {
		self.JsonResponseWithError(ctx, errors.New("制品不存在或无权操作"), 404)
		return
	}

	tag, _ := dao.Tag.Where(dao.Tag.ID.Eq(int32(params.TagId))).First()
	if tag == nil {
		self.JsonResponseWithError(ctx, errors.New("标签不存在"), 500)
		return
	}

	if _, err := dao.Q.TagFormula.Where(dao.Q.TagFormula.TagID.Eq(tag.ID), dao.Q.TagFormula.FormulaID.Eq(formula.ID)).Delete(); err != nil {
		self.JsonResponseWithServerError(ctx, err)
		return
	}
	self.JsonSuccessResponse(ctx)
}

func (self Tag) List(ctx *gin.Context) {
	type ParamsValidate struct {
		Limit int `form:"limit" binding:"omitempty"`
	}

	params := ParamsValidate{}
	if !self.Validate(ctx, &params) {
		return
	}

	tag, _ := dao.Tag.Limit(params.Limit).Find()
	self.JsonResponseWithoutError(ctx, gin.H{
		"list": tag,
	})
	return
}
