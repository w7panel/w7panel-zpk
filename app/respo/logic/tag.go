package logic

import (
	"errors"
	"strings"

	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	"github.com/w7panel/w7panel-zpk/common/service/w7"
	zpk_market "github.com/w7panel/w7panel-zpk/common/service/w7/zpk-market"
	"gorm.io/gorm"
)

type Category struct {
}

func (l Category) List(limit int) ([]zpk_market.FormulaCategory, error) {
	return w7.ZpkMarketSdk.GetFormulaCategories(limit)
}

func (l Category) SaveFormulaCategory(formulaID int32, name string) (*entity.Tag, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		_, err := dao.Q.TagFormula.Where(dao.Q.TagFormula.FormulaID.Eq(formulaID)).Delete()
		return nil, err
	}

	categories, err := l.List(999)
	if err != nil {
		return nil, err
	}
	categoryExists := false
	for _, category := range categories {
		if category.Name == name {
			categoryExists = true
			break
		}
	}
	if !categoryExists {
		return nil, errors.New("分类不存在")
	}

	var selected *entity.Tag
	err = dao.Q.Transaction(func(tx *dao.Query) error {
		selected, err = tx.Tag.Where(tx.Tag.Name.Eq(name)).First()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			selected = &entity.Tag{Name: name}
			if err = tx.Tag.Create(selected); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}

		if _, err = tx.TagFormula.Where(tx.TagFormula.FormulaID.Eq(formulaID)).Delete(); err != nil {
			return err
		}
		return tx.TagFormula.Create(&entity.TagFormula{
			FormulaID: formulaID,
			TagID:     selected.ID,
		})
	})
	return selected, err
}

func (l Category) Sync() error {
	categories, err := l.List(999)
	if err != nil {
		return err
	}
	for _, category := range categories {
		name := strings.TrimSpace(category.Name)
		if name == "" {
			continue
		}
		_, err = dao.Q.Tag.Where(dao.Q.Tag.Name.Eq(name)).First()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err = dao.Q.Tag.Create(&entity.Tag{Name: name}); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
	}
	return nil
}
