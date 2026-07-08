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

	return 0
}
