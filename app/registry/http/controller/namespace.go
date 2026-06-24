package controller

import (
	"errors"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/app/registry/logic"
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
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	existsNamespace, _ := logic.Namespace{}.GetByName(params.Name)
	if existsNamespace != nil {
		c.JsonResponseWithServerError(ctx, errors.New("namespace 已存在"))
		return
	}

	user := logic2.User{}.GetUser(ctx)
	err := dao.Q.Transaction(func(tx *dao.Query) error {
		err := tx.RegistryNamespace.Create(&entity.RegistryNamespace{
			Name:        params.Name,
			VisibleType: int32(params.VisibleType),
			UserID:      logic2.User{}.GetUser(ctx).ID,
		})
		if err != nil {
			return err
		}

		if user != nil && !(logic2.User{}.IsAdminUser(user)) {
			err = tx.RegistryUserPermission.Create(&entity.RegistryUserPermission{
				UserID:        user.ID,
				ResourceType:  string(logic.PermissionResourceTypeNamespace),
				ResourceValue: params.Name,
				Action:        &accessor.PermissionActionOption{string(logic.PermissionActionTypePush), string(logic.PermissionActionTypePull)},
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
	curNamespace, _ := logic.Namespace{}.GetByIdWithUserPermission(params.Id, user)
	if curNamespace == nil {
		c.JsonResponseWithServerError(ctx, errors.New("namespace 不存在"))
		return
	}

	respInfo := NamespaceRespInfo{
		RegistryNamespace: *curNamespace,
		CanEdit:           logic.Permission{}.IsCanOperate(ctx, curNamespace.UserID),
	}

	c.JsonResponseWithoutError(ctx, respInfo)
}

func (c Namespace) List(ctx *gin.Context) {
	query := dao.Q.RegistryNamespace.Order(dao.Q.RegistryNamespace.CreatedAt.Desc())
	user := logic2.User{}.GetUser(ctx)
	if user != nil && !(logic2.User{}.IsAdminUser(user)) {
		namespacePermissions, _ := dao.Q.RegistryUserPermission.
			Where(dao.Q.RegistryUserPermission.UserID.Eq(user.ID)).
			Where(dao.Q.RegistryUserPermission.ResourceType.Eq(string(logic.PermissionResourceTypeNamespace))).
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
			CanEdit:           logic.Permission{}.IsCanOperate(ctx, item.UserID),
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
		Namespace string `json:"namespace" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	user := logic2.User{}.GetUser(ctx)
	if user != nil && !(logic2.User{}.IsAdminUser(user)) {
		exists := logic.Permission{}.UserHasPermissionByResource(int(user.ID), params.Namespace, logic.PermissionResourceTypeNamespace)
		if !exists {
			c.JsonResponseWithServerError(ctx, errors.New("user no permission"))
			return
		}
	}

	list, err := logic.Namespace{}.ListSubNamespacesByNamespace(params.Namespace)
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
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	curNamespace, _ := logic.Namespace{}.GetByIdWithUser(params.Id, logic2.User{}.GetUser(ctx))
	if curNamespace == nil {
		c.JsonResponseWithServerError(ctx, errors.New("namespace 不存在"))
		return
	}

	oldNamespaceName := curNamespace.Name
	namespaceRenamed := oldNamespaceName != params.Name
	if namespaceRenamed {
		existsNamespace, _ := logic.Namespace{}.GetByName(params.Name)
		if existsNamespace != nil && existsNamespace.ID != curNamespace.ID {
			c.JsonResponseWithServerError(ctx, errors.New("namespace 已存在"))
			return
		}
		curNamespace.Name = params.Name
	}
	curNamespace.VisibleType = int32(params.VisibleType)

	err := dao.Q.Transaction(func(tx *dao.Query) error {
		if namespaceRenamed {
			_, err := tx.RegistryUserPermission.
				Where(tx.RegistryUserPermission.ResourceType.Eq(string(logic.PermissionResourceTypeNamespace))).
				Where(tx.RegistryUserPermission.ResourceValue.Eq(oldNamespaceName)).
				Update(tx.RegistryUserPermission.ResourceValue, params.Name)
			if err != nil {
				return err
			}

			_, err = tx.RegistryRepository.
				Where(tx.RegistryRepository.Namespace.Eq(oldNamespaceName)).
				Update(tx.RegistryRepository.Namespace, params.Name)
			if err != nil {
				return err
			}
		}

		return tx.RegistryNamespace.Save(curNamespace)
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

	curNamespace, _ := logic.Namespace{}.GetByIdWithUser(params.Id, logic2.User{}.GetUser(ctx))
	if curNamespace == nil {
		c.JsonResponseWithServerError(ctx, errors.New("namespace 不存在"))
		return
	}
	if curNamespace.Name == logic2.DefaultNamespace {
		c.JsonResponseWithServerError(ctx, errors.New("default namespace 不可删除"))
		return
	}

	err := dao.Q.Transaction(func(tx *dao.Query) error {
		_, err := tx.RegistryNamespace.Delete(curNamespace)
		if err != nil {
			return err
		}
		return logic.Permission{}.DelPermissionByResource(tx, curNamespace.Name, logic.PermissionResourceTypeNamespace)
	})
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	c.JsonSuccessResponse(ctx)
}
