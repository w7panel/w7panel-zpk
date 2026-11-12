package controller

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/app/workflow/logic"
	"github.com/w7panel/w7panel-zpk/common/entity"
	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/controller"
)

type Deploy struct {
	controller.Abstract
	Access func(*gin.Context, int32) error
}

func (c Deploy) getRuleWithRepository(ctx *gin.Context, ruleID int) (*entity.RegistryRepositoryDeployRule, error) {
	rule, err := (logic.Deploy{}).Rule(int32(ruleID))
	if err != nil {
		return nil, err
	}

	if c.Access == nil {
		return nil, errors.New("未配置工作流访问校验")
	}
	if err := c.Access(ctx, rule.RepositoryID); err != nil {
		return nil, err
	}

	return rule, nil
}

func (c Deploy) K8sProxy(ctx *gin.Context) {
	type ParamsValidate struct {
		Path string `uri:"path" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	if err := (logic.Deploy{}).Proxy(ctx.Writer, ctx.Request, params.Path); err != nil {
		c.JsonResponseWithServerError(ctx, err)
	}
}

func (c Deploy) GetK8sNamespace(ctx *gin.Context) {
	type ParamsValidate struct {
		K8sConfig string `json:"k8s_config"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	if !c.allowClusterConfig(ctx, params.K8sConfig) {
		return
	}
	namespaces, err := (logic.Deploy{}).Namespaces(ctx.Request.Context(), params.K8sConfig)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	c.JsonResponseWithoutError(ctx, map[string]interface{}{"list": namespaces})
}

func (c Deploy) GetK8sApps(ctx *gin.Context) {
	type ParamsValidate struct {
		K8sConfig         string `json:"k8s_config"`
		K8sNamespace      string `json:"k8s_namespace" binding:"required"`
		K8sControllerType string `json:"k8s_controller_type" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	if !c.allowClusterConfig(ctx, params.K8sConfig) {
		return
	}
	appList, err := (logic.Deploy{}).Applications(ctx.Request.Context(), params.K8sConfig, params.K8sNamespace, params.K8sControllerType)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	c.JsonResponseWithoutError(ctx, map[string]interface{}{"list": appList})
}

func (c Deploy) GetK8sAppContainers(ctx *gin.Context) {
	type ParamsValidate struct {
		K8sConfig         string `json:"k8s_config"`
		K8sNamespace      string `json:"k8s_namespace" binding:"required"`
		K8sControllerType string `json:"k8s_controller_type" binding:"required"`
		K8sAppName        string `json:"k8s_app_name" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	if !c.allowClusterConfig(ctx, params.K8sConfig) {
		return
	}
	containers, err := (logic.Deploy{}).Containers(ctx.Request.Context(), params.K8sConfig, params.K8sNamespace, params.K8sControllerType, params.K8sAppName)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	c.JsonResponseWithoutError(ctx, map[string]interface{}{"list": containers})
}

func (c Deploy) AddRule(ctx *gin.Context) {
	type ParamsValidate struct {
		K8sConfig         string `json:"k8s_config"`
		RepositoryId      int    `json:"repository_id" binding:"required"`
		RepositoryTag     string `json:"repository_tag" binding:"required"`
		DeployType        int    `json:"deploy_type" binding:"required"`
		MatchType         int    `json:"match_type"`
		K8sNamespace      string `json:"k8s_namespace" binding:"required"`
		K8sControllerType string `json:"k8s_controller_type" binding:"required"`
		K8sAppName        string `json:"k8s_app_name" binding:"required"`
		K8sContainerName  string `json:"k8s_container_name" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	if c.Access == nil {
		c.JsonResponseWithServerError(ctx, errors.New("未配置工作流访问校验"))
		return
	}
	if err := c.Access(ctx, int32(params.RepositoryId)); err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	if !c.allowClusterConfig(ctx, params.K8sConfig) {
		return
	}

	err := (logic.Deploy{}).SaveRule(entity.RegistryRepositoryDeployRule{
		RepositoryID:      int32(params.RepositoryId),
		DeployType:        int32(params.DeployType),
		MatchType:         int32(params.MatchType),
		K8sNamespace:      params.K8sNamespace,
		K8sControllerType: params.K8sControllerType,
		K8sAppName:        params.K8sAppName,
		K8sContainerName:  params.K8sContainerName,
		TagName:           params.RepositoryTag,
		K8sConfig:         params.K8sConfig,
	})
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	c.JsonSuccessResponse(ctx)
}

func (c Deploy) EditRule(ctx *gin.Context) {
	type ParamsValidate struct {
		Id                int    `json:"id" binding:"required"`
		K8sConfig         string `json:"k8s_config"`
		RepositoryId      int    `json:"repository_id" binding:"required"`
		RepositoryTag     string `json:"repository_tag" binding:"required"`
		DeployType        int    `json:"deploy_type" binding:"required"`
		MatchType         int    `json:"match_type"`
		K8sNamespace      string `json:"k8s_namespace" binding:"required"`
		K8sControllerType string `json:"k8s_controller_type" binding:"required"`
		K8sAppName        string `json:"k8s_app_name" binding:"required"`
		K8sContainerName  string `json:"k8s_container_name" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	_, err := c.getRuleWithRepository(ctx, params.Id)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	if c.Access == nil {
		c.JsonResponseWithServerError(ctx, errors.New("未配置工作流访问校验"))
		return
	}
	if err := c.Access(ctx, int32(params.RepositoryId)); err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	if !c.allowClusterConfig(ctx, params.K8sConfig) {
		return
	}

	err = (logic.Deploy{}).SaveRule(entity.RegistryRepositoryDeployRule{
		ID:                int32(params.Id),
		RepositoryID:      int32(params.RepositoryId),
		DeployType:        int32(params.DeployType),
		MatchType:         int32(params.MatchType),
		K8sNamespace:      params.K8sNamespace,
		K8sControllerType: params.K8sControllerType,
		K8sAppName:        params.K8sAppName,
		K8sContainerName:  params.K8sContainerName,
		TagName:           params.RepositoryTag,
		K8sConfig:         params.K8sConfig,
	})
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	c.JsonSuccessResponse(ctx)
}

func (c Deploy) QueryRule(ctx *gin.Context) {
	type ParamsValidate struct {
		Id int `json:"id" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	info, err := c.getRuleWithRepository(ctx, params.Id)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	info, err = (logic.Deploy{}).RuleInfo(info)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	c.JsonResponseWithoutError(ctx, map[string]interface{}{"info": info})
}

func (c Deploy) DelRule(ctx *gin.Context) {
	type ParamsValidate struct {
		Id int `json:"id" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	rule, err := c.getRuleWithRepository(ctx, params.Id)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	err = (logic.Deploy{}).DeleteRule(rule)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	c.JsonSuccessResponse(ctx)
}

func (c Deploy) ListRule(ctx *gin.Context) {
	type ParamsValidate struct {
		RepositoryId int `json:"repository_id" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	if c.Access == nil {
		c.JsonResponseWithServerError(ctx, errors.New("未配置工作流访问校验"))
		return
	}
	if err := c.Access(ctx, int32(params.RepositoryId)); err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	list, err := (logic.Deploy{}).Rules(int32(params.RepositoryId))
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	c.JsonResponseWithoutError(ctx, map[string]interface{}{"list": list})
}

func (c Deploy) RuleDeployLog(ctx *gin.Context) {
	type ParamsValidate struct {
		Id       int `json:"id" binding:"required"`
		Page     int `json:"page" binding:"required"`
		PageSize int `json:"page_size" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	_, err := c.getRuleWithRepository(ctx, params.Id)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	list, total, err := (logic.Deploy{}).RuleLogs(int32(params.Id), params.Page, params.PageSize)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	c.JsonResponseWithoutError(ctx, gin.H{
		"total": total,
		"limit": params.PageSize,
		"page":  params.Page,
		"list":  list,
	})
}

func (c Deploy) allowClusterConfig(ctx *gin.Context, content string) bool {
	if content != "" {
		return true
	}
	user := commonlogic.User{}.GetUser(ctx)
	if user == nil || !(commonlogic.User{}).IsAdminUser(user) {
		c.JsonResponseWithServerError(ctx, errors.New("当前集群操作仅管理员可用，请提供目标集群 KUBECONFIG"))
		return false
	}
	return true
}
