package logic

import (
	"context"

	"github.com/w7panel/w7panel-zpk/modules/document/common/service/permission"
)

func (d *Document) SetPermission(ctx context.Context, a Actor, ref, p string, userID int32, actions []string) error {
	if !a.Principal.Admin {
		return permission.ErrDenied
	}
	c, e := d.Library(ctx, ref)
	if e != nil {
		return e
	}
	resource := permission.LibraryResource(c.Reference())
	if p != "" {
		if _, e = d.file(ctx, c.ID, p); e != nil {
			return e
		}
		resource = permission.FileResource(ref, p)
	}
	if userID <= 0 {
		return ErrInvalid
	}
	return d.Permissions.Replace(ctx, a.Token, []permission.Grant{{UserID: userID, Resource: resource, Actions: actions}})
}
