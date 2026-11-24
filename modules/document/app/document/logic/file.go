package logic

import (
	"context"
	"errors"
	"io"
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

type File struct{}

type FileInfo struct {
	Path      string `json:"path"`
	Title     string `json:"title"`
	Size      int64  `json:"size"`
	Position  int32  `json:"position"`
	Protected int32  `json:"protected"`
	CanWrite  bool   `json:"can_write"`
	CanDelete bool   `json:"can_delete"`
}

type Tree struct {
	Files     []FileInfo `json:"files"`
	CanCreate bool       `json:"can_create"`
	Name      string     `json:"name"`
}

func (l File) List(ctx context.Context, user zpk.User, ref string) (Tree, error) {
	c, e := (Library{}).Get(ctx, ref)
	if e != nil {
		return Tree{}, e
	}
	if err := permission.Require(service.PermissionSdk.CanRead(ctx, user, *c)); err != nil {
		return Tree{}, err
	}
	files, e := dao.Q.DocumentFile.WithContext(ctx).Where(dao.Q.DocumentFile.LibraryID.Eq(c.ID)).Order(dao.Q.DocumentFile.Position, dao.Q.DocumentFile.ID).Find()
	if e != nil {
		return Tree{}, e
	}
	out := Tree{Files: []FileInfo{}, Name: c.Name}
	out.CanCreate, e = service.PermissionSdk.CanManage(ctx, user, *c)
	if e != nil {
		return Tree{}, e
	}
	for _, file := range files {
		canChange := out.CanCreate && file.Protected == 0
		out.Files = append(out.Files, FileInfo{file.Path, file.Title, file.Size, file.Position, file.Protected, canChange, canChange})
	}
	return out, nil
}

func (l File) get(ctx context.Context, c int32, p string) (*entity.DocumentFile, error) {
	q := dao.Q.DocumentFile
	return q.WithContext(ctx).Where(q.LibraryID.Eq(c), q.Path.Eq(p)).First()
}

func (l File) Read(ctx context.Context, user zpk.User, ref, p string) (*entity.DocumentFile, []byte, error) {
	c, e := (Library{}).Get(ctx, ref)
	if e != nil {
		return nil, nil, e
	}
	if err := permission.Require(service.PermissionSdk.CanRead(ctx, user, *c)); err != nil {
		return nil, nil, err
	}
	f, e := l.get(ctx, c.ID, p)
	if e != nil {
		return nil, nil, e
	}
	r, e := service.DocumentStorage.Open(ctx, f.StorageKey)
	if e != nil {
		return nil, nil, e
	}
	defer r.Close()
	b, e := io.ReadAll(io.LimitReader(r, MaxContentSize+1))
	if len(b) > MaxContentSize {
		return nil, nil, ErrInvalid
	}
	return f, b, e
}

// Save creates or overwrites a document; empty content is valid, never deletion.
func (l File) Save(ctx context.Context, user zpk.User, ref, p, title string, content []byte, protected *int32) (*entity.DocumentFile, error) {
	c, e := (Library{}).Get(ctx, ref)
	if e != nil {
		return nil, e
	}
	if err := permission.Require(service.PermissionSdk.CanManage(ctx, user, *c)); err != nil {
		return nil, err
	}
	key, e := service.DocumentStorage.Put(ctx, content)
	if e != nil {
		return nil, e
	}
	if title == "" {
		title = path.Base(p)
	}
	var file *entity.DocumentFile
	e = dao.Q.Transaction(func(tx *dao.Query) error {
		library, err := (Library{}).touch(ctx, tx, ref)
		if err != nil {
			return err
		}
		if err := permission.Require(service.PermissionSdk.CanManage(ctx, user, *library)); err != nil {
			return err
		}
		query := tx.DocumentFile
		file, err = query.WithContext(ctx).Unscoped().Where(query.LibraryID.Eq(library.ID), query.Path.Eq(p)).First()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			file = &entity.DocumentFile{LibraryID: library.ID, Path: p, Title: title, StorageKey: key, Size: int64(len(content)), Position: 1000}
			if protected != nil {
				file.Protected = *protected
			}
			return query.WithContext(ctx).Create(file)
		}
		if err != nil {
			return err
		}
		if file.Protected == 1 && protected == nil {
			return permission.ErrDenied
		}
		updates := entity.DocumentFile{StorageKey: key, Size: int64(len(content)), Title: title, UpdatedAt: time.Now()}
		fields := []field.Expr{query.StorageKey, query.Size, query.Title, query.UpdatedAt}
		if protected != nil {
			updates.Protected = *protected
			fields = append(fields, query.Protected)
		}
		write := query.WithContext(ctx).Where(query.ID.Eq(file.ID))
		if file.DeletedAt.Valid {
			write = write.Unscoped()
			updates.Position = 1000
			fields = append(fields, query.DeletedAt, query.Position)
		}
		result, err := write.Select(fields...).Updates(updates)
		if err != nil {
			return err
		}
		if result.RowsAffected == 0 {
			if _, err := query.WithContext(ctx).Select(query.ID).Where(query.ID.Eq(file.ID)).First(); err != nil {
				return err
			}
		}
		file.Title, file.StorageKey, file.Size = title, key, int64(len(content))
		if protected != nil {
			file.Protected = *protected
		}
		if file.DeletedAt.Valid {
			file.DeletedAt = gorm.DeletedAt{}
			file.Position = 1000
		}
		return nil
	})
	if e != nil {
		return nil, e
	}
	return file, nil
}

