package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/app/respo/logic"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	"github.com/w7panel/w7panel-zpk/common/service/w7/zpk"
)

type FormulaRemote struct {
	Abstract
}

func (c FormulaRemote) AddRemote(ctx *gin.Context) {
	params := zpk.Formula{}
	if !c.Validate(ctx, &params) {
		return
	}

	err := logic.FormulaRemote{}.AddFormulaFromRemote(ctx, params)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	c.JsonSuccessResponse(ctx)
}

func (c FormulaRemote) PushToOfficialZpkStore(ctx *gin.Context) {
	type ParamsValidate struct {
		Identifie string `form:"identifie" json:"identifie" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	formula, err := c.getDepot().GetFormula(params.Identifie, "", nil)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	err = logic.FormulaRemote{}.PushFormulaToOfficialStore(formula)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	c.JsonSuccessResponse(ctx)
}

func (c FormulaRemote) OfficialZpkStoreAuditNotify(ctx *gin.Context) {
	type ParamsValidate struct {
		Identifie   string `form:"identifie" json:"identifie" binding:"required"`
		AuditStatus int    `form:"audit_status" json:"audit_status" binding:"required"`
		AuditRemark string `form:"audit_remark" json:"audit_remark"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	formula, err := c.getDepot().GetFormula(params.Identifie, "", nil)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	_, err = dao.Q.Formula.Where(dao.Q.Formula.ID.Eq(formula.ID)).Updates(entity.Formula{
		PublishOfficialStoreStatus: int32(params.AuditStatus),
		AuditRemark:                params.AuditRemark,
	})
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	c.JsonSuccessResponse(ctx)
}
