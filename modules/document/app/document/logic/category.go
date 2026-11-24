package logic

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/w7panel/w7panel-zpk/modules/document/common/dao"
	"github.com/w7panel/w7panel-zpk/modules/document/common/entity"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service"
	"gorm.io/gorm"
)

type Category struct{}

func (l Category) CreateCategory(ctx context.Context, userID int32, identifie, name, description string) error {
	c := entity.DocumentCategory{Identifie: identifie, Name: name, Description: description, UserID: userID}
	query := dao.Q.DocumentCategory
	if _, err := query.WithContext(ctx).Unscoped().Select(query.ID).Where(query.Identifie.Eq(identifie)).First(); err == nil {
		return ErrConflict
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if err := service.ZPKSdk.EnsureNamespace(ctx, identifie); err != nil {
		return err
	}
	return query.WithContext(ctx).Create(&c)
}

func (l Category) Categories(ctx context.Context) ([]*entity.DocumentCategory, error) {
	rows, e := dao.Q.DocumentCategory.WithContext(ctx).Order(dao.Q.DocumentCategory.ID).Find()
	if e != nil {
		return nil, e
	}
	return rows, nil
}

func (l Category) EditCategory(ctx context.Context, identifie, newIdentifie, name, description string) error {
	if identifie == "app" && newIdentifie != identifie {
		return fmt.Errorf("%w：默认分类标识不可修改", ErrInvalid)
	}
	var renamedNamespaceID int32
	err := dao.Q.Transaction(func(tx *dao.Query) error {
		query := tx.DocumentCategory
		category, err := query.WithContext(ctx).Where(query.Identifie.Eq(identifie)).First()
		if err != nil {
			return err
		}
		if newIdentifie != identifie {
			if count, err := query.WithContext(ctx).Unscoped().Where(query.Identifie.Eq(newIdentifie)).Count(); err != nil {
				return err
			} else if count != 0 {
				return ErrConflict
			}
			namespace, err := service.ZPKSdk.GetNamespace(ctx, identifie)
			if err != nil {
				return err
			}
			if err := service.ZPKSdk.RenameNamespace(ctx, namespace.ID, newIdentifie); err != nil {
				return err
			}
			renamedNamespaceID = namespace.ID
			library := tx.DocumentLibrary
			if _, err := library.WithContext(ctx).Unscoped().Where(library.Category.Eq(identifie)).Update(library.Category, newIdentifie); err != nil {
				return err
			}
		}
		_, err = query.WithContext(ctx).Where(query.ID.Eq(category.ID)).
			Select(query.Identifie, query.Name, query.Description, query.UpdatedAt).Updates(entity.DocumentCategory{
			Identifie: newIdentifie, Name: name, Description: description, UpdatedAt: time.Now(),
		})
		return err
	})
	if err != nil && renamedNamespaceID != 0 {
		return errors.Join(err, service.ZPKSdk.RenameNamespace(ctx, renamedNamespaceID, identifie))
	}
	return err
}

func (l Category) DeleteCategory(ctx context.Context, identifie string) error {
	if identifie == "app" {
		return fmt.Errorf("%w：默认分类不可删除", ErrInvalid)
	}
	return dao.Q.Transaction(func(tx *dao.Query) error {
		query := tx.DocumentCategory
		category, err := query.WithContext(ctx).Where(query.Identifie.Eq(identifie)).First()
		if err != nil {
			return err
		}
		count, e := tx.DocumentLibrary.WithContext(ctx).Where(tx.DocumentLibrary.Category.Eq(identifie)).Count()
		if e != nil {
			return e
		}
		if count > 0 {
			return fmt.Errorf("%w：分类下仍有文档库", ErrInvalid)
		}
		namespace, err := service.ZPKSdk.GetNamespace(ctx, identifie)
		if err != nil {
			return err
		}
		if err := service.ZPKSdk.DeleteNamespace(ctx, namespace.ID); err != nil {
			return err
		}
		result, e := query.WithContext(ctx).Where(query.ID.Eq(category.ID)).Delete()
		if e == nil && result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return e
	})
}
