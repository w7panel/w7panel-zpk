package controller

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/app/registry/logic"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/controller"
)

type UserManager struct {
	controller.Abstract
}

func (c UserManager) Create(ctx *gin.Context) {
	type ParamsValidate struct {
		UserName   string `json:"user_name" binding:"required"`
		Password   string `json:"password" binding:"required"`
		Desc       string `json:"desc"`
		ExpireDays int    `json:"expire_days"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	existsUser, _ := logic2.User{}.GetByUsername(params.UserName)
	if existsUser != nil {
		c.JsonResponseWithServerError(ctx, errors.New("用户已存在"))
		return
	}

	user := entity.RegistryUser{
		Username:   params.UserName,
		Desc:       params.Desc,
		Password:   logic2.User{}.MakeUserPassword(params.Password),
		ExpireDays: int32(params.ExpireDays),
	}
	err := dao.Q.RegistryUser.Create(&user)
	if err != nil {
		c.JsonResponseWithServerError(ctx, errors.New("创建用户失败，err:"+err.Error()))
		return
	}

	user.Password = ""
	c.JsonResponseWithoutError(ctx, map[string]interface{}{
		"user": user,
	})
}

func (c UserManager) Info(ctx *gin.Context) {
	type ParamsValidate struct {
		Id int `json:"id" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	curUser, _ := logic2.User{}.GetById(params.Id)
	if curUser == nil {
		c.JsonResponseWithServerError(ctx, errors.New("user 不存在"))
		return
	}

	curUser.Password = ""
	c.JsonResponseWithoutError(ctx, curUser)
}

func (c UserManager) List(ctx *gin.Context) {
	list, err := dao.Q.RegistryUser.
		Select(dao.Q.RegistryUser.ID, dao.Q.RegistryUser.Type, dao.Q.RegistryUser.Username, dao.Q.RegistryUser.Desc, dao.Q.RegistryUser.ExpireDays, dao.Q.RegistryUser.CreatedAt).
		Order(dao.Q.RegistryUser.CreatedAt.Desc()).
		Find()
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	c.JsonResponseWithoutError(ctx, map[string]interface{}{
		"list": list,
	})
}

func (c UserManager) Edit(ctx *gin.Context) {
	type ParamsValidate struct {
		Id         int    `json:"id" binding:"required"`
		UserName   string `json:"user_name" binding:"required"`
		Password   string `json:"password"`
		Desc       string `json:"desc"`
		ExpireDays int    `json:"expire_days"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	curUser, _ := logic2.User{}.GetById(params.Id)
	if curUser == nil {
		c.JsonResponseWithServerError(ctx, errors.New("用户不存在"))
		return
	}
	if curUser.Type == logic2.UserTypeSuperAdminForRegistry {
		c.JsonResponseWithServerError(ctx, errors.New("管理员不可修改"))
		return
	}

	if curUser.Username != params.UserName {
		existsUser, _ := logic2.User{}.GetByUsername(params.UserName)
		if existsUser != nil && existsUser.ID != curUser.ID {
			c.JsonResponseWithServerError(ctx, errors.New("用户已存在"))
			return
		}
	}

	curUser.Username = params.UserName
	curUser.Desc = params.Desc
	curUser.ExpireDays = int32(params.ExpireDays)
	if params.Password != "" {
		curUser.Password = logic2.User{}.MakeUserPassword(params.Password)
	}

	err := dao.Q.RegistryUser.Save(curUser)
	if err != nil {
		c.JsonResponseWithServerError(ctx, errors.New("修改用户失败，err:"+err.Error()))
		return
	}

	c.JsonResponseWithoutError(ctx, map[string]interface{}{
		"user": curUser,
	})
}

func (c UserManager) Delete(ctx *gin.Context) {
	type ParamsValidate struct {
		Id int `json:"id" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	curUser, _ := logic2.User{}.GetById(params.Id)
	if curUser == nil {
		c.JsonResponseWithServerError(ctx, errors.New("用户不存在"))
		return
	}
	if curUser.Type == logic2.UserTypeSuperAdminForRegistry {
		c.JsonResponseWithServerError(ctx, errors.New("管理员不可删除"))
		return
	}

	err := dao.Q.Transaction(func(tx *dao.Query) error {
		_, err := tx.RegistryUser.Delete(curUser)
		if err != nil {
			return err
		}
		return logic.Permission{}.DelUserPermission(tx, int(curUser.ID))
	})
	if err != nil {
		c.JsonResponseWithServerError(ctx, errors.New("删除用户失败, err:"+err.Error()))
		return
	}

	c.JsonSuccessResponse(ctx)
}
