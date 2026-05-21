package logic

import (
	"fmt"
	"time"

	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	"github.com/w7panel/w7panel-zpk/common/function"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
)

var UserRoleSuper = "super"
var UserRoleFounder = "founder"
var UserRoleTech = "tech"
var UserRoleUser = "user"

var UserTypeOrdinaryForRegistry = int32(0)
var UserTypeSuperAdminForRegistry = int32(1)

type User struct {
	Logic
}

func (l User) MakeUserPassword(password string) string {
	salt := facade.GetConfig().GetString("setting.registry.user_pwd_secret")
	if salt == "" {
		salt = password[0:min(len(password), 6)]
	}
	return function.GetMd5(password + salt)
}

func (l User) GetById(id int) (*entity.RegistryUser, error) {
	return dao.Q.RegistryUser.Where(dao.RegistryUser.ID.Eq(int32(id))).First()
}

func (l User) GetByUsername(username string) (*entity.RegistryUser, error) {
	return dao.Q.RegistryUser.Where(dao.RegistryUser.Username.Eq(username)).First()
}

func (l User) GetUserAvatar(uid int) string {
	p := [3]int{0, 0, 0} // 使用固定长度的数组

	// 从后向前遍历数组
	for i := len(p) - 1; i >= 0; i-- {
		if i > 0 {
			p[i] = uid % 1000
			uid = uid / 1000 // Go 中的整数除法等同于 PHP 的 floor($uid / 1000)
		} else {
			p[i] = uid
		}
	}

	// 拼接字符串，使用 fmt.Sprintf 替代 join
	// 注意：Go 的 time.Now().Unix() 返回的是 int64
	return fmt.Sprintf("https://avatar.w7.cc/images/avatar/%d/%d/%d.jpg?v=%d&imageView2/5/w/100/h/100/format/webp", p[0], p[1], p[2], time.Now().Unix())
}
