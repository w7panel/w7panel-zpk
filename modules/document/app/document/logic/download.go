package logic

import (
	"archive/zip"
	"context"
	"database/sql"
	"errors"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/w7panel/w7panel-zpk/modules/document/common/dao"
	"github.com/w7panel/w7panel-zpk/modules/document/common/entity"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/permission"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/zpk"
)

type Download struct{}

type Snapshot struct {
	Library entity.DocumentLibrary
	Files   []*entity.DocumentFile
}

// Read metadata in one transaction; immutable blob keys keep the snapshot stable.
func (d Download) snapshot(ctx context.Context, ref string) (Snapshot, error) {
	category, identifie := SplitRef(ref)
	var snapshot Snapshot
	err := dao.Q.Transaction(func(tx *dao.Query) error {
		library := tx.DocumentLibrary
		c, e := library.WithContext(ctx).Where(library.Category.Eq(category), library.Identifie.Eq(identifie)).First()
		if e != nil {
			return e
		}
		q := tx.DocumentFile
		rows, e := q.WithContext(ctx).Where(q.LibraryID.Eq(c.ID)).Order(q.Position, q.ID).Find()
		if e != nil {
			return e
		}
		snapshot = Snapshot{Library: *c, Files: rows}
		return nil
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	return snapshot, err
}

// Export spools a ZIP snapshot to disk before sending headers.
func (d Download) Export(ctx context.Context, user zpk.User, ref string) (*os.File, error) {
	s, e := d.snapshot(ctx, ref)
	if e != nil {
		return nil, e
	}
	if err := permission.Require(service.PermissionSdk.CanRead(ctx, user, s.Library)); err != nil {
		return nil, err
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
	current, e := (Library{}).Get(ctx, ref)
	if e != nil {
		return nil, e
	}
	if err := permission.Require(service.PermissionSdk.CanRead(ctx, user, *current)); err != nil {
		return nil, err
	}
	if _, e = output.Seek(0, io.SeekStart); e != nil {
		return nil, e
	}
	success = true
	return output, nil
}

func (d Download) copyBlob(ctx context.Context, w io.Writer, f *entity.DocumentFile) error {
	r, e := service.DocumentStorage.Open(ctx, f.StorageKey)
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

func (Download) URL() (string, error) {
	domain, err := url.Parse(strings.TrimSpace(facade.GetConfig().GetString("document.domain_url")))
	if err != nil || domain.Host == "" || (domain.Scheme != "http" && domain.Scheme != "https") || domain.User != nil || (domain.Path != "" && domain.Path != "/") || domain.RawQuery != "" || domain.Fragment != "" {
		return "", errors.New("文档 DOMAIN_URL 未配置为有效的 HTTP(S) 域名地址")
	}
	domain.Path = "/doc/download"
	domain.RawPath = ""
	return domain.String(), nil
}
