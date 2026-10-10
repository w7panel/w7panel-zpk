package controller

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	formulalogic "github.com/w7panel/w7panel-zpk/app/respo/logic/formula"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"gorm.io/gen/field"
)

type formulaSelectableListItem struct {
	Name            string          `json:"name"`
	Identifie       string          `json:"identifie"`
	Version         *entity.Version `json:"version"`
	InstallOnlyOnce bool            `json:"install_only_once"`
	GoodsID         int32           `json:"goods_id"`
}

func selectableListPageBounds(page, limit, total int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 8
	}
	start := (page - 1) * limit
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}
	return start, end
}

// SelectableList returns the lightweight, install-once formula projection
// used by selectors for external dependencies and child applications.
func (c Formula) SelectableList(ctx *gin.Context) {
	type paramsValidate struct {
		Limit   int     `form:"limit,default=8" binding:"omitempty"`
		Page    int     `form:"page,default=1" binding:"omitempty,gt=0"`
		Keyword string  `form:"keyword" binding:"omitempty"`
		Status  []int32 `form:"status"`
	}
	params := paramsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	if params.Page < 1 {
		params.Page = 1
	}
	if params.Limit < 1 {
		params.Limit = 8
	}
	if len(params.Status) == 0 {
		params.Status = []int32{formulalogic.FORMULA_DISPLAY, formulalogic.FORMULA_RECOMMEND}
	}

	query := dao.Q.Formula.Preload(dao.Formula.Version).
		Where(dao.Q.Formula.Status.In(params.Status...)).
		Order(dao.Q.Formula.ID.Desc())
	if params.Keyword != "" {
		pattern := "%" + params.Keyword + "%"
		query = query.Where(field.Or(
			dao.Q.Formula.Title.Like(pattern),
			dao.Q.Formula.Name.Like(pattern),
		))
	}
	formulaList, err := query.Find()
	if err != nil {
		c.JsonResponseWithError(ctx, err, http.StatusInternalServerError)
		return
	}

	depot := c.getDepot()
	items := make([]formulaSelectableListItem, 0, len(formulaList))
	for _, formula := range formulaList {
		installOnlyOnce, metadataErr := depot.FormulaInstallOnlyOnce(formula)
		if metadataErr != nil {
			slog.Warn("load selectable formula metadata failed", "formula", formula.Name, "err", metadataErr)
			continue
		}
		if !installOnlyOnce {
			continue
		}
		items = append(items, formulaSelectableListItem{
			Name:            formula.Title,
			Identifie:       formula.Name,
			Version:         formula.Version,
			InstallOnlyOnce: true,
			GoodsID:         formula.GoodsID,
		})
	}

	total := len(items)
	start, end := selectableListPageBounds(params.Page, params.Limit, total)
	c.JsonResponseWithoutError(ctx, gin.H{
		"total":  total,
		"limit":  params.Limit,
		"page":   params.Page,
		"list":   items[start:end],
		"webUrl": "https://" + facade.GetConfig().GetString("setting.depot.external_domain") + "/zpk",
	})
}
