package logic

import (
	"archive/zip"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/w7panel/w7panel-zpk/modules/document/common/entity"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/permission"
	"gorm.io/gorm"
)

type Snapshot struct {
	Library entity.DocumentLibrary
	Files   []*entity.DocumentFile
}

// Immutable blob keys plus a library revision check form a stable content snapshot.
func (d *Document) snapshot(ctx context.Context, ref string) (Snapshot, error) {
	for i := 0; i < 3; i++ {
		c, e := d.Library(ctx, ref)
		if e != nil {
			return Snapshot{}, e
		}
		if c.Ready != 1 {
			return Snapshot{}, gorm.ErrRecordNotFound
		}
		q := d.Q.DocumentFile
		rows, e := q.WithContext(ctx).Where(q.LibraryID.Eq(c.ID)).Order(q.Position, q.ID).Find()
		if e != nil {
			return Snapshot{}, e
		}
		next, e := d.Library(ctx, ref)
		if e != nil {
			return Snapshot{}, e
		}
		if next.Revision == c.Revision {
			return Snapshot{*c, rows}, nil
		}
	}
	return Snapshot{}, ErrConflict
}

func (d *Document) selectFiles(ctx context.Context, a Actor, s Snapshot, scope map[string]bool, full bool) ([]*entity.DocumentFile, error) {
	paths := make([]string, 0, len(s.Files))
	for _, file := range s.Files {
		if scope == nil || scope[file.Path] {
			paths = append(paths, file.Path)
		}
	}
	capabilities, err := permission.FileActions(ctx, d.Permissions, a.Token, s.Library.Category, s.Library.Identifie, paths, []string{"read"})
	if err != nil {
		return nil, err
	}
	result := make([]*entity.DocumentFile, 0, len(paths))
	for _, file := range s.Files {
		if scope != nil && !scope[file.Path] {
			continue
		}
		if !capabilities[file.Path]["read"] {
			if full {
				return nil, permission.ErrDenied
			}
			continue
		}
		result = append(result, file)
	}
	if len(result) == 0 {
		return nil, permission.ErrDenied
	}
	return result, nil
}

// Export spools to disk before sending headers. It never accumulates all document bodies in memory.
func (d *Document) Export(ctx context.Context, a Actor, ref, format string, full bool, scope map[string]bool) (*os.File, error) {
	if format != "zip" && format != "markdown" {
		return nil, ErrInvalid
	}
	s, e := d.snapshot(ctx, ref)
	if e != nil {
		return nil, e
	}
	files, e := d.selectFiles(ctx, a, s, scope, full)
	if e != nil {
		return nil, e
	}
	var total int64
	for _, f := range files {
		total += f.Size
	}
	if total > 1<<30 {
		return nil, errors.New("导出内容超过 1 GiB，请分库下载")
	}
	output, e := os.CreateTemp("", "document-export-*")
	if e != nil {
		return nil, e
	}
	success := false
	defer func() {
		if !success {
			output.Close()
			os.Remove(output.Name())
		}
	}()
	if format == "zip" {
		z := zip.NewWriter(output)
		for _, f := range files {
			if e = ctx.Err(); e != nil {
				z.Close()
				return nil, e
			}
			entry, e := z.Create(f.Path)
			if e != nil {
				z.Close()
				return nil, e
			}
			if e = d.copyBlob(ctx, entry, f); e != nil {
				z.Close()
				return nil, e
			}
		}
		if e = z.Close(); e != nil {
			return nil, e
		}
	} else {
		if _, e = fmt.Fprintf(output, "# %s\n\n文档标识：%s\n\n修订号：%d\n", s.Library.Name, ref, s.Library.Revision); e != nil {
			return nil, e
		}
		for _, f := range files {
			if !strings.HasSuffix(strings.ToLower(f.Path), ".md") {
				continue
			}
			if _, e = fmt.Fprintf(output, "\n\n---\n\n## %s\n\n路径：%s\n\n", f.Title, f.Path); e != nil {
				return nil, e
			}
			if e = d.copyBlob(ctx, output, f); e != nil {
				return nil, e
			}
		}
	}
	// Revocation during a long export must prevent the generated artifact from being released.
	if _, e = d.selectFiles(ctx, a, Snapshot{Library: s.Library, Files: files}, nil, true); e != nil {
		return nil, e
	}
	if _, e = output.Seek(0, io.SeekStart); e != nil {
		return nil, e
	}
	success = true
	return output, nil
}

func (d *Document) copyBlob(ctx context.Context, w io.Writer, f *entity.DocumentFile) error {
	r, e := d.Store.Open(ctx, f.StorageKey)
	if e != nil {
		return e
	}
	defer r.Close()
	n, e := io.Copy(w, io.LimitReader(r, f.Size+1))
	if e != nil {
		return e
	}
	if n != f.Size {
		return errors.New("文档内容不完整")
	}
	return ctx.Err()
}

func tokenHash(t string) string { s := sha256.Sum256([]byte(t)); return hex.EncodeToString(s[:]) }

