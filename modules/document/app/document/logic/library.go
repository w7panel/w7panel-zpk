package logic

import (
	"context"
	"path"
	"time"

	"github.com/w7panel/w7panel-zpk/modules/document/common/dao"
	"github.com/w7panel/w7panel-zpk/modules/document/common/entity"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/permission"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/zpk"
	"gorm.io/gen/field"
	"gorm.io/gorm"
)

type Library struct{}

type CreateFile struct {
	Path      string
	Title     string
	Content   string
	Protected int32
}

type Page struct {
	List  []*entity.DocumentLibrary `json:"list"`
	Total int64                     `json:"total"`
}

func (l Library) Get(ctx context.Context, ref string) (*entity.DocumentLibrary, error) {
	category, identifie := SplitRef(ref)
	query := dao.Q.DocumentLibrary
	return query.WithContext(ctx).Where(query.Category.Eq(category), query.Identifie.Eq(identifie)).First()
}

func (l Library) Info(ctx context.Context, user zpk.User, ref string) (*entity.DocumentLibrary, error) {
	library, err := l.Get(ctx, ref)
	if err != nil {
		return nil, err
	}
	if err := permission.Require(service.PermissionSdk.CanRead(ctx, user, *library)); err != nil {
		return nil, err
	}
	return library, nil
}

func (l Library) List(ctx context.Context, user zpk.User, category string, page, limit int) (Page, error) {
	library := dao.Q.DocumentLibrary
	query := library.WithContext(ctx)
	if !user.Admin {
		query = query.Where(library.UserID.Eq(user.ID))
	}
	if category != "" {
		query = query.Where(library.Category.Eq(category))
	}
	rows, total, err := query.Order(library.ID.Desc()).FindByPage((page-1)*limit, limit)
	return Page{List: rows, Total: total}, err
}

func (l Library) Create(ctx context.Context, user zpk.User, library entity.DocumentLibrary, files []CreateFile) (*entity.DocumentLibrary, error) {
	if library.UserID == 0 {
		library.UserID = user.ID
	}
	if library.UserID != user.ID && !user.Admin {
		return nil, permission.ErrDenied
	}
	if library.UserID == user.ID && user.Username != "" {
		library.Username = user.Username
	}
	initialFiles := make([]*entity.DocumentFile, 0, len(files))
	for position, file := range files {
		key, err := service.DocumentStorage.Put(ctx, []byte(file.Content))
		if err != nil {
			return nil, err
		}
		if file.Title == "" {
			file.Title = path.Base(file.Path)
		}
		initialFiles = append(initialFiles, &entity.DocumentFile{Path: file.Path, Title: file.Title, StorageKey: key, Size: int64(len(file.Content)), Position: int32(position), Protected: file.Protected})
	}
	err := dao.Q.Transaction(func(tx *dao.Query) error {
		if _, err := tx.DocumentCategory.WithContext(ctx).Where(tx.DocumentCategory.Identifie.Eq(library.Category)).First(); err != nil {
			return err
		}
		query := tx.DocumentLibrary
		count, err := query.WithContext(ctx).Unscoped().Where(query.Category.Eq(library.Category), query.Identifie.Eq(library.Identifie)).Count()
		if err != nil {
			return err
		}
		if count > 0 {
			return ErrConflict
		}
		if err := query.WithContext(ctx).Create(&library); err != nil {
			return err
		}
		for _, file := range initialFiles {
			file.LibraryID = library.ID
		}
		if len(initialFiles) > 0 {
			if err := tx.DocumentFile.WithContext(ctx).Create(initialFiles...); err != nil {
				return err
			}
		}
		if err := service.PermissionSdk.SetResource(ctx, permission.LibraryResource(library), library.UserID, libraryVisibleType(library.Public)); err != nil {
			return err
		}
		return service.PermissionSdk.AddOwner(ctx, library)
	})
	return &library, err
}

func (l Library) Edit(ctx context.Context, user zpk.User, ref, name, description string, public *int32) error {
	category, identifie := SplitRef(ref)
	return dao.Q.Transaction(func(tx *dao.Query) error {
		query := tx.DocumentLibrary
		library, err := query.WithContext(ctx).Where(query.Category.Eq(category), query.Identifie.Eq(identifie)).First()
		if err != nil {
			return err
		}
		if err := permission.Require(service.PermissionSdk.CanManage(ctx, user, *library)); err != nil {
			return err
		}
		updates := entity.DocumentLibrary{Name: name, Description: description, UpdatedAt: time.Now()}
		fields := []field.Expr{query.Name, query.Description, query.UpdatedAt}
		if public != nil {
			updates.Public = *public
			library.Public = *public
			fields = append(fields, query.Public)
		}
		result, err := query.WithContext(ctx).Where(query.ID.Eq(library.ID)).Select(fields...).Updates(updates)
		if err == nil && result.RowsAffected == 0 {
			_, err = query.WithContext(ctx).Select(query.ID).Where(query.ID.Eq(library.ID)).First()
		}
		if err == nil && public != nil {
			err = service.PermissionSdk.SetResource(ctx, permission.LibraryResource(*library), library.UserID, libraryVisibleType(library.Public))
		}
		return err
	})
}

func (l Library) Delete(ctx context.Context, user zpk.User, ref string) error {
	category, identifie := SplitRef(ref)
	return dao.Q.Transaction(func(tx *dao.Query) error {
		query := tx.DocumentLibrary
		library, err := query.WithContext(ctx).Where(query.Category.Eq(category), query.Identifie.Eq(identifie)).First()
		if err != nil {
			return err
		}
		if library.Protected == 1 {
			return permission.ErrDenied
		}
		if err := permission.Require(service.PermissionSdk.CanManage(ctx, user, *library)); err != nil {
			return err
		}
		result, err := query.WithContext(ctx).Where(query.ID.Eq(library.ID), query.Protected.Eq(0)).Delete()
		if err != nil {
			return err
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		file := tx.DocumentFile
		_, err = file.WithContext(ctx).Where(file.LibraryID.Eq(library.ID)).Delete()
		if err != nil {
			return err
		}
		return service.PermissionSdk.ClearLibrary(ctx, *library)
	})
}

func (l Library) touch(ctx context.Context, tx *dao.Query, ref string) (*entity.DocumentLibrary, error) {
	category, identifie := SplitRef(ref)
	query := tx.DocumentLibrary
	_, err := query.WithContext(ctx).Where(query.Category.Eq(category), query.Identifie.Eq(identifie)).Update(query.UpdatedAt, time.Now())
	if err != nil {
		return nil, err
	}
	return query.WithContext(ctx).Where(query.Category.Eq(category), query.Identifie.Eq(identifie)).First()
}

func libraryVisibleType(public int32) int32 {
	if public == 1 {
		return permission.VisibleTypePublicRead
	}
	return permission.VisibleTypePrivate
}
