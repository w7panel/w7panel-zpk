package permission

import (
	"errors"

	"github.com/w7panel/w7panel-zpk/modules/document/common/entity"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/zpk"
)

var ErrDenied = errors.New("无操作权限")

// CanManage limits mutations to the owner and founder.
func CanManage(principal zpk.User, library entity.DocumentLibrary) bool {
	return principal.ID > 0 && (principal.Admin || principal.ID == library.UserID)
}

func CanRead(principal zpk.User, library entity.DocumentLibrary) bool {
	return library.Ready == 1 && (library.Public == 1 || CanManage(principal, library))
}

// Default homepage protection is a document invariant, including for founders.
func CanChangeFile(principal zpk.User, library entity.DocumentLibrary, filename string) bool {
	return CanManage(principal, library) && !(library.Protected == 1 && filename == "index.md")
}
