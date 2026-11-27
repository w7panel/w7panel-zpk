package common

import (
	"errors"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"

	registrycommon "github.com/w7panel/w7panel-zpk/app/registry/logic/common"
	"github.com/w7panel/w7panel-zpk/common/accessor"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/controller"
)

type NamespaceRespInfo struct {
	entity.RegistryNamespace
	CanEdit bool `json:"can_edit"`
}

type Namespace struct {
	controller.Abstract
}

func (c Namespace) Create(ctx *gin.Context) {
	type ParamsValidate struct {
		Name        string `json:"name" binding:"required"`
		VisibleType uint8  `json:"visible_type" binding:"required,oneof=1 2 4"`
		Visible     *bool  `json:"visible"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	existsNamespace, _ := registrycommon.Namespace{}.GetByName(params.Name)
	if existsNamespace != nil {
		c.JsonResponseWithServerError(ctx, errors.New("namespace 已存在"))
		return
	}

	user := logic2.User{}.GetUser(ctx)
	err := dao.Q.Transaction(func(tx *dao.Query) error {
		namespace := &entity.RegistryNamespace{
			Name:        params.Name,
			VisibleType: int32(params.VisibleType),
			UserID:      logic2.User{}.GetUser(ctx).ID,
		}
		err := tx.RegistryNamespace.Create(namespace)
		if err != nil {
			return err
		}
		if params.Visible != nil && !*params.Visible {
			if _, err := tx.RegistryNamespace.Where(tx.RegistryNamespace.ID.Eq(namespace.ID)).Update(tx.RegistryNamespace.Visible, false); err != nil {
				return err
			}
		}

		if user != nil && !(logic2.User{}.IsAdminUser(user)) {
			err = tx.RegistryUserPermission.Create(&entity.RegistryUserPermission{
				UserID:        user.ID,
				ResourceType:  string(registrycommon.PermissionResourceTypeNamespace),
				ResourceValue: params.Name,
				Action:        &accessor.PermissionActionOption{string(registrycommon.PermissionActionTypePush), string(registrycommon.PermissionActionTypePull)},
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		c.JsonResponseWithServerError(ctx, errors.New("namespace 创建失败, err: "+err.Error()))
		return
	}

	c.JsonSuccessResponse(ctx)
}

func (c Namespace) Info(ctx *gin.Context) {
	type ParamsValidate struct {
		Id int `json:"id" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	user := logic2.User{}.GetUser(ctx)
	curNamespace, _ := registrycommon.Namespace{}.GetByIdWithUserPermission(params.Id, user)
	if curNamespace == nil {
		c.JsonResponseWithServerError(ctx, errors.New("namespace 不存在"))
		return
	}
	c.JsonResponseWithoutError(ctx, NamespaceRespInfo{
		RegistryNamespace: *curNamespace,
		CanEdit:           registrycommon.ResourcePermission{}.IsCanOperate(ctx, curNamespace.UserID),
	})
}

func (c Namespace) List(ctx *gin.Context) {
	query := dao.Q.RegistryNamespace.Order(dao.Q.RegistryNamespace.CreatedAt.Desc())
	query = query.Where(dao.Q.RegistryNamespace.Visible.Is(true))
	user := logic2.User{}.GetUser(ctx)
	if user != nil && !(logic2.User{}.IsAdminUser(user)) {
		namespacePermissions, _ := dao.Q.RegistryUserPermission.
			Where(dao.Q.RegistryUserPermission.UserID.Eq(user.ID)).
			Where(dao.Q.RegistryUserPermission.ResourceType.Eq(string(registrycommon.PermissionResourceTypeNamespace))).
			Find()
		namespace := make([]string, len(namespacePermissions))
		for i, item := range namespacePermissions {
			namespace[i] = item.ResourceValue
		}

		query = query.Where(dao.Q.RegistryNamespace.Name.In(namespace...))
	}
	list, err := query.Find()
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	isAdminUser := user != nil && logic2.User{}.IsAdminUser(user)
	ignoreNamespaceConfig := facade.GetConfig().GetString("setting.registry.list_ignore_namespaces")
	ignoredNamespaces := strings.Split(ignoreNamespaceConfig, ",")
	respList := make([]NamespaceRespInfo, 0)
	for _, item := range list {
		if !isAdminUser && slices.Contains(ignoredNamespaces, item.Name) {
			continue
		}
		info := NamespaceRespInfo{
			RegistryNamespace: *item,
			CanEdit:           registrycommon.ResourcePermission{}.IsCanOperate(ctx, item.UserID),
		}
		respList = append(respList, info)
	}

	type NamespaceRegistryCount struct {
		Namespace string `json:"namespace"`
		Count     int    `json:"count"`
	}
	var namespacesRegistryCount []NamespaceRegistryCount
	var namespacesRegistryCountMap = make(map[string]NamespaceRegistryCount)
	var namespaces []string
	for _, item := range respList {
		namespaces = append(namespaces, item.Name)
	}
	if len(namespaces) > 0 {
		_ = dao.Q.RegistryRepository.
			Select(dao.Q.RegistryRepository.Namespace, dao.RegistryRepository.Namespace.Count().As("Count")).
			Where(dao.Q.RegistryRepository.Namespace.In(namespaces...)).
			Group(dao.Q.RegistryRepository.Namespace).
			Scan(&namespacesRegistryCount)
		for _, item := range namespacesRegistryCount {
			namespacesRegistryCountMap[item.Namespace] = item
		}
	}

	c.JsonResponseWithoutError(ctx, map[string]interface{}{
		"list":                     respList,
		"namespace_registry_count": namespacesRegistryCountMap,
	})
}

func (c Namespace) SubNamespaceList(ctx *gin.Context) {
	type ParamsValidate struct {
		Namespace    string `json:"namespace" binding:"required"`
		ResourceType string `json:"resource_type" binding:"required,max=64"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	user := logic2.User{}.GetUser(ctx)
	if user != nil && !(logic2.User{}.IsAdminUser(user)) {
		exists := registrycommon.ResourcePermission{}.UserHasPermissionByResource(int(user.ID), params.Namespace, registrycommon.PermissionResourceTypeNamespace)
		if !exists {
			c.JsonResponseWithServerError(ctx, errors.New("user no permission"))
			return
		}
	}

	list, err := registrycommon.Namespace{}.ListSubNamespacesByNamespace(params.Namespace, params.ResourceType)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	c.JsonResponseWithoutError(ctx, map[string]interface{}{
		"list": list,
	})
}

func (c Namespace) Edit(ctx *gin.Context) {
	type ParamsValidate struct {
		Id          int    `json:"id" binding:"required"`
		Name        string `json:"name" binding:"required"`
		VisibleType uint8  `json:"visible_type" binding:"required,oneof=1 2 4"`
		Visible     *bool  `json:"visible"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	curNamespace, _ := registrycommon.Namespace{}.GetByIdWithUser(params.Id, logic2.User{}.GetUser(ctx))
	if curNamespace == nil {
		c.JsonResponseWithServerError(ctx, errors.New("namespace 不存在"))
		return
	}

	oldNamespaceName := curNamespace.Name
	namespaceRenamed := oldNamespaceName != params.Name
	if namespaceRenamed {
		if oldNamespaceName == logic2.DefaultNamespace {
			c.JsonResponseWithServerError(ctx, errors.New("default namespace 不可重命名"))
			return
		}
		existsNamespace, _ := registrycommon.Namespace{}.GetByName(params.Name)
		if existsNamespace != nil && existsNamespace.ID != curNamespace.ID {
			c.JsonResponseWithServerError(ctx, errors.New("namespace 已存在"))
			return
		}
		curNamespace.Name = params.Name
	}
	curNamespace.VisibleType = int32(params.VisibleType)
	if params.Visible != nil {
		curNamespace.Visible = *params.Visible
	}

	err := dao.Q.Transaction(func(tx *dao.Query) error {
		if namespaceRenamed {
			if err := (registrycommon.Namespace{}).RenameInTransaction(ctx.Request.Context(), tx, curNamespace.ID, params.Name); err != nil {
				return err
			}
		}

		query := tx.RegistryNamespace
		_, err := query.Where(query.ID.Eq(curNamespace.ID)).Select(query.Name, query.VisibleType, query.Visible).Updates(entity.RegistryNamespace{
			Name: curNamespace.Name, VisibleType: curNamespace.VisibleType, Visible: curNamespace.Visible,
		})
		return err
	})
	if err != nil {
		c.JsonResponseWithServerError(ctx, errors.New("namespace 保存失败, err: "+err.Error()))
		return
	}

	c.JsonSuccessResponse(ctx)
}

func (c Namespace) Delete(ctx *gin.Context) {
	type ParamsValidate struct {
		Id int `json:"id" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	curNamespace, _ := registrycommon.Namespace{}.GetByIdWithUser(params.Id, logic2.User{}.GetUser(ctx))
	if curNamespace == nil {
		c.JsonResponseWithServerError(ctx, errors.New("namespace 不存在"))
		return
	}
	if curNamespace.Name == logic2.DefaultNamespace {
		c.JsonResponseWithServerError(ctx, errors.New("default namespace 不可删除"))
		return
	}

	err := (registrycommon.Namespace{}).Delete(ctx.Request.Context(), curNamespace.ID)

	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	c.JsonSuccessResponse(ctx)
}
