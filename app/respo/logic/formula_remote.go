package logic

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/w7panel/w7panel-zpk/common/service/w7"
	"github.com/w7panel/w7panel-zpk/common/service/w7/zpk"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
)

type FormulaRemote struct {
}

func (l FormulaRemote) AddFormulaFromRemote(ctx *gin.Context, formula zpk.Formula) error {
	formulaEntity := GetFormulaByName(formula.Identifie)
	if formulaEntity != nil && formulaEntity.RemoteFormulaInfoURL != formula.RemoteFormulaInfoUrl {
		return errors.New("该制品已存在，请修改制品标识后重试")
	}
	var tags []*entity.Tag
	if len(formula.Tags) > 0 {
		var err error
		tags, err = dao.Q.Tag.Where(dao.Q.Tag.Name.In(formula.Tags...)).Find()
		if err != nil {
			return err
		}
		if len(tags) != len(formula.Tags) {
			return errors.New("制品 tags 非法")
		}
	}

	if formulaEntity == nil {
		depotLogin, _ := NewDepot()
		err := depotLogin.AddFormula(formula.Identifie, formula.LatestVersion, logic2.User{}.GetUser(ctx))
		if err != nil {
			return err
		}
		formulaEntity = GetFormulaByName(formula.Identifie)
	}
	if formulaEntity == nil {
		return errors.New("添加制品失败")
	}

	updateFormula := entity.Formula{
		Title:                formula.Title,
		RemoteFormulaInfoURL: formula.RemoteFormulaInfoUrl,
	}
	if formulaEntity.AuditStatus == FORMULA_AUDIT_FAIL {
		updateFormula.AuditStatus = FOEMULA_AUDIT_ING
	}
	_, err := dao.Q.Formula.Where(dao.Q.Formula.ID.Eq(formulaEntity.ID)).Updates(updateFormula)
	if err != nil {
		return err
	}

	var formulaTags []*entity.TagFormula
	if len(formula.Tags) > 0 {
		for _, item := range tags {
			formulaTags = append(formulaTags, &entity.TagFormula{
				TagID:     item.ID,
				FormulaID: formulaEntity.ID,
			})
		}
	}

	err = dao.Q.Transaction(func(tx *dao.Query) error {
		_, err = tx.Version.Where(tx.Version.FormulaID.Eq(formulaEntity.ID)).Update(tx.Version.Name, formula.LatestVersion)
		if err != nil {
			return err
		}

		if len(formula.Tags) > 0 {
			_, err = tx.TagFormula.Where(tx.TagFormula.FormulaID.Eq(formulaEntity.ID)).Delete()
			if err != nil {
				return err
			}

			err = tx.TagFormula.CreateInBatches(formulaTags, 20)
			if err != nil {
				return err
			}
		}

		return nil
	})

	return nil
}

func (l FormulaRemote) PushFormulaToOfficialStore(formula *Formula) error {
	if formula.AuditStatus != FORMULA_AUDIT_SUCCESS {
		return errors.New("该制品未审核通过")
	}

	tags := make([]string, 0)
	if formula.Tags != nil {
		for _, tag := range formula.Tags {
			tags = append(tags, tag.Name)
		}
	}
	err := w7.ZpkSdk.PushToOfficialZpkStore(zpk.Formula{
		Identifie:            formula.Name,
		Title:                formula.Title,
		Description:          formula.Manifest.Application.Description,
		LatestVersion:        formula.Version,
		Tags:                 tags,
		RemoteFormulaInfoUrl: "https://" + facade.GetConfig().GetString("setting.depot.external_domain") + "/zpk/respo/info/" + formula.Name,
	})
	if err != nil {
		return err
	}
	if formula.PublishOfficialStoreStatus != PUSH_OFFICIAL_STORE_STATUS_AUDIT_SUCCESS {
		_, _ = dao.Q.Formula.Where(dao.Formula.ID.Eq(formula.ID)).Update(dao.Q.Formula.PublishOfficialStoreStatus, PUSH_OFFICIAL_STORE_STATUS_AUDIT_ING)
	}

	return nil
}
