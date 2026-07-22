package controller

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/app/registry/logic"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	"github.com/w7panel/w7panel-zpk/common/function"
	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/controller"
	"gorm.io/gorm"
	"oras.land/oras-go/v2"
)

type RepositoryRespInfo struct {
	entity.RegistryRepository
	CanEdit bool `json:"can_edit"`
}

type Repository struct {
	controller.Abstract
}

func (c Repository) checkNamespacePermission(ctx *gin.Context, namespace string, requirePush bool) error {
	namespaceModel, err := logic.Namespace{}.GetByName(namespace)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("namespace 不存在")
		}
		return err
	}
	if namespaceModel == nil {
		return errors.New("namespace 不存在")
	}

	user := logic2.User{}.GetUser(ctx)
	if user == nil {
		return errors.New("用户信息异常")
	}
	if (logic2.User{}).IsAdminUser(user) || namespaceModel.UserID == user.ID {
		return nil
	}

	permission, err := dao.Q.RegistryUserPermission.
		Where(dao.Q.RegistryUserPermission.UserID.Eq(user.ID)).
		Where(dao.Q.RegistryUserPermission.ResourceType.Eq(string(logic.PermissionResourceTypeNamespace))).
		Where(dao.Q.RegistryUserPermission.ResourceValue.Eq(namespace)).
		First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user no permission")
		}
		return err
	}
	if permission == nil {
		return errors.New("user no permission")
	}
	if requirePush {
		if permission.Action == nil || !slices.Contains(*permission.Action, string(logic.PermissionActionTypePush)) {
			return errors.New("invalid action")
		}
	}

	return nil
}

