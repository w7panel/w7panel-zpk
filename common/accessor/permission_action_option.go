package accessor

import (
	"database/sql/driver"
	"fmt"
	"strings"
)

type PermissionActionOption []string

func (c PermissionActionOption) Value() (driver.Value, error) {
	if len(c) == 0 {
		return "", nil
	}
	return strings.Join(c, ","), nil
}

func (c *PermissionActionOption) Scan(value interface{}) error {
	if value == nil {
		*c = PermissionActionOption{}
		return nil
	}

	var str string
	switch v := value.(type) {
	case string:
		str = v
	case []byte:
		str = string(v)
	default:
		return fmt.Errorf("不支持的类型: %T", value)
	}

	if str == "" {
		*c = PermissionActionOption{}
		return nil
	}

	*c = strings.Split(str, ",")
	return nil
}
