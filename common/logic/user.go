package logic

import (
	"errors"
	"sync"

	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	"github.com/w7panel/w7panel-zpk/common/function"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"gorm.io/gorm"
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

var panelUserCreationLocks sync.Map

func lockPanelUserCreation(username string) func() {
	lock, _ := panelUserCreationLocks.LoadOrStore(username, &sync.Mutex{})
	mu := lock.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
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

func (l User) GetOrCreatePanelUser(userId string, username string, userRole string) (*entity.RegistryUser, error) {
	if userId == "" || username == "" || userRole == "" {
		return nil, errors.New("面板用户信息不完整")
	}
	unlock := lockPanelUserCreation(userId)
	defer unlock()

	var user *entity.RegistryUser
	w7panelUser, err := dao.Q.W7panelUser.Where(dao.Q.W7panelUser.W7panelUID.Eq(userId)).First()
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if w7panelUser == nil {
		// Display names can be reused; only the verified panel subject identifies an account.
		existing, err := dao.Q.W7panelUser.Where(dao.Q.W7panelUser.W7panelUsername.Eq(username)).First()
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		if existing != nil {
			w7panelUser = existing
		}
	}
	if w7panelUser != nil {
		user, err = dao.Q.RegistryUser.Where(dao.Q.RegistryUser.ID.Eq(w7panelUser.UserUID)).First()
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	if user == nil {
		err := dao.Q.Transaction(func(tx *dao.Query) error {
			user = &entity.RegistryUser{
				Username: "w7" + username,
				Password: l.MakeUserPassword(function.GetRandomString(16)),
				Desc:     "w7panel 用户",
				Type:     UserTypeOrdinaryForRegistry,
				Role:     userRole,
			}
			err := tx.RegistryUser.Create(user)
			if err != nil {
				return err
			}

			if w7panelUser != nil {
				_, err = tx.W7panelUser.Where(tx.W7panelUser.ID.Eq(w7panelUser.ID)).Update(tx.W7panelUser.UserUID, user.ID)
			} else {
				err = tx.W7panelUser.Create(&entity.W7panelUser{
					W7panelUID:      userId,
					W7panelUsername: username,
					UserUID:         user.ID,
				})
			}

			return err
		})
		if err != nil {
			return nil, err
		}

	} else {
		_, err := dao.Q.RegistryUser.Where(dao.RegistryUser.ID.Eq(user.ID)).Updates(entity.RegistryUser{
			Role: userRole,
		})
		if err != nil {
			return nil, err
		}
		user.Role = userRole
	}

	return user, nil
}
