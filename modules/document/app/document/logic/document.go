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
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/lock"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/permission"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/storage"
	"github.com/w7panel/w7panel-zpk/modules/document/common/types"
	"gorm.io/gorm"
)

var ErrConflict = errors.New("内容已更新或标识已存在，请刷新后重试")
var ErrInvalid = errors.New("参数无效")
var identifiePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,95}$`)

const MaxContentSize = 4 << 20

type Actor = types.Actor
type Document struct {
	Q           *dao.Query
	Permissions permission.Service
	Store       storage.Store
}

func New(query *dao.Query, permissions permission.Service, store storage.Store) *Document {
	return &Document{Q: query, Permissions: permissions, Store: store}
}

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

func (d *Document) Library(ctx context.Context, ref string) (*entity.DocumentLibrary, error) {
	category, id, e := ValidateRef(ref)
	if e != nil {
		return nil, e
	}
	q := d.Q.DocumentLibrary
	return q.WithContext(ctx).Where(q.Category.Eq(category), q.Identifie.Eq(id)).First()
}

func (d *Document) category(ctx context.Context, id string) (*entity.DocumentCategory, error) {
	q := d.Q.DocumentCategory
	return q.WithContext(ctx).Where(q.Identifie.Eq(id)).First()
}

type Page struct {
	List  []*entity.DocumentLibrary `json:"list"`
	Total int64                     `json:"total"`
}

func (d *Document) List(ctx context.Context, a Actor, category string, page, limit int) (Page, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	q := d.Q.DocumentLibrary.WithContext(ctx).Where(d.Q.DocumentLibrary.Ready.Eq(1))
	if !a.Principal.Admin {
		q = q.Where(d.Q.DocumentLibrary.UserID.Eq(a.Principal.ID))
	}
	if category != "" {
		q = q.Where(d.Q.DocumentLibrary.Category.Eq(category))
	}
	rows, total, e := q.Order(d.Q.DocumentLibrary.ID.Desc()).FindByPage((page-1)*limit, limit)
	return Page{rows, total}, e
}

// Create can initialize a homepage with a read-only file permission policy.
func (d *Document) Create(ctx context.Context, a Actor, ref, name, description, homepage string, initialize bool) (*entity.DocumentLibrary, error) {
	release := lock.Acquire("library:" + ref)
	defer release()
	category, id, e := ValidateRef(ref)
	if e != nil || strings.TrimSpace(name) == "" || len(name) > 255 || len(description) > 4096 || len(homepage) > MaxContentSize {
		return nil, ErrInvalid
	}
	if a.Principal.ID <= 0 {
		return nil, permission.ErrDenied
	}
	if _, e = d.category(ctx, category); e != nil {
		return nil, e
	}
	v, e := d.Permissions.Check(ctx, a.Token, []permission.Resource{permission.CategoryResource(category)}, "create")
	if e != nil {
		return nil, e
	}
	if !v.Allowed {
		return nil, permission.ErrDenied
	}
	c, e := d.Library(ctx, ref)
	if e == nil {
		if c.UserID != a.Principal.ID || c.Ready < 0 || (!initialize && c.Ready == 1) {
			return nil, ErrConflict
		}
		if c.Ready == 1 {
			return c, nil
		}
	} else if !errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, e
	} else {
		c = &entity.DocumentLibrary{Category: category, Identifie: id, Name: name, Description: description, UserID: a.Principal.ID, Revision: 1}
		if e = d.Q.DocumentLibrary.WithContext(ctx).Create(c); e != nil {
			return nil, ErrConflict
		}
	}
	actions := []string{"read", "write", "create", "delete"}
	if initialize {
		actions = []string{"read", "write", "create"}
	}
	grants := []permission.Grant{{UserID: a.Principal.ID, Resource: permission.LibraryResource(ref), Actions: actions}}
	if initialize {
		grants = append(grants, permission.Grant{UserID: a.Principal.ID, Resource: permission.FileResource(ref, "index.md"), Actions: []string{"read"}})
	}
	if e = d.Permissions.Initialize(ctx, a.Token, grants); e != nil {
		return nil, e
	}
	if initialize {
		key, e := d.Store.Put(ctx, []byte(homepage))
		if e != nil {
			return nil, e
		}
		f := &entity.DocumentFile{LibraryID: c.ID, Path: "index.md", Title: "首页", StorageKey: key, Size: int64(len(homepage)), Revision: 1}
		count, err := d.Q.DocumentFile.WithContext(ctx).Where(d.Q.DocumentFile.LibraryID.Eq(c.ID), d.Q.DocumentFile.Path.Eq("index.md")).Count()
		if err != nil {
			return nil, err
		}
		if count == 0 {
			if e = d.Q.DocumentFile.WithContext(ctx).Create(f); e != nil {
				return nil, e
			}
		}
	}
	if _, e = d.Q.DocumentLibrary.WithContext(ctx).Where(d.Q.DocumentLibrary.ID.Eq(c.ID)).Update(d.Q.DocumentLibrary.Ready, 1); e != nil {
		return nil, e
	}
	c.Ready = 1
	return c, nil
}

func (d *Document) EditLibrary(ctx context.Context, a Actor, ref, name, description string, rev int32) error {
	release := lock.Acquire("library:" + ref)
	defer release()
	c, e := d.Library(ctx, ref)
	if e != nil {
		return e
	}
	if e = permission.Require(ctx, d.Permissions, a.Token, permission.Resources(c.Category, c.Identifie, ""), "write"); e != nil {
		return e
	}
	if name == "" || len(name) > 255 || len(description) > 4096 || rev <= 0 {
		return ErrInvalid
	}
	q := d.Q.DocumentLibrary
	r, e := q.WithContext(ctx).Where(q.ID.Eq(c.ID), q.Revision.Eq(rev), q.Ready.Eq(1)).Updates(map[string]interface{}{"name": name, "description": description, "revision": rev + 1, "updated_at": time.Now()})
	if e != nil {
		return e
	}
	if r.RowsAffected != 1 {
		return ErrConflict
	}
	return nil
}

func (d *Document) DeleteLibrary(ctx context.Context, a Actor, ref string) error {
	release := lock.Acquire("library:" + ref)
	defer release()
	c, e := d.Library(ctx, ref)
	if e != nil {
		return e
	}
	if e = permission.Require(ctx, d.Permissions, a.Token, permission.Resources(c.Category, c.Identifie, ""), "delete"); e != nil {
		return e
	}
	files, e := d.Q.DocumentFile.WithContext(ctx).Where(d.Q.DocumentFile.LibraryID.Eq(c.ID)).Find()
	if e != nil {
		return e
	}
	for _, f := range files {
		if e = permission.Require(ctx, d.Permissions, a.Token, permission.Resources(c.Category, c.Identifie, f.Path), "delete"); e != nil {
			return e
		}
	}
	// Keep an explicit deny so recreation cannot inherit stale file-level grants.
	grants := []permission.Grant{{UserID: c.UserID, Resource: permission.LibraryResource(ref), Actions: []string{}}}
	for _, f := range files {
		grants = append(grants, permission.Grant{UserID: c.UserID, Resource: permission.FileResource(ref, f.Path), Actions: []string{}})
	}
	// Resources use immutable paths; recreation is prevented by retaining the library tombstone.
	if e = d.Permissions.Replace(ctx, a.Token, grants); e != nil {
		return e
	}
	return d.Q.Transaction(func(tx *dao.Query) error {
		q := tx.DocumentLibrary
		r, e := q.WithContext(ctx).Where(q.ID.Eq(c.ID), q.Revision.Eq(c.Revision), q.Ready.Eq(1)).Updates(map[string]interface{}{"ready": -1, "revision": c.Revision + 1})
		if e != nil {
			return e
		}
		if r.RowsAffected != 1 {
			return ErrConflict
		}
		_, e = tx.DocumentDownloadLink.WithContext(ctx).Where(tx.DocumentDownloadLink.LibraryID.Eq(c.ID)).Delete()
		return e
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

func (d *Document) Files(ctx context.Context, a Actor, ref string) (Tree, error) {
	c, e := d.Library(ctx, ref)
	if e != nil {
		return Tree{}, e
	}
	if c.Ready != 1 {
		return Tree{}, gorm.ErrRecordNotFound
	}
	files, e := d.Q.DocumentFile.WithContext(ctx).Where(d.Q.DocumentFile.LibraryID.Eq(c.ID)).Order(d.Q.DocumentFile.Position, d.Q.DocumentFile.ID).Find()
	if e != nil {
		return Tree{}, e
	}
	out := Tree{Files: []FileInfo{}, Revision: c.Revision, Name: c.Name}
	if out.CanCreate, e = permission.Allowed(ctx, d.Permissions, a.Token, permission.Resources(c.Category, c.Identifie, ""), "create"); e != nil {
		return out, e
	}
	canRead, e := permission.Allowed(ctx, d.Permissions, a.Token, permission.Resources(c.Category, c.Identifie, ""), "read")
	if e != nil {
		return out, e
	}
	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, file.Path)
	}
	capabilities, err := permission.FileActions(ctx, d.Permissions, a.Token, c.Category, c.Identifie, paths, []string{"read", "write", "delete"})
	if err != nil {
		return out, err
	}
	for _, file := range files {
		rights := capabilities[file.Path]
		if !rights["read"] {
			continue
		}
		out.Files = append(out.Files, FileInfo{file.Path, file.Title, file.Size, file.Position, file.Revision, rights["write"], rights["delete"]})
	}
	if !canRead && len(out.Files) == 0 {
		return out, permission.ErrDenied
	}
	return out, nil
}

func (d *Document) file(ctx context.Context, c int32, p string) (*entity.DocumentFile, error) {
	if e := ValidatePath(p); e != nil {
		return nil, e
	}
	q := d.Q.DocumentFile
	return q.WithContext(ctx).Where(q.LibraryID.Eq(c), q.Path.Eq(p)).First()
}

func (d *Document) Read(ctx context.Context, a Actor, ref, p string) (*entity.DocumentFile, []byte, error) {
	c, e := d.Library(ctx, ref)
	if e != nil {
		return nil, nil, e
	}
	if c.Ready != 1 {
		return nil, nil, gorm.ErrRecordNotFound
	}
	if e = permission.Require(ctx, d.Permissions, a.Token, permission.Resources(c.Category, c.Identifie, p), "read"); e != nil {
		return nil, nil, e
	}
	f, e := d.file(ctx, c.ID, p)
	if e != nil {
		return nil, nil, e
	}
	r, e := d.Store.Open(ctx, f.StorageKey)
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

func (d *Document) bump(ctx context.Context, tx *dao.Query, c *entity.DocumentLibrary) error {
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
func (d *Document) Save(ctx context.Context, a Actor, ref, p, title string, content []byte, revision int32) (*entity.DocumentFile, error) {
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
	if create {
		if revision != 0 {
			return nil, ErrConflict
		}
		if e = permission.Require(ctx, d.Permissions, a.Token, permission.Resources(c.Category, c.Identifie, ""), "create"); e != nil {
			return nil, e
		}
		v, e := d.Permissions.Check(ctx, a.Token, []permission.Resource{permission.FileResource(ref, p)}, "write")
		if e != nil {
			return nil, e
		}
		if v.Explicit && !v.Allowed {
			return nil, permission.ErrDenied
		}
	} else {
		if revision <= 0 || revision != f.Revision {
			return nil, ErrConflict
		}
		if e = permission.Require(ctx, d.Permissions, a.Token, permission.Resources(c.Category, c.Identifie, p), "write"); e != nil {
			return nil, e
		}
	}
	if create {
		grants := []permission.Grant{{UserID: c.UserID, Resource: permission.FileResource(ref, p), Actions: []string{"read", "write", "delete"}}}
		if a.Principal.ID != c.UserID && !a.Principal.Admin {
			grants = append(grants, permission.Grant{UserID: a.Principal.ID, Resource: permission.FileResource(ref, p), Actions: []string{"read", "write", "delete"}})
		}
		if err := d.Permissions.Initialize(ctx, a.Token, grants); err != nil {
			return nil, err
		}
	}
	key, e := d.Store.Put(ctx, content)
	if e != nil {
		return nil, e
	}
	if title == "" {
		title = path.Base(p)
	}
	if create {
		f = &entity.DocumentFile{LibraryID: c.ID, Path: p, Title: title, StorageKey: key, Size: int64(len(content)), Revision: 1, Position: 1000}
	}
	e = d.Q.Transaction(func(tx *dao.Query) error {
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

func (d *Document) DeleteFile(ctx context.Context, a Actor, ref, p string, revision int32) error {
	release := lock.Acquire("library:" + ref)
	defer release()
	c, e := d.Library(ctx, ref)
	if e != nil {
		return e
	}
	if e = permission.Require(ctx, d.Permissions, a.Token, permission.Resources(c.Category, c.Identifie, p), "delete"); e != nil {
		return e
	}
	f, e := d.file(ctx, c.ID, p)
	if e != nil {
		return e
	}
	if revision <= 0 || f.Revision != revision {
		return ErrConflict
	}
	return d.Q.Transaction(func(tx *dao.Query) error {
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

func (d *Document) Rename(ctx context.Context, a Actor, ref, p, title string, revision int32) error {
	release := lock.Acquire("library:" + ref)
	defer release()
	// Rename the display title rather than moving the permission-bearing path.
	c, e := d.Library(ctx, ref)
	if e != nil {
		return e
	}
	if e = permission.Require(ctx, d.Permissions, a.Token, permission.Resources(c.Category, c.Identifie, p), "write"); e != nil {
		return e
	}
	if title == "" || len(title) > 255 || revision <= 0 {
		return ErrInvalid
	}
	return d.Q.Transaction(func(tx *dao.Query) error {
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

func (d *Document) Order(ctx context.Context, a Actor, ref string, paths []string, revision int32) error {
	release := lock.Acquire("library:" + ref)
	defer release()
	c, e := d.Library(ctx, ref)
	if e != nil {
		return e
	}
	if revision != c.Revision || revision <= 0 {
		return ErrConflict
	}
	if e = permission.Require(ctx, d.Permissions, a.Token, permission.Resources(c.Category, c.Identifie, ""), "write"); e != nil {
		return e
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
		if e = permission.Require(ctx, d.Permissions, a.Token, permission.Resources(c.Category, c.Identifie, p), "read"); e != nil {
			return e
		}
	}
	return d.Q.Transaction(func(tx *dao.Query) error {
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
