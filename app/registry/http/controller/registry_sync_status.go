package controller

import (
	"errors"
	"log/slog"

	"github.com/gin-gonic/gin"
	registrylogic "github.com/w7panel/w7panel-zpk/app/registry/logic"
	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/controller"
)

type RegistrySyncStatus struct {
	controller.Abstract
}

// List returns every active or failed synchronization for the selected repository.
func (c RegistrySyncStatus) List(ctx *gin.Context) {
	type ParamsValidate struct {
		Id int `json:"id" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	repository, _ := (registrylogic.Repository{}).GetByIdWithUserPermission(params.Id, commonlogic.User{}.GetUser(ctx))
	if repository == nil {
		c.JsonResponseWithServerError(ctx, errors.New("repository 不存在"))
		return
	}

	repositoryName := commonlogic.BuildRepositoryName(repository.Name, repository.Namespace)
	statuses, err := (registrylogic.RegistrySyncStatus{}).ListStatuses(repositoryName)
	if err != nil {
		slog.Warn("获取镜像同步状态失败", "err", err, "repositoryName", repositoryName)
		c.JsonResponseWithServerError(ctx, errors.New("获取镜像同步状态失败"))
		return
	}
	c.JsonResponseWithoutError(ctx, map[string]interface{}{"list": statuses})
}
