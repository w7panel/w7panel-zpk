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

type Category struct {
	Q           *dao.Query
	Permissions permission.Service
}

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
	return l.Permissions.Replace(ctx, a.Token, []permission.Grant{{UserID: a.Principal.ID, Resource: permission.CategoryResource(id), Actions: []string{"read", "write", "create", "delete", "manage_permissions"}}})
}

func (l Category) Categories(ctx context.Context, a types.Actor) ([]*entity.DocumentCategory, error) {
	rows, e := l.Q.DocumentCategory.WithContext(ctx).Order(l.Q.DocumentCategory.ID).Find()
	if e != nil {
		return nil, e
	}
	result := make([]*entity.DocumentCategory, 0, len(rows))
	for _, c := range rows {
		v, e := l.Permissions.Check(ctx, a.Token, []permission.Resource{permission.CategoryResource(c.Identifie)}, "read")
		if e != nil {
			return nil, e
		}
		if a.Principal.Admin || c.UserID == a.Principal.ID || v.Allowed {
			result = append(result, c)
		}
	}
	return result, nil
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

func (l Category) SetPermission(ctx context.Context, actor types.Actor, identifie string, targetUserID int32, actions []string) error {
	if !actor.Principal.Admin {
		return permission.ErrDenied
	}
	if targetUserID <= 0 {
		return errors.New("被授权用户无效")
	}
	query := l.Q.DocumentCategory
	if _, err := query.WithContext(ctx).Where(query.Identifie.Eq(identifie)).First(); err != nil {
		return err
	}
	return l.Permissions.Replace(ctx, actor.Token, []permission.Grant{{UserID: targetUserID, Resource: permission.CategoryResource(identifie), Actions: actions}})
}
