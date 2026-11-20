package logic

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/w7panel/w7panel-zpk/modules/document/common/dao"
	"github.com/w7panel/w7panel-zpk/modules/document/common/entity"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/permission"
	"github.com/w7panel/w7panel-zpk/modules/document/common/types"
)

type Category struct{ Q *dao.Query }

func (l Category) CreateCategory(ctx context.Context, a types.Actor, id, name, description string) error {
	if !a.Principal.Admin {
		return permission.ErrDenied
	}
	if !identifiePattern.MatchString(id) || strings.TrimSpace(name) == "" || len(name) > 255 || len(description) > 4096 {
		return ErrInvalid
	}
	c := entity.DocumentCategory{Identifie: id, Name: name, Description: description, UserID: a.Principal.ID, Revision: 1}
	if e := l.Q.DocumentCategory.WithContext(ctx).Create(&c); e != nil {
		return ErrConflict
	}
	return nil
}

func (l Category) Categories(ctx context.Context, a types.Actor) ([]*entity.DocumentCategory, error) {
	rows, e := l.Q.DocumentCategory.WithContext(ctx).Order(l.Q.DocumentCategory.ID).Find()
	if e != nil {
		return nil, e
	}
	return rows, nil
}

func (l Category) EditCategory(ctx context.Context, a types.Actor, id, name, description string, revision int32) error {
	if !a.Principal.Admin {
		return permission.ErrDenied
	}
	if name == "" || len(name) > 255 || len(description) > 4096 || revision <= 0 {
		return ErrInvalid
	}
	q := l.Q.DocumentCategory
	r, e := q.WithContext(ctx).Where(q.Identifie.Eq(id), q.Revision.Eq(revision)).Updates(map[string]interface{}{"name": name, "description": description, "revision": revision + 1, "updated_at": time.Now()})
	if e != nil {
		return e
	}
	if r.RowsAffected != 1 {
		return ErrConflict
	}
	return nil
}

func (l Category) DeleteCategory(ctx context.Context, a types.Actor, id string) error {
	if !a.Principal.Admin {
		return permission.ErrDenied
	}
	if id == "applications" {
		return errors.New("默认分类不可删除")
	}
	return l.Q.Transaction(func(tx *dao.Query) error {
		count, e := tx.DocumentLibrary.WithContext(ctx).Where(tx.DocumentLibrary.Category.Eq(id)).Count()
		if e != nil {
			return e
		}
		if count > 0 {
			return errors.New("分类下仍有文档库")
		}
		_, e = tx.DocumentCategory.WithContext(ctx).Where(tx.DocumentCategory.Identifie.Eq(id)).Delete()
		return e
	})
}
