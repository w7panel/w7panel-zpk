package controller

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/app/respo/logic"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/w7panel/w7panel-zpk/common/service/w7/devcenter"
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

func (c Version) Publish(ctx *gin.Context) {
	type ParamsValidate struct {
		Identifie string `form:"identifie" binding:"required"`
		Version   string `form:"version" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	depotLogin := c.getDepot()
	formula, err := depotLogin.GetFormula(params.Identifie, params.Version, logic2.User{}.GetUser(ctx))
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}

	consoleUid := logic2.User{}.GetConsoleUid(ctx)
	err = c.publishFormula(consoleUid, formula)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	c.JsonResponseWithoutError(ctx, gin.H{
		"status":  logic.SYNC_STATUS_PROCESS,
		"message": "发起打包成功",
	})
	return
}

func (c Version) Unpublish(ctx *gin.Context) {
	type ParamsValidate struct {
		Identifie string `form:"identifie" json:"identifie" binding:"required"`
		Version   string `form:"version" json:"version" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	depotLogin := c.getDepot()
	formula, err := depotLogin.GetFormula(params.Identifie, params.Version, logic2.User{}.GetUser(ctx))
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}
	if formula.LatestVersionId != formula.VersionId {
		c.JsonResponseWithError(ctx, errors.New("只能下架当前线上版本"), 500)
		return
	}

	currentVersion, err := dao.Q.Version.Where(dao.Q.Version.ID.Eq(formula.VersionId)).First()
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}
	if currentVersion.PublishStatus != logic.FormulaPublishStatusSuccess && currentVersion.PublishStatus != 0 {
		c.JsonResponseWithError(ctx, errors.New("当前版本不是已发布状态"), 500)
		return
	}

	prevVersion, err := dao.Q.Version.
		Where(dao.Q.Version.FormulaID.Eq(formula.ID)).
		Where(dao.Q.Version.ID.Lt(currentVersion.ID)).
		Where(dao.Q.Version.PublishStatus.In(logic.FormulaPublishStatusSuccess, 0)).
		Order(dao.Q.Version.ID.Desc()).
		First()
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}
	if prevVersion == nil {
		c.JsonResponseWithError(ctx, errors.New("没有可顺延的已发布版本"), 500)
		return
	}

	prevVersionFormula, err := depotLogin.GetFormula(params.Identifie, prevVersion.Name, logic2.User{}.GetUser(ctx))
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}

	err = dao.Q.Transaction(func(tx *dao.Query) error {
		_, err := tx.Version.Where(tx.Version.ID.Eq(currentVersion.ID)).Updates(entity.Version{
			PublishStatus:     -1,
			PublishFailReason: "",
		})
		if err != nil {
			return err
		}

		_, err = tx.Formula.Where(tx.Formula.ID.Eq(formula.ID)).Update(tx.Formula.VersionLatestID, prevVersion.ID)
		return err
	})
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}

	consoleUid := logic2.User{}.GetConsoleUid(ctx)
	err = c.publishFormula(consoleUid, prevVersionFormula)
	if err != nil {
		c.JsonResponseWithServerError(ctx, errors.New("顺延版本发布失败, 请尝试手动发布，err:"+err.Error()))
		return
	}

	c.JsonSuccessResponse(ctx)
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
			CreatedAt: c.formatVersionCreatedAt(result[i].CreatedAt),
		})
	}

	c.JsonResponseWithoutError(ctx, gin.H{
		"total": total,
		"limit": params.Limit,
		"page":  params.Page,
		"list":  list,
	})
}

func (c Version) publishFormula(consoleUid int32, formula *logic.Formula) error {
	// 如果没有zip包，只有镜像时，不需要打包发布
	// 将文件打包到 Storage 目录，需要同步的再进行同步
	err := c.getDepot().Pack(formula, false)
	if err != nil {
		return err
	}

	if formula.Setting != nil && formula.Setting.SupportAutoPublishToZpkMarket {
		if consoleUid <= 0 {
			return errors.New("请先在面板绑定微擎云端账号")
		}
		if formula.ConsoleUid > 0 && formula.ConsoleUid != consoleUid {
			return errors.New("请先在面板绑定微擎云端账号")
		}
		err = logic.FormulaGoods{}.PublishGoods(formula, devcenter.PublishGoodsReq{
			ConsoleUid: int(consoleUid),
		})
		if err != nil {
			return err
		}
	}

	return logic.AddFormulaPublishTask(formula.Name, formula.Version, formula.VersionId)
}

func (c Version) formatVersionCreatedAt(createdAt time.Time) string {
	if createdAt.IsZero() {
		return ""
	}
	return createdAt.Format("2006-01-02 15:04:05")
}
