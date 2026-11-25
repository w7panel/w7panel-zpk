package controller

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/app/workflow/logic"
	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/controller"
)

type Deploy struct {
	controller.Abstract
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