type LinkResult struct {
	ID        int32     `json:"id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (d *Document) CreateLink(ctx context.Context, a Actor, ref string, ttl int) (LinkResult, error) {
	if ttl < 1 || ttl > 720 {
		return LinkResult{}, ErrInvalid
	}
	s, e := d.snapshot(ctx, ref)
	if e != nil {
		return LinkResult{}, e
	}
	files, e := d.selectFiles(ctx, a, s, nil, false)
	if e != nil {
		return LinkResult{}, e
	}
	paths := make([]string, 0, len(files))
	for _, f := range files {
		paths = append(paths, f.Path)
	}
	b, e := json.Marshal(paths)
	if e != nil {
		return LinkResult{}, e
	}
	random := make([]byte, 32)
	if _, e = rand.Read(random); e != nil {
		return LinkResult{}, e
	}
	token := hex.EncodeToString(random)
	expiry := time.Now().Add(time.Duration(ttl) * time.Hour)
	link := entity.DocumentDownloadLink{LibraryID: s.Library.ID, UserID: a.Principal.ID, TokenHash: tokenHash(token), Paths: string(b), ExpiresAt: expiry}
	if e = d.Q.DocumentDownloadLink.WithContext(ctx).Create(&link); e != nil {
		return LinkResult{}, e
	}
	return LinkResult{link.ID, token, expiry}, nil
}

func (d *Document) Links(ctx context.Context, a Actor, ref string) ([]*entity.DocumentDownloadLink, error) {
	c, e := d.Library(ctx, ref)
	if e != nil {
		return nil, e
	}
	q := d.Q.DocumentDownloadLink.WithContext(ctx).Where(d.Q.DocumentDownloadLink.LibraryID.Eq(c.ID))
	if !a.Principal.Admin {
		q = q.Where(d.Q.DocumentDownloadLink.UserID.Eq(a.Principal.ID))
	}
	links, err := q.Order(d.Q.DocumentDownloadLink.ID.Desc()).Limit(100).Find()
	for i := range links {
		links[i].TokenHash = ""
		links[i].Paths = ""
	}
	return links, err
}

func (d *Document) RevokeLink(ctx context.Context, a Actor, id int32) error {
	q := d.Q.DocumentDownloadLink.WithContext(ctx).Where(d.Q.DocumentDownloadLink.ID.Eq(id))
	if !a.Principal.Admin {
		q = q.Where(d.Q.DocumentDownloadLink.UserID.Eq(a.Principal.ID))
	}
	r, err := q.Update(d.Q.DocumentDownloadLink.Revoked, 1)
	if err != nil {
		return err
	}
	if r.RowsAffected != 1 {
		return permission.ErrDenied
	}
	return nil
}

// Link downloads re-evaluate the issuer's current permissions using trusted service authentication.
func (d *Document) ExportLink(ctx context.Context, token, format string) (*os.File, error) {
	if len(token) != 64 {
		return nil, permission.ErrDenied
	}
	q := d.Q.DocumentDownloadLink
	l, e := q.WithContext(ctx).Where(q.TokenHash.Eq(tokenHash(token)), q.Revoked.Eq(0), q.ExpiresAt.Gt(time.Now())).First()
	if e != nil {
		return nil, permission.ErrDenied
	}
	c, e := d.Q.DocumentLibrary.WithContext(ctx).Where(d.Q.DocumentLibrary.ID.Eq(l.LibraryID)).First()
	if e != nil {
		return nil, permission.ErrDenied
	}
	var paths []string
	if e = json.Unmarshal([]byte(l.Paths), &paths); e != nil {
		return nil, e
	}
	scope := map[string]bool{}
	for _, p := range paths {
		scope[p] = true
	}
	delegate := *d
	delegate.Permissions = linkPermission{Service: d.Permissions, UserID: l.UserID}
	f, e := delegate.Export(ctx, Actor{Principal: permission.Principal{ID: l.UserID}}, c.Reference(), format, false, scope)
	if e != nil {
		return nil, e
	}
	// Check expiration/revocation again after the file is prepared.
	count, err := d.Q.DocumentDownloadLink.WithContext(ctx).Where(d.Q.DocumentDownloadLink.ID.Eq(l.ID), d.Q.DocumentDownloadLink.Revoked.Eq(0), d.Q.DocumentDownloadLink.ExpiresAt.Gt(time.Now())).Count()
	if err != nil || count != 1 {
		f.Close()
		os.Remove(f.Name())
		return nil, permission.ErrDenied
	}
	return f, nil
}

type linkPermission struct {
	permission.Service
	UserID int32
}

func (p linkPermission) Check(ctx context.Context, _ string, r []permission.Resource, a string) (permission.Decision, error) {
	return p.Service.CheckUser(ctx, p.UserID, r, a)
}

func (p linkPermission) Batch(ctx context.Context, _ string, requests []permission.Request) ([]permission.Decision, error) {
	return p.Service.BatchUser(ctx, p.UserID, requests)
}
