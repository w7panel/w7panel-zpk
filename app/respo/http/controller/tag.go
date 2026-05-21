package controller

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/app/respo/logic"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
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

	formula := logic.GetFormulaByName(params.Identifie)
	if formula == nil {
		self.JsonResponseWithError(ctx, errors.New("制品不存在"), 500)
		return
	}

	tag, _ := dao.Q.Tag.Where(dao.Q.Tag.Name.Eq(params.Name)).First()
	if tag == nil {
		tag = &entity.Tag{
			Name: params.Name,
		}
		_ = dao.Q.Tag.Create(tag)
	}

	formulaTag, _ := dao.TagFormula.Where(dao.TagFormula.Where(dao.TagFormula.TagID.Eq(tag.ID),
		dao.TagFormula.FormulaID.Eq(formula.ID)),
	).First()

	if formulaTag == nil {
		_ = dao.TagFormula.Create(&entity.TagFormula{
			FormulaID: formula.ID,
			TagID:     tag.ID,
		})
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

	tag, _ := dao.Tag.Where(dao.Tag.ID.Eq(int32(params.TagId))).First()
	if tag == nil {
		self.JsonResponseWithError(ctx, errors.New("标签不存在"), 500)
		return
	}

	_, _ = dao.Q.TagFormula.Where(dao.Q.TagFormula.TagID.Eq(tag.ID), dao.Q.TagFormula.FormulaID.Eq(int32(params.FormulaId))).Delete()
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
