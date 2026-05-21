package controller

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/common/dao"
	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/controller"
)

type User struct {
	controller.Abstract
}

func (c User) CurUserInfo(ctx *gin.Context) {
	user := logic2.User{}.GetUser(ctx)

	username := ""
	userRole := ""
	userId := int32(0)
	if user != nil {
		username = user.Username
		userRole = user.Role
		userId = user.ID
	}

	c.JsonResponseWithoutError(ctx, map[string]interface{}{
		"user": map[string]interface{}{
			"username": username,
			"role":     userRole,
			"id":       userId,
		},
	})
}

func (c User) EditUser(ctx *gin.Context) {
	user := logic2.User{}.GetUser(ctx)
	if user == nil {
		c.JsonResponseWithServerError(ctx, errors.New("用户信息异常"))
		return
	}

	type ParamsValidate struct {
		Password string `json:"password"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	curUser, _ := logic2.User{}.GetById(int(user.ID))
	if params.Password != "" {
		curUser.Password = logic2.User{}.MakeUserPassword(params.Password)
	}

	err := dao.Q.RegistryUser.Save(curUser)
	if err != nil {
		c.JsonResponseWithServerError(ctx, errors.New("修改用户失败，err:"+err.Error()))
		return
	}

	c.JsonSuccessResponse(ctx)
}
