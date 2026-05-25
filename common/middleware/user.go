package middleware

import (
	"sync"

	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	"github.com/w7panel/w7panel-zpk/common/function"
	"github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/w7panel/w7panel-zpk/common/types/registry"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
)

const superRole = "super"
const founderRole = "founder"

var userCreationLocks sync.Map

func lockUserCreation(username string) func() {
	lock, _ := userCreationLocks.LoadOrStore(username, &sync.Mutex{})
	mu := lock.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

func getOrCreateUser(userId string, username string, userRole string, autoCreate bool) (*entity.RegistryUser, error) {
	unlock := lockUserCreation(username)
	defer unlock()

	var user *entity.RegistryUser
	w7panelUser, _ := dao.Q.W7panelUser.Where(dao.Q.W7panelUser.W7panelUsername.Eq(username)).First()
	if w7panelUser != nil {
		user, _ = dao.Q.RegistryUser.Where(dao.Q.RegistryUser.ID.Eq(w7panelUser.UserUID)).First()
	}
	if user == nil && autoCreate {
		err := dao.Q.Transaction(func(tx *dao.Query) error {
			user = &entity.RegistryUser{
				Username: "w7" + username,
				Password: logic.User{}.MakeUserPassword(function.GetRandomString(16)),
				Desc:     "w7panel 用户",
				Type:     logic.UserTypeOrdinaryForRegistry,
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

		facade.GetEvent().Publish(registry.AddUserPermissionEvent, registry.AddUserPermissionPayload{
			UserID:        user.ID,
			ResourceType:  "namespace",
			ResourceValue: facade.GetConfig().GetString("setting.depot.oci_namespace"),
			Actions:       []string{"push", "pull"},
		})
	}

	return user, nil
}
