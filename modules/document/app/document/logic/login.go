package logic

import (
	"context"

	"github.com/w7panel/w7panel-zpk/modules/document/common/service/permission"
)

type Login struct{ Permissions permission.Service }

func (l Login) FromPanel(ctx context.Context, accessToken string) (string, error) {
	return l.Permissions.Login(ctx, accessToken)
}
