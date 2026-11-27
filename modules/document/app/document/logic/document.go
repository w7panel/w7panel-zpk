package logic

import (
	"errors"
	"strings"
)

var ErrConflict = errors.New("标识已存在或操作冲突")
var ErrInvalid = errors.New("参数无效")

const MaxContentSize = 4 << 20

func SplitRef(ref string) (string, string) {
	namespace, identifie, _ := strings.Cut(ref, "/")
	return namespace, identifie
}
