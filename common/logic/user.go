package logic

import (
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