func (c Repository) Create(ctx *gin.Context) {
	type ParamsValidate struct {
		Name        string `json:"name" binding:"required"`
		Namespace   string `json:"namespace" binding:"required"`
		VisibleType uint8  `json:"visible_type" binding:"required,oneof=1 2 3 4"`
		Desc        string `json:"desc"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	if err := c.checkNamespacePermission(ctx, params.Namespace, true); err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	existsRepository, _ := logic.Repository{}.GetByNameAndNamespace(params.Name, params.Namespace)
	if existsRepository != nil {
		c.JsonResponseWithServerError(ctx, errors.New("repository 已存在"))
		return
	}

	err := dao.Q.RegistryRepository.Create(&entity.RegistryRepository{
		Name:        params.Name,
		Namespace:   params.Namespace,
		VisibleType: int32(params.VisibleType),
		Desc:        params.Desc,
		UserID:      logic2.User{}.GetUser(ctx).ID,
	})
	if err != nil {
		c.JsonResponseWithServerError(ctx, errors.New("repository 创建失败, err: "+err.Error()))
		return
	}

	c.JsonSuccessResponse(ctx)
}

func (c Repository) Info(ctx *gin.Context) {
	type ParamsValidate struct {
		Id int `json:"id" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	user := logic2.User{}.GetUser(ctx)
	curRepository, _ := logic.Repository{}.GetByIdWithUserPermission(params.Id, user)
	if curRepository == nil {
		c.JsonResponseWithServerError(ctx, errors.New("repository 不存在"))
		return
	}
	if curRepository.Registry == "" {
		curRepository.Registry = facade.GetConfig().GetString("registry_cli.default.external_domain")
	}

	respInfo := RepositoryRespInfo{
		RegistryRepository: *curRepository,
		CanEdit:            logic.Permission{}.IsCanOperate(ctx, curRepository.UserID),
	}

	c.JsonResponseWithoutError(ctx, respInfo)
}

func (c Repository) List(ctx *gin.Context) {
	type ParamsValidate struct {
		Namespace    string `json:"namespace"`
		SubNamespace string `json:"sub_namespace"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	user := logic2.User{}.GetUser(ctx)
	query := dao.Q.RegistryRepository.Order(dao.Q.RegistryRepository.CreatedAt.Desc())
	if params.Namespace != "" {
		if err := c.checkNamespacePermission(ctx, params.Namespace, false); err != nil {
			c.JsonResponseWithServerError(ctx, err)
			return
		}
		query = query.Where(dao.Q.RegistryRepository.Namespace.Eq(params.Namespace))
	} else if user != nil && !(logic2.User{}.IsAdminUser(user)) {
		namespacePermissions, _ := dao.Q.RegistryUserPermission.
			Where(dao.Q.RegistryUserPermission.UserID.Eq(user.ID)).
			Where(dao.Q.RegistryUserPermission.ResourceType.Eq(string(logic.PermissionResourceTypeNamespace))).
			Find()
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

	ignoreNamespaceConfig := facade.GetConfig().GetString("setting.registry.list_ignore_namespaces")
	ignoredNamespaces := strings.Split(ignoreNamespaceConfig, ",")
	retLst := make([]RepositoryRespInfo, 0)
	for index, item := range list {
		if item.Namespace != "" && slices.Contains(ignoredNamespaces, item.Namespace) {
			continue
		}
		if item.Registry == "" {
			item.Registry = facade.GetConfig().GetString("registry_cli.default.external_domain")
			list[index] = item
		}

		respInfo := RepositoryRespInfo{
			RegistryRepository: *item,
			CanEdit:            logic.Permission{}.IsCanOperate(ctx, item.UserID),
		}

		retLst = append(retLst, respInfo)
	}

	c.JsonResponseWithoutError(ctx, map[string]interface{}{
		"list": retLst,
		"registry": map[string]interface{}{
			"server_url": facade.GetConfig().GetString("registry_cli.default.url"),
		},
	})
}

func (c Repository) Tags(ctx *gin.Context) {
	type ParamsValidate struct {
		Id       int `json:"id" binding:"required"`
		Page     int `json:"page"`
		PageSize int `json:"page_size"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	if params.PageSize <= 0 {
		params.PageSize = 10
	}
	if params.Page <= 0 {
		params.Page = 1
	}

	curRepository, _ := logic.Repository{}.GetByIdWithUserPermission(params.Id, logic2.User{}.GetUser(ctx))
	if curRepository == nil {
		c.JsonResponseWithServerError(ctx, errors.New("repository 不存在"))
		return
	}

	repositoryName := logic2.BuildRepositoryName(curRepository.Name, curRepository.Namespace)
	registryClient := logic2.GetDefaultRegistryClient()
	tags, _ := registryClient.ListTags(repositoryName)
	tagInfos := make([]logic.TagInfo, 0)
	totalCount := 0
	if tags != nil {
		totalCount = len(tags)
		tagUpdateTimeMap := make(map[string]*time.Time)
		localTags, _ := dao.Q.RegistryRepositoryTag.Where(dao.Q.RegistryRepositoryTag.RepositoryID.Eq(curRepository.ID)).Find()
		for _, item := range localTags {
			if item.LatestPushAt != nil {
				tagUpdateTimeMap[item.Name] = item.LatestPushAt
			}
		}

		function.SortTagsDesc(tags)

		start := (params.Page - 1) * params.PageSize
		if start > len(tags) {
			tags = []string{}
		} else {
			end := start + params.PageSize
			if end > len(tags) {
				end = len(tags) // 结束位置超出时截断
			}
			tags = tags[start:end]
		}

		for _, tag := range tags {
			manifest, digest, err := registryClient.PullManifest(repositoryName, tag)
			if err != nil {
				slog.Error("获取tag详情失败", "err", err, "repositoryName", repositoryName, "tag", tag)
				continue
			}

			tagInfo, _ := logic.Repository{}.GetRepositoryTagInfo(repositoryName, tag, manifest)
			if tagInfo == nil {
				tagInfo = &logic.TagInfo{}
			}
			if _, exists := tagUpdateTimeMap[tag]; exists && !tagUpdateTimeMap[tag].IsZero() {
				tagInfo.UpdatedAt = tagUpdateTimeMap[tag].Format(time.RFC3339)
			}

			tagInfos = append(tagInfos, logic.TagInfo{
				RepositoryName: curRepository.Name,
				TagName:        tagInfo.TagName,
				Manifest:       manifest,
				Digest:         digest,
				Type:           tagInfo.Type,
				Platform:       tagInfo.Platform,
				CreatedAt:      tagInfo.CreatedAt,
				UpdatedAt:      tagInfo.UpdatedAt,
				Size:           tagInfo.Size,
			})
		}
	}

	c.JsonResponseWithoutError(ctx, map[string]interface{}{
		"list":  tagInfos,
		"total": totalCount,
		"page":  params.Page,
		"size":  params.PageSize,
	})
}

func (c Repository) DelTag(ctx *gin.Context) {
	type ParamsValidate struct {
		Id  int    `json:"id" binding:"required"`
		Tag string `json:"tag" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	curRepository, _ := logic.Repository{}.GetByIdWithUser(params.Id, logic2.User{}.GetUser(ctx))
	if curRepository == nil {
		c.JsonResponseWithServerError(ctx, errors.New("repository 不存在"))
		return
	}

	repositoryName := logic2.BuildRepositoryName(curRepository.Name, curRepository.Namespace)
	registryClient := logic2.GetDefaultRegistryClient()
	err := registryClient.DeleteManifest(repositoryName, params.Tag)
	if err != nil {
		slog.Error("删除 tag 失败", "err", err, "repository", repositoryName, "tag", params.Tag)
	}

	c.JsonSuccessResponse(ctx)
}

func (c Repository) Edit(ctx *gin.Context) {
	type ParamsValidate struct {
		Id          int    `json:"id" binding:"required"`
		Name        string `json:"name" binding:"required"`
		Namespace   string `json:"namespace" binding:"required"`
		VisibleType uint8  `json:"visible_type" binding:"required,oneof=1 2 3 4"`
		Desc        string `json:"desc"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	curRepository, _ := logic.Repository{}.GetByIdWithUser(params.Id, logic2.User{}.GetUser(ctx))
	if curRepository == nil {
		c.JsonResponseWithServerError(ctx, errors.New("repository 不存在"))
		return
	}

	curRepositoryName := logic2.BuildRepositoryName(curRepository.Name, curRepository.Namespace)
	if curRepository.Name != params.Name {
		existsRepository, _ := logic.Repository{}.GetByNameAndNamespace(params.Name, params.Namespace)
		if existsRepository != nil && existsRepository.ID != curRepository.ID {
			c.JsonResponseWithServerError(ctx, errors.New("repository 已存在"))
			return
		}
		curRepository.Name = params.Name
	}
	curRepository.Namespace = params.Namespace
	curRepository.Desc = params.Desc
	curRepository.VisibleType = int32(params.VisibleType)
	err := dao.Q.RegistryRepository.Save(curRepository)
	if err != nil {
		c.JsonResponseWithServerError(ctx, errors.New("repository 保存失败, err: "+err.Error()))
		return
	}

	newRepositoryName := logic2.BuildRepositoryName(curRepository.Name, curRepository.Namespace)
	if curRepositoryName != newRepositoryName {
		registryClient := logic2.GetDefaultRegistryClient()
		tags, err := registryClient.ListTags(curRepositoryName)
		if err != nil {
			c.JsonResponseWithServerError(ctx, errors.New("获取tag失败, err:"+err.Error()))
			return
		}

		if len(tags) > 0 {
			var srcRegistryOci oras.GraphTarget
			var destRegistryOci oras.GraphTarget
			var err error
			srcRegistryOci, err = logic2.GetDefaultRemoteOci(curRepositoryName)
			destRegistryOci, err = logic2.GetDefaultRemoteOci(newRepositoryName)
			if err != nil {
				c.JsonResponseWithServerError(ctx, errors.New("初始化源oci, err:"+err.Error()))
				return
			}

			for _, tag := range tags {
				_, err = oras.Copy(context.Background(), srcRegistryOci, tag, destRegistryOci, tag, oras.DefaultCopyOptions)
				if err != nil {
					c.JsonResponseWithServerError(ctx, errors.New("镜像复制, err:"+err.Error()))
					return
				}
			}
		}
	}

	c.JsonSuccessResponse(ctx)
}

func (c Repository) Delete(ctx *gin.Context) {
	type ParamsValidate struct {
		Id int `json:"id" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	curRepository, _ := logic.Repository{}.GetByIdWithUser(params.Id, logic2.User{}.GetUser(ctx))
	if curRepository == nil {
		c.JsonResponseWithServerError(ctx, errors.New("repository 不存在"))
		return
	}

	err := dao.Q.Transaction(func(tx *dao.Query) error {
		_, err := tx.RegistryRepository.Delete(curRepository)
		if err != nil {
			return err
		}
		resourceValue := logic2.BuildRepositoryName(curRepository.Name, curRepository.Namespace)
		return logic.Permission{}.DelPermissionByResource(tx, resourceValue, logic.PermissionResourceTypeRepository)
	})
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	c.JsonSuccessResponse(ctx)
}
