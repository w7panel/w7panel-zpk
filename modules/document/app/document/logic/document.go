package logic

import (
	"context"
	"errors"
	"io"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/w7panel/w7panel-zpk/modules/document/common/dao"
	"github.com/w7panel/w7panel-zpk/modules/document/common/entity"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/lock"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/permission"
	"github.com/w7panel/w7panel-zpk/modules/document/common/types"
	"gorm.io/gorm"
)

var ErrConflict = errors.New("内容已更新或标识已存在，请刷新后重试")
var ErrInvalid = errors.New("参数无效")
var identifiePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,95}$`)

const MaxContentSize = 4 << 20

type Actor = types.Actor
type Document struct{}

func ValidateRef(ref string) (string, string, error) {
	parts := strings.Split(ref, "/")
	if len(parts) != 2 || !identifiePattern.MatchString(parts[0]) || !identifiePattern.MatchString(parts[1]) {
		return "", "", ErrInvalid
	}
	return parts[0], parts[1], nil
}

func ValidatePath(p string) error {
	if p == "" || len(p) > 240 || strings.ContainsAny(p, "\\\x00\r\n") || path.IsAbs(p) || path.Clean(p) != p || p == "." || p == ".." || strings.HasPrefix(p, "../") {
		return ErrInvalid
	}
	return nil
}

func (d Document) Library(ctx context.Context, ref string) (*entity.DocumentLibrary, error) {
	category, id, e := ValidateRef(ref)
	if e != nil {
		return nil, e
	}
	q := dao.Q.DocumentLibrary
	return q.WithContext(ctx).Where(q.Category.Eq(category), q.Identifie.Eq(id)).First()
}

func (d Document) category(ctx context.Context, id string) (*entity.DocumentCategory, error) {
	q := dao.Q.DocumentCategory
	return q.WithContext(ctx).Where(q.Identifie.Eq(id)).First()
}

type Page struct {
	List  []*entity.DocumentLibrary `json:"list"`
	Total int64                     `json:"total"`
}

func (d Document) List(ctx context.Context, a Actor, category string, page, limit int) (Page, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	q := dao.Q.DocumentLibrary.WithContext(ctx).Where(dao.Q.DocumentLibrary.Ready.Eq(1))
	if !a.Principal.Admin {
		q = q.Where(dao.Q.DocumentLibrary.UserID.Eq(a.Principal.ID))
	}
	if category != "" {
		q = q.Where(dao.Q.DocumentLibrary.Category.Eq(category))
	}
	rows, total, e := q.Order(dao.Q.DocumentLibrary.ID.Desc()).FindByPage((page-1)*limit, limit)
	return Page{rows, total}, e
}

// Create optionally initializes a protected library and immutable default homepage.
func (d Document) Create(ctx context.Context, a Actor, ref, name, description, homepage string, initialize bool, public int32) (*entity.DocumentLibrary, error) {
	release := lock.Acquire("library:" + ref)
	defer release()
	category, id, e := ValidateRef(ref)
	if e != nil || strings.TrimSpace(name) == "" || len(name) > 255 || len(description) > 4096 || len(homepage) > MaxContentSize || (public != 0 && public != 1) {
		return nil, ErrInvalid
	}
	if a.Principal.ID <= 0 {
		return nil, permission.ErrDenied
	}
	if _, e = d.category(ctx, category); e != nil {
		return nil, e
	}
	c, e := d.Library(ctx, ref)
	if e == nil {
		if c.UserID != a.Principal.ID || c.Ready < 0 || (!initialize && c.Ready == 1) {
			return nil, ErrConflict
		}
		if c.Ready == 1 && (!initialize || c.Protected == 1) {
			return c, nil
		}
	} else if !errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, e
	} else {
		c = &entity.DocumentLibrary{Category: category, Identifie: id, Name: name, Description: description, UserID: a.Principal.ID, Revision: 1, Public: public}
		if initialize {
			c.Protected = 1
		}
		if e = dao.Q.DocumentLibrary.WithContext(ctx).Create(c); e != nil {
			return nil, ErrConflict
		}
	}
	if initialize && c.Protected != 1 {
		if _, e = dao.Q.DocumentLibrary.WithContext(ctx).Where(dao.Q.DocumentLibrary.ID.Eq(c.ID)).Update(dao.Q.DocumentLibrary.Protected, 1); e != nil {
			return nil, e
		}
		c.Protected = 1
	}
	if initialize {
		key, e := service.DocumentStorage.Put(ctx, []byte(homepage))
		if e != nil {
			return nil, e
		}
		f := &entity.DocumentFile{LibraryID: c.ID, Path: "index.md", Title: "首页", StorageKey: key, Size: int64(len(homepage)), Revision: 1}
		count, err := dao.Q.DocumentFile.WithContext(ctx).Where(dao.Q.DocumentFile.LibraryID.Eq(c.ID), dao.Q.DocumentFile.Path.Eq("index.md")).Count()
		if err != nil {
			return nil, err
		}
		if count == 0 {
			if e = dao.Q.DocumentFile.WithContext(ctx).Create(f); e != nil {
				return nil, e
			}
		}
	}
	if _, e = dao.Q.DocumentLibrary.WithContext(ctx).Where(dao.Q.DocumentLibrary.ID.Eq(c.ID)).Update(dao.Q.DocumentLibrary.Ready, 1); e != nil {
		return nil, e
	}
	c.Ready = 1
	return c, nil
}

func (d Document) EditLibrary(ctx context.Context, a Actor, ref, name, description string, rev int32, public *int32) error {
	release := lock.Acquire("library:" + ref)
	defer release()
	c, e := d.Library(ctx, ref)
	if e != nil {
		return e
	}
	if !permission.CanManage(a.Principal, *c) {
		return permission.ErrDenied
	}
	if name == "" || len(name) > 255 || len(description) > 4096 || rev <= 0 || (public != nil && *public != 0 && *public != 1) {
		return ErrInvalid
	}
	visibility := c.Public
	if public != nil {
		visibility = *public
	}
	q := dao.Q.DocumentLibrary
	r, e := q.WithContext(ctx).Where(q.ID.Eq(c.ID), q.Revision.Eq(rev), q.Ready.Eq(1)).Updates(map[string]interface{}{"name": name, "description": description, "public": visibility, "revision": rev + 1, "updated_at": time.Now()})
	if e != nil {
		return e
	}
	if r.RowsAffected != 1 {
		return ErrConflict
	}
	return nil
}

func (d Document) DeleteLibrary(ctx context.Context, a Actor, ref string) error {
	release := lock.Acquire("library:" + ref)
	defer release()
	c, e := d.Library(ctx, ref)
	if e != nil {
		return e
	}
	if !permission.CanManage(a.Principal, *c) || c.Protected == 1 {
		return permission.ErrDenied
	}
	return dao.Q.Transaction(func(tx *dao.Query) error {
		q := tx.DocumentLibrary
		r, e := q.WithContext(ctx).Where(q.ID.Eq(c.ID), q.Revision.Eq(c.Revision), q.Ready.Eq(1)).Updates(map[string]interface{}{"ready": -1, "revision": c.Revision + 1})
		if e != nil {
			return e
		}
		if r.RowsAffected != 1 {
			return ErrConflict
		}
		return nil
	})
}

type FileInfo struct {
	Path      string `json:"path"`
	Title     string `json:"title"`
	Size      int64  `json:"size"`
	Position  int32  `json:"position"`
	Revision  int32  `json:"revision"`
	CanWrite  bool   `json:"can_write"`
	CanDelete bool   `json:"can_delete"`
}

type Tree struct {
	Files     []FileInfo `json:"files"`
	Revision  int32      `json:"revision"`
	CanCreate bool       `json:"can_create"`
	Name      string     `json:"name"`
}

func (d Document) Files(ctx context.Context, a Actor, ref string) (Tree, error) {
	c, e := d.Library(ctx, ref)
	if e != nil {
		return Tree{}, e
	}
	if c.Ready != 1 {
		return Tree{}, gorm.ErrRecordNotFound
	}
	files, e := dao.Q.DocumentFile.WithContext(ctx).Where(dao.Q.DocumentFile.LibraryID.Eq(c.ID)).Order(dao.Q.DocumentFile.Position, dao.Q.DocumentFile.ID).Find()
	if e != nil {
		return Tree{}, e
	}
	out := Tree{Files: []FileInfo{}, Revision: c.Revision, Name: c.Name}
	if !permission.CanRead(a.Principal, *c) {
		return out, permission.ErrDenied
	}
	out.CanCreate = permission.CanManage(a.Principal, *c)
	for _, file := range files {
		canChange := permission.CanChangeFile(a.Principal, *c, file.Path)
		out.Files = append(out.Files, FileInfo{file.Path, file.Title, file.Size, file.Position, file.Revision, canChange, canChange})
	}
	return out, nil
}

func (d Document) file(ctx context.Context, c int32, p string) (*entity.DocumentFile, error) {
	if e := ValidatePath(p); e != nil {
		return nil, e
	}
	q := dao.Q.DocumentFile
	return q.WithContext(ctx).Where(q.LibraryID.Eq(c), q.Path.Eq(p)).First()
}

func (d Document) Read(ctx context.Context, a Actor, ref, p string) (*entity.DocumentFile, []byte, error) {
	c, e := d.Library(ctx, ref)
	if e != nil {
		return nil, nil, e
	}
	if c.Ready != 1 {
		return nil, nil, gorm.ErrRecordNotFound
	}
	if !permission.CanRead(a.Principal, *c) {
		return nil, nil, permission.ErrDenied
	}
	f, e := d.file(ctx, c.ID, p)
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

func (d Document) bump(ctx context.Context, tx *dao.Query, c *entity.DocumentLibrary) error {
	q := tx.DocumentLibrary
	r, e := q.WithContext(ctx).Where(q.ID.Eq(c.ID), q.Revision.Eq(c.Revision), q.Ready.Eq(1)).Updates(map[string]interface{}{"revision": c.Revision + 1, "updated_at": time.Now()})
	if e != nil {
		return e
	}
	if r.RowsAffected != 1 {
		return ErrConflict
	}
	return nil
}

// Save uses explicit create/update semantics: empty content is a valid document, never deletion.
func (d Document) Save(ctx context.Context, a Actor, ref, p, title string, content []byte, revision int32) (*entity.DocumentFile, error) {
	release := lock.Acquire("library:" + ref)
	defer release()
	if e := ValidatePath(p); e != nil {
		return nil, e
	}
	if len(content) > MaxContentSize || len(title) > 255 {
		return nil, ErrInvalid
	}
	c, e := d.Library(ctx, ref)
	if e != nil {
		return nil, e
	}
	if c.Ready != 1 {
		return nil, gorm.ErrRecordNotFound
	}
	f, e := d.file(ctx, c.ID, p)
	create := errors.Is(e, gorm.ErrRecordNotFound)
	if e != nil && !create {
		return nil, e
	}
	if !permission.CanChangeFile(a.Principal, *c, p) {
		return nil, permission.ErrDenied
	}
	if create {
		if revision != 0 {
			return nil, ErrConflict
		}
	} else if revision <= 0 || revision != f.Revision {
		return nil, ErrConflict
	}
	key, e := service.DocumentStorage.Put(ctx, content)
	if e != nil {
		return nil, e
	}
	if title == "" {
		title = path.Base(p)
	}
	if create {
		f = &entity.DocumentFile{LibraryID: c.ID, Path: p, Title: title, StorageKey: key, Size: int64(len(content)), Revision: 1, Position: 1000}
	}
	e = dao.Q.Transaction(func(tx *dao.Query) error {
		if e := d.bump(ctx, tx, c); e != nil {
			return e
		}
		if create {
			return tx.DocumentFile.WithContext(ctx).Create(f)
		}
		q := tx.DocumentFile
		r, err := q.WithContext(ctx).Where(q.ID.Eq(f.ID), q.Revision.Eq(revision)).Updates(map[string]interface{}{"storage_key": key, "size": len(content), "title": title, "revision": revision + 1, "updated_at": time.Now()})
		if err != nil {
			return err
		}
		if r.RowsAffected != 1 {
			return ErrConflict
		}
		return nil
	})
	if e != nil {
		return nil, e
	}
	if !create {
		f.Revision++
	}
	return f, nil
}

func (d Document) DeleteFile(ctx context.Context, a Actor, ref, p string, revision int32) error {
	release := lock.Acquire("library:" + ref)
	defer release()
	c, e := d.Library(ctx, ref)
	if e != nil {
		return e
	}
	if !permission.CanChangeFile(a.Principal, *c, p) {
		return permission.ErrDenied
	}
	f, e := d.file(ctx, c.ID, p)
	if e != nil {
		return e
	}
	if revision <= 0 || f.Revision != revision {
		return ErrConflict
	}
	return dao.Q.Transaction(func(tx *dao.Query) error {
		if e := d.bump(ctx, tx, c); e != nil {
			return e
		}
		q := tx.DocumentFile
		r, err := q.WithContext(ctx).Where(q.ID.Eq(f.ID), q.Revision.Eq(revision)).Delete()
		if err != nil {
			return err
		}
		if r.RowsAffected != 1 {
			return ErrConflict
		}
		return nil
	})
}

func (d Document) Rename(ctx context.Context, a Actor, ref, p, title string, revision int32) error {
	release := lock.Acquire("library:" + ref)
	defer release()
	// Rename only the display title; file paths remain stable.
	c, e := d.Library(ctx, ref)
	if e != nil {
		return e
	}
	if !permission.CanChangeFile(a.Principal, *c, p) {
		return permission.ErrDenied
	}
	if title == "" || len(title) > 255 || revision <= 0 {
		return ErrInvalid
	}
	return dao.Q.Transaction(func(tx *dao.Query) error {
		if e := d.bump(ctx, tx, c); e != nil {
			return e
		}
		q := tx.DocumentFile
		r, err := q.WithContext(ctx).Where(q.LibraryID.Eq(c.ID), q.Path.Eq(p), q.Revision.Eq(revision)).Updates(map[string]interface{}{"title": title, "revision": revision + 1, "updated_at": time.Now()})
		if err != nil {
			return err
		}
		if r.RowsAffected != 1 {
			return ErrConflict
		}
		return nil
	})
}

func (d Document) Order(ctx context.Context, a Actor, ref string, paths []string, revision int32) error {
	release := lock.Acquire("library:" + ref)
	defer release()
	c, e := d.Library(ctx, ref)
	if e != nil {
		return e
	}
	if revision != c.Revision || revision <= 0 {
		return ErrConflict
	}
	if !permission.CanManage(a.Principal, *c) {
		return permission.ErrDenied
	}
	if len(paths) > 10000 {
		return ErrInvalid
	}
	seen := map[string]bool{}
	for _, p := range paths {
		if seen[p] {
			return ErrInvalid
		}
		seen[p] = true
		if e = ValidatePath(p); e != nil {
			return e
		}
	}
	return dao.Q.Transaction(func(tx *dao.Query) error {
		if e := d.bump(ctx, tx, c); e != nil {
			return e
		}
		for i, p := range paths {
			q := tx.DocumentFile
			r, err := q.WithContext(ctx).Where(q.LibraryID.Eq(c.ID), q.Path.Eq(p)).Update(q.Position, int32(i))
			if err != nil {
				return err
			}
			if r.RowsAffected != 1 {
				return ErrConflict
			}
		}
		return nil
	})
}
