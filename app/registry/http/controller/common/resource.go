package common

import (
	"errors"
	"github.com/gin-gonic/gin"
	registrycommon "github.com/w7panel/w7panel-zpk/app/registry/logic/common"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/controller"
	"gorm.io/gorm"
	"slices"
	"strings"
)

type Resource struct{ controller.Abstract }
type ResourceRespInfo struct {
	entity.RegistryRepository
	CanEdit bool `json:"can_edit"`
}

func (c Resource) List(ctx *gin.Context) {
	type ParamsValidate struct {
		ResourceType string `json:"resource_type" binding:"required,max=64,excludesall= \t\x00\r\n"`
		Namespace    string `json:"namespace"`
		SubNamespace string `json:"sub_namespace"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	user := logic2.User{}.GetUser(ctx)
	query := dao.Q.RegistryRepository.Where(dao.Q.RegistryRepository.ResourceType.Eq(params.ResourceType)).Order(dao.Q.RegistryRepository.CreatedAt.Desc())
	if params.Namespace != "" {
		if err := CheckNamespacePermission(ctx, params.Namespace, false); err != nil {
			c.JsonResponseWithServerError(ctx, err)
			return
		}
		query = query.Where(dao.Q.RegistryRepository.Namespace.Eq(params.Namespace))
	} else if user != nil && !(logic2.User{}.IsAdminUser(user)) {
		namespacePermissions, err := dao.Q.RegistryUserPermission.
			Where(dao.Q.RegistryUserPermission.UserID.Eq(user.ID)).
			Where(dao.Q.RegistryUserPermission.ResourceType.Eq(string(registrycommon.PermissionResourceTypeNamespace))).
			Find()
		if err != nil {
			c.JsonResponseWithServerError(ctx, err)
			return
		}
		namespace := make([]string, len(namespacePermissions))
		for i, item := range namespacePermissions {
			namespace[i] = item.ResourceValue
		}

		query = query.Where(dao.Q.RegistryRepository.Namespace.In(namespace...))
	}
	if params.SubNamespace != "" {
		query = query.Where(dao.Q.RegistryRepository.Name.Like(params.SubNamespace + "/%"))
	}

	list, err := query.Find()
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	ignoredNamespaces := strings.Split(facade.GetConfig().GetString("setting.registry.list_ignore_namespaces"), ",")
	retLst := make([]ResourceRespInfo, 0, len(list))
	for _, item := range list {
		if item.Namespace != "" && slices.Contains(ignoredNamespaces, item.Namespace) {
			continue
		}
		item.Registry = facade.GetConfig().GetString("registry_cli.default.external_domain")
		respInfo := ResourceRespInfo{
			RegistryRepository: *item,
			CanEdit:            registrycommon.ResourcePermission{}.IsCanOperate(ctx, item.UserID),
		}

		retLst = append(retLst, respInfo)
	}

	c.JsonResponseWithoutError(ctx, map[string]interface{}{"list": retLst})
}

func (c Resource) Create(ctx *gin.Context) {
	type ParamsValidate struct {
		ResourceType string `json:"resource_type" binding:"required,max=64"`
		Name         string `json:"name" binding:"required,max=512"`
		Namespace    string `json:"namespace" binding:"required"`
		VisibleType  uint8  `json:"visible_type" binding:"required,oneof=1 2 3 4"`
		Desc         string `json:"desc"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	if err := CheckNamespacePermission(ctx, params.Namespace, true); err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	existsRepository, lookupErr := (registrycommon.Resource{}).Get(ctx.Request.Context(), registrycommon.ResourcePermissionResource{Type: params.ResourceType, Value: logic2.BuildRepositoryName(params.Name, params.Namespace)})
	if lookupErr != nil && !errors.Is(lookupErr, gorm.ErrRecordNotFound) {
		c.JsonResponseWithServerError(ctx, lookupErr)
		return
	}
	if lookupErr == nil && existsRepository != nil {
		c.JsonResponseWithServerError(ctx, errors.New("repository 已存在"))
		return
	}

	err := (registrycommon.Resource{}).Save(ctx.Request.Context(), &entity.RegistryRepository{
		ResourceType: params.ResourceType,
		Name:         params.Name,
		Namespace:    params.Namespace,
		VisibleType:  int32(params.VisibleType),
		Desc:         params.Desc,
		UserID:       logic2.User{}.GetUser(ctx).ID,
	}, "")
	if err != nil {
		c.JsonResponseWithServerError(ctx, errors.New("repository 创建失败, err: "+err.Error()))
		return
	}

	c.JsonSuccessResponse(ctx)
}

func (c Resource) Info(ctx *gin.Context) {
	type ParamsValidate struct {
		Id           int    `json:"id" binding:"required"`
		ResourceType string `json:"resource_type" binding:"required,max=64"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	user := logic2.User{}.GetUser(ctx)
	curRepository, _ := (registrycommon.Resource{}).GetByID(ctx.Request.Context(), int32(params.Id), params.ResourceType, user, false)
	if curRepository == nil {
		c.JsonResponseWithServerError(ctx, errors.New("repository 不存在"))
		return
	}
	curRepository.Registry = facade.GetConfig().GetString("registry_cli.default.external_domain")
	respInfo := ResourceRespInfo{
		RegistryRepository: *curRepository,
		CanEdit:            registrycommon.ResourcePermission{}.IsCanOperate(ctx, curRepository.UserID),
	}

	c.JsonResponseWithoutError(ctx, respInfo)
}
