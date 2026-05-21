package controller

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/app/respo/logic"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	"github.com/w7panel/w7panel-zpk/common/service/w7"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
)

type FormulaAudit struct {
	Abstract
}

func (c FormulaAudit) List(ctx *gin.Context) {
	type ParamsValidate struct {
		PageSize    int     `form:"limit,default=30" json:"limit" binding:"omitempty"`
		Page        int     `form:"page,default=1" json:"page" binding:"omitempty,gt=0"`
		Keyword     string  `form:"keyword" json:"keyword" binding:"omitempty"`
		AuditStatus []int32 `form:"audit_status" json:"audit_status"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 10
	}

	query := dao.Q.Formula.Preload(dao.Formula.Version)
	if len(params.AuditStatus) > 0 {
		query = query.Where(dao.Q.Formula.AuditStatus.In(params.AuditStatus...))
	}

	if params.Keyword != "" {
		query = query.Where(dao.Q.Formula.Title.Like("%" + params.Keyword + "%"))
	}

	type ResultNode struct {
		Name                       string          `json:"name"`
		Description                string          `json:"description"`
		Identifie                  string          `json:"identifie"`
		Icon                       string          `json:"icon"`
		Version                    *entity.Version `json:"version"`
		AuditStatus                int32           `json:"audit_status"`
		AuditRemark                string          `json:"audit_remark"`
		PublishOfficialStoreStatus int32           `json:"publish_official_store_status"`
		RemoteFormulaInfoURL       string          `json:"remote_formula_info_url"`
	}
	var result []ResultNode
	formulaList, total, _ := query.Order(dao.Q.Formula.CreatedAt.Desc()).FindByPage((params.Page-1)*params.PageSize, params.PageSize)
	if formulaList != nil {
		depot := c.getDepot()
		for _, item := range formulaList {
			formula, err := depot.GetFormula(item.Name, "", nil)
			if err == nil {
				result = append(result, ResultNode{
					Name:                       item.Title,
					Description:                formula.Manifest.Application.Description,
					Identifie:                  item.Name,
					Icon:                       formula.Icon,
					Version:                    item.Version,
					AuditStatus:                item.AuditStatus,
					AuditRemark:                item.AuditRemark,
					PublishOfficialStoreStatus: item.PublishOfficialStoreStatus,
					RemoteFormulaInfoURL:       item.RemoteFormulaInfoURL,
				})
			}
		}
	}

	c.JsonResponseWithoutError(ctx, gin.H{
		"total":                   total,
		"limit":                   params.PageSize,
		"page":                    params.Page,
		"list":                    result,
		"can_push_official_store": !strings.Contains(facade.GetConfig().GetString("setting.depot.external_domain"), "zpk.w7.cc"),
		"webUrl":                  "https://" + facade.GetConfig().GetString("setting.depot.external_domain") + "/zpk",
	})
}

func (c FormulaAudit) Audit(ctx *gin.Context) {
	type ParamsValidate struct {
		Identifie   string `form:"identifie" json:"identifie" binding:"required"`
		AuditStatus int    `form:"audit_status" json:"audit_status" binding:"required,oneof=1 2 3"`
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

	if formula.RemoteFormulaInfoUrl != "" && params.AuditStatus != logic.FOEMULA_AUDIT_ING {
		err = w7.ZpkSdk.NotifyFormulaAuditResult(formula.Name, formula.RemoteFormulaInfoUrl, params.AuditStatus, params.AuditRemark)
		if err != nil {
			c.JsonResponseWithServerError(ctx, err)
			return
		}
	}

	_, err = dao.Q.Formula.Where(dao.Q.Formula.ID.Eq(formula.ID)).Updates(entity.Formula{
		AuditStatus: int32(params.AuditStatus),
		AuditRemark: params.AuditRemark,
	})
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	c.JsonSuccessResponse(ctx)
}
