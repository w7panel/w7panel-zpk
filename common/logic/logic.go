package logic

import (
	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/common/entity"
)

type Logic struct {
}

func (l Logic) GetUser(ctx *gin.Context) *entity.RegistryUser {
	val, exists := ctx.Get("user")
	if exists {
		return val.(*entity.RegistryUser)
	}

	user, _ := Session{}.ReadUserSession(ctx)
	if user != nil && user.UserID > 0 {
		_user, err := User{}.GetById(int(user.UserID))
		if err == nil {
			return _user
		}
	}

	return nil
}

func (l Logic) IsAdminUser(user *entity.RegistryUser) bool {
	return user.Role == UserRoleSuper || user.Role == UserRoleFounder
}

func (c Logic) GetConsoleUid(ctx *gin.Context) int32 {
	val, exists := ctx.Get("console_uid")
	if exists {
		return val.(int32)
	}

	user, _ := Session{}.ReadUserSession(ctx)
	if user != nil {
		return user.ConsoleUid
	}

	return 0
}
