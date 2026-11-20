package types

import "github.com/w7panel/w7panel-zpk/modules/document/common/service/permission"

type Actor struct {
	Principal permission.Principal
	Token     string
}
