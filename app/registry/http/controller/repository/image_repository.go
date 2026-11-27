package repository

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	registrycommon "github.com/w7panel/w7panel-zpk/app/registry/logic/common"
	registryrepository "github.com/w7panel/w7panel-zpk/app/registry/logic/repository"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/function"
	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/controller"
	"oras.land/oras-go/v2"
)

type Repository struct {
	controller.Abstract
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

	curRepository, _ := (registrycommon.Resource{}).GetByID(ctx.Request.Context(), int32(params.Id), "repository", logic2.User{}.GetUser(ctx), false)
	if curRepository == nil {
		c.JsonResponseWithServerError(ctx, errors.New("repository 不存在"))
		return
	}

	repositoryName := logic2.BuildRepositoryName(curRepository.Name, curRepository.Namespace)
	registryClient := logic2.GetDefaultRegistryClient()
	tags, _ := registryClient.ListTags(repositoryName)
	rebuiltImages, imageErr := (registryrepository.Repository{}).RebuiltImages(curRepository.ID)
	if imageErr != nil {
		c.JsonResponseWithServerError(ctx, imageErr)
		return
	}
	tagInfos := make([]registryrepository.TagInfo, 0)
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

			tagInfo, _ := registryrepository.Repository{}.GetRepositoryTagInfo(repositoryName, tag, manifest)
			if tagInfo == nil {
				tagInfo = &registryrepository.TagInfo{}
			}
			if _, exists := tagUpdateTimeMap[tag]; exists && !tagUpdateTimeMap[tag].IsZero() {
				tagInfo.UpdatedAt = tagUpdateTimeMap[tag].Format(time.RFC3339)
			}
			tagInfos = append(tagInfos, registryrepository.TagInfo{
				Rebuilt:        rebuiltImages[tag],
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

	curRepository, _ := (registrycommon.Resource{}).GetByID(ctx.Request.Context(), int32(params.Id), "repository", logic2.User{}.GetUser(ctx), true)
	if curRepository == nil {
		c.JsonResponseWithServerError(ctx, errors.New("repository 不存在"))
		return
	}

	result, err := (registryrepository.Repository{}).DeleteTags(ctx.Request.Context(), curRepository, []string{params.Tag})
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	if len(result.FailedTags) > 0 {
		c.JsonResponseWithServerError(ctx, errors.New(result.FailedTags[0].Error))
		return
	}
	c.JsonSuccessResponse(ctx)
}

func (c Repository) BatchDelTags(ctx *gin.Context) {
	type ParamsValidate struct {
		Id   int      `json:"id" binding:"required,min=1"`
		Tags []string `json:"tags" binding:"required,min=1,max=1000,dive,required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	curRepository, _ := (registrycommon.Resource{}).GetByID(ctx.Request.Context(), int32(params.Id), "repository", logic2.User{}.GetUser(ctx), true)
	if curRepository == nil {
		c.JsonResponseWithServerError(ctx, errors.New("repository 不存在"))
		return
	}
	result, err := (registryrepository.Repository{}).DeleteTags(ctx.Request.Context(), curRepository, params.Tags)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	c.JsonResponseWithoutError(ctx, result)
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

	curRepository, _ := (registrycommon.Resource{}).GetByID(ctx.Request.Context(), int32(params.Id), "repository", logic2.User{}.GetUser(ctx), true)
	if curRepository == nil {
		c.JsonResponseWithServerError(ctx, errors.New("repository 不存在"))
		return
	}

	curRepositoryName := logic2.BuildRepositoryName(curRepository.Name, curRepository.Namespace)
	if curRepository.Name != params.Name {
		existsRepository, _ := (registrycommon.Resource{}).GetByNameAndNamespace(ctx.Request.Context(), "repository", params.Name, params.Namespace)
		if existsRepository != nil && existsRepository.ID != curRepository.ID {
			c.JsonResponseWithServerError(ctx, errors.New("repository 已存在"))
			return
		}
		curRepository.Name = params.Name
	}
	curRepository.Namespace = params.Namespace
	curRepository.Desc = params.Desc
	curRepository.VisibleType = int32(params.VisibleType)
	err := (registrycommon.Resource{}).Save(ctx.Request.Context(), curRepository, curRepositoryName)
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

	curRepository, _ := (registrycommon.Resource{}).GetByID(ctx.Request.Context(), int32(params.Id), "repository", logic2.User{}.GetUser(ctx), true)
	if curRepository == nil {
		c.JsonResponseWithServerError(ctx, errors.New("repository 不存在"))
		return
	}

	err := (registryrepository.Repository{}).Delete(ctx.Request.Context(), curRepository)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	c.JsonSuccessResponse(ctx)
}
