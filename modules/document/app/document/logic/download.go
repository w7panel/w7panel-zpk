package logic

import (
	"archive/zip"
	"context"
	"errors"
	"io"
	"os"

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

// Export spools a ZIP snapshot to disk before sending headers.
func (d *Document) Export(ctx context.Context, a Actor, ref string) (*os.File, error) {
	s, e := d.snapshot(ctx, ref)
	if e != nil {
		return nil, e
	}
	if !permission.CanRead(a.Principal, s.Library) {
		return nil, permission.ErrDenied
	}
	files := s.Files
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

	// Recheck current visibility and deletion after preparing the snapshot.
	current, e := d.Library(ctx, ref)
	if e != nil {
		return nil, e
	}
	if !permission.CanRead(a.Principal, *current) {
		return nil, permission.ErrDenied
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