func (l File) Delete(ctx context.Context, user zpk.User, ref, p string) error {
	return dao.Q.Transaction(func(tx *dao.Query) error {
		library, err := (Library{}).touch(ctx, tx, ref)
		if err != nil {
			return err
		}
		if err := permission.Require(service.PermissionSdk.CanManage(ctx, user, *library)); err != nil {
			return err
		}
		q := tx.DocumentFile
		file, err := q.WithContext(ctx).Where(q.LibraryID.Eq(library.ID), q.Path.Eq(p)).First()
		if err != nil {
			return err
		}
		if file.Protected == 1 {
			return permission.ErrDenied
		}
		r, err := q.WithContext(ctx).Where(q.ID.Eq(file.ID)).Delete()
		if err != nil {
			return err
		}
		if r.RowsAffected != 1 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (l File) Rename(ctx context.Context, user zpk.User, ref, p, title string) error {
	// Rename only the display title; file paths remain stable.
	return dao.Q.Transaction(func(tx *dao.Query) error {
		library, err := (Library{}).touch(ctx, tx, ref)
		if err != nil {
			return err
		}
		if err := permission.Require(service.PermissionSdk.CanManage(ctx, user, *library)); err != nil {
			return err
		}
		q := tx.DocumentFile
		file, err := q.WithContext(ctx).Where(q.LibraryID.Eq(library.ID), q.Path.Eq(p)).First()
		if err != nil {
			return err
		}
		if file.Protected == 1 {
			return permission.ErrDenied
		}
		result, err := q.WithContext(ctx).Where(q.ID.Eq(file.ID)).Select(q.Title, q.UpdatedAt).Updates(entity.DocumentFile{Title: title, UpdatedAt: time.Now()})
		if err == nil && result.RowsAffected == 0 {
			_, err = q.WithContext(ctx).Select(q.ID).Where(q.LibraryID.Eq(library.ID), q.Path.Eq(p)).First()
		}
		return err
	})
}

func (l File) Order(ctx context.Context, user zpk.User, ref string, paths []string) error {
	return dao.Q.Transaction(func(tx *dao.Query) error {
		library, err := (Library{}).touch(ctx, tx, ref)
		if err != nil {
			return err
		}
		if err := permission.Require(service.PermissionSdk.CanManage(ctx, user, *library)); err != nil {
			return err
		}
		for i, p := range paths {
			q := tx.DocumentFile
			r, err := q.WithContext(ctx).Where(q.LibraryID.Eq(library.ID), q.Path.Eq(p)).Update(q.Position, int32(i))
			if err != nil {
				return err
			}
			if r.RowsAffected == 0 {
				if _, err := q.WithContext(ctx).Select(q.ID).Where(q.LibraryID.Eq(library.ID), q.Path.Eq(p)).First(); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
