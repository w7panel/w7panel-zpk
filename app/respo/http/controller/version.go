package controller

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/app/respo/logic"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
	"gorm.io/gen"
)

type Version struct {
	Abstract
}

func (c Version) Add(ctx *gin.Context) {
	type ParamsValidate struct {
		Identifie   string `form:"identifie" binding:"required"`
		Version     string `form:"version" binding:"required,semver"`
		Description string `form:"description" binding:"omitempty"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	user := logic2.User{}.GetUser(ctx)
	depot := c.getDepot()
	srcFormula, err := depot.GetFormula(params.Identifie, "", user)
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}

	var versionRow *entity.Version
	where := []gen.Condition{
		dao.Q.Version.FormulaID.Eq(srcFormula.ID),
		dao.Q.Version.Name.Eq(params.Version),
	}
	versionRow, err = dao.Q.Version.Where(where...).First()
	if versionRow != nil {
		_, err = dao.Q.Version.Where(dao.Q.Version.ID.Eq(versionRow.ID)).Update(dao.Q.Version.Description, params.Description)
		if err != nil {
			c.JsonResponseWithError(ctx, err, 500)
			return
		}
	} else {
		versionRow = &entity.Version{
			Name:          params.Version,
			Description:   params.Description,
			FormulaID:     srcFormula.ID,
			PublishStatus: -1,
		}
		err = dao.Q.Version.Create(versionRow)

		destFormula, err := depot.GetFormula(params.Identifie, params.Version, user)
		if err != nil {
			c.JsonResponseWithError(ctx, err, 500)
			return
		}
		err = depot.Copy(srcFormula, destFormula)
		if err != nil {
			c.JsonResponseWithError(ctx, err, 500)
			return
		}
	}

	c.JsonResponseWithoutError(ctx, gin.H{
		"id":      versionRow.ID,
		"version": versionRow.Name,
	})
	return
}

func (c Version) GetList(ctx *gin.Context) {
	type ParamsValidate struct {
		Limit     int    `form:"limit,default=30" json:"limit" binding:"omitempty"`
		Page      int    `form:"page,default=1" json:"page" binding:"omitempty,gt=0"`
		Identifie string `form:"identifie" json:"identifie" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	if params.Page < 1 {
		params.Page = 1
	}
	if params.Limit < 1 {
		params.Limit = 10
	}

	formula, err := c.getDepot().GetFormula(params.Identifie, "", logic2.User{}.GetUser(ctx))
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}

	result, total, _ := dao.Version.Where(dao.Version.FormulaID.Eq(formula.ID)).
		Order(dao.Version.ID.Desc()).
		FindByPage((params.Page-1)*params.Limit, params.Limit)

	type versionListItem struct {
		*entity.Version
		CreatedAt string `json:"created_at"`
	}

	list := make([]versionListItem, 0, len(result))
	for i, item := range result {
		if item.PublishStatus == 0 {
			result[i].PublishStatus = logic.FormulaPublishStatusSuccess
		}
		list = append(list, versionListItem{
			Version:   result[i],
			CreatedAt: formatVersionCreatedAt(result[i].CreatedAt),
		})
	}

	c.JsonResponseWithoutError(ctx, gin.H{
		"total": total,
		"limit": params.Limit,
		"page":  params.Page,
		"list":  list,
	})
}

func formatVersionCreatedAt(createdAt time.Time) string {
	if createdAt.IsZero() {
		return ""
	}
	return createdAt.Format("2006-01-02 15:04:05")
}
