package middleware

import (
	"github.com/w7panel/w7panel-zpk/common/entity"
	"github.com/w7panel/w7panel-zpk/common/logic"
)

const superRole = "super"
const founderRole = "founder"

func getOrCreateUser(userId string, username string, userRole string, autoCreate bool) (*entity.RegistryUser, error) {
	return logic.User{}.GetOrCreatePanelUser(userId, username, userRole, autoCreate)
}
