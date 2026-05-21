package controller

import (
	"errors"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/app/registry/logic"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/controller"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
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

	k8sConfig := ""
	if facade.GetConfig().GetString("app.env") == "debug" {
		k8sConfig = "apiVersion: v1\nclusters:\n- cluster:\n    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJkekNDQVIyZ0F3SUJBZ0lCQURBS0JnZ3Foa2pPUFFRREFqQWpNU0V3SHdZRFZRUUREQmhyTTNNdGMyVnkKZG1WeUxXTmhRREUzTlRJMU5qUTRPVE13SGhjTk1qVXdOekUxTURjek5EVXpXaGNOTXpVd056RXpNRGN6TkRVegpXakFqTVNFd0h3WURWUVFEREJock0zTXRjMlZ5ZG1WeUxXTmhRREUzTlRJMU5qUTRPVE13V1RBVEJnY3Foa2pPClBRSUJCZ2dxaGtqT1BRTUJCd05DQUFUYTd6Z2EzU1dJbXU2OXZtUWdhQmpjNDdManhPL3d0NUc3dE9wSThSYlYKWnYySmRpTXlrRSs5RVRZSGVnc0RMS3YxZWY5QS83UlNnMmRPSmlwTWFSdGhvMEl3UURBT0JnTlZIUThCQWY4RQpCQU1DQXFRd0R3WURWUjBUQVFIL0JBVXdBd0VCL3pBZEJnTlZIUTRFRmdRVXpFUGNOOFdGNlBEZ3l4L3hnaVc1Cmo3c25GVzh3Q2dZSUtvWkl6ajBFQXdJRFNBQXdSUUlnZUlYYXliQkM1UGk0L0JIcTdFTFpUZXZZN0tudFVIQWEKZXVvK2RtOCsvOFFDSVFEcFU1K2psUllqbjdZL2lxZW4vUGNXMEdpTThwQUJQMGRRbzZvUVRtVzB2QT09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K\n    server: https://218.23.2.55:6443\n  name: default\ncontexts:\n- context:\n    cluster: default\n    user: default\n  name: default\ncurrent-context: default\nkind: Config\npreferences: {}\nusers:\n- name: default\n  user:\n    client-certificate-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJrVENDQVRlZ0F3SUJBZ0lJZG1PMlZtUnJUN3N3Q2dZSUtvWkl6ajBFQXdJd0l6RWhNQjhHQTFVRUF3d1kKYXpOekxXTnNhV1Z1ZEMxallVQXhOelV5TlRZME9Ea3pNQjRYRFRJMU1EY3hOVEEzTXpRMU0xb1hEVEkyTURjeApOVEEzTXpRMU0xb3dNREVYTUJVR0ExVUVDaE1PYzNsemRHVnRPbTFoYzNSbGNuTXhGVEFUQmdOVkJBTVRESE41CmMzUmxiVHBoWkcxcGJqQlpNQk1HQnlxR1NNNDlBZ0VHQ0NxR1NNNDlBd0VIQTBJQUJHM1loS2tKTS9waGV0UkEKbE4wTXlnRmg4b0dPT3ZhR2ZTVUU2ZVhTOHh0Sjk1dCtvUHc5SmRoLzhLMlJqSDYwcTZ1Wm0wblZDcHZiVXlibAo4UXcvSGtDalNEQkdNQTRHQTFVZER3RUIvd1FFQXdJRm9EQVRCZ05WSFNVRUREQUtCZ2dyQmdFRkJRY0RBakFmCkJnTlZIU01FR0RBV2dCUm9NUXMxTmJRcGErS0xsVW1OZkxlcHRQWXNsREFLQmdncWhrak9QUVFEQWdOSUFEQkYKQWlFQW81Smc1ais3S3Q1dXBuVGdISUR5RU45NG1XWStKazNyTUR3MkhhVXppeUFDSUE0bTA2VVJneGxYbEFJLwpTWmFPcFJIdHpFSU9pam15UVRHYWszOXc3bWhsCi0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0KLS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJlRENDQVIyZ0F3SUJBZ0lCQURBS0JnZ3Foa2pPUFFRREFqQWpNU0V3SHdZRFZRUUREQmhyTTNNdFkyeHAKWlc1MExXTmhRREUzTlRJMU5qUTRPVE13SGhjTk1qVXdOekUxTURjek5EVXpXaGNOTXpVd056RXpNRGN6TkRVegpXakFqTVNFd0h3WURWUVFEREJock0zTXRZMnhwWlc1MExXTmhRREUzTlRJMU5qUTRPVE13V1RBVEJnY3Foa2pPClBRSUJCZ2dxaGtqT1BRTUJCd05DQUFSV2F4YVJ6a3FISHErRFRhbVlvVEZwYkI0Sm5NMjh1MFhGUFhQelQ4N3UKL2hjaVBCRkN3WGpCYmRBRExMa3JSR0tqSFpyQmQ0cCt1YVh3V0N2T0RuSHRvMEl3UURBT0JnTlZIUThCQWY4RQpCQU1DQXFRd0R3WURWUjBUQVFIL0JBVXdBd0VCL3pBZEJnTlZIUTRFRmdRVWFERUxOVFcwS1d2aWk1VkpqWHkzCnFiVDJMSlF3Q2dZSUtvWkl6ajBFQXdJRFNRQXdSZ0loQUxNTDQ2UmpDRS9xdGJYUm1pazUvR0JETHd5eUQxS2cKekVpeng1cVNWZUhEQWlFQXVFZk45QStoaStsaWJoR2cxcjZ2UjFDQjUzdjdyOW1Qb210SUlVd2doUWM9Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K\n    client-key-data: LS0tLS1CRUdJTiBFQyBQUklWQVRFIEtFWS0tLS0tCk1IY0NBUUVFSUJwZVBRdk1URGsyY3NjMm1DSzhoQjA1UDF4Rlp2NXZ3NHVrWWVET1d1ekFvQW9HQ0NxR1NNNDkKQXdFSG9VUURRZ0FFYmRpRXFRa3orbUY2MUVDVTNRektBV0h5Z1k0NjlvWjlKUVRwNWRMekcwbjNtMzZnL0QwbAoySC93clpHTWZyU3JxNW1iU2RVS205dFRKdVh4REQ4ZVFBPT0KLS0tLS1FTkQgRUMgUFJJVkFURSBLRVktLS0tLQo="
	}
	restConfig, err := logic.Deploy{}.MakeK8sConfig(k8sConfig)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	result, err := url.Parse(restConfig.Host)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	// 3. 设置HTTP代理
	ctx.Request.URL.Path = params.Path
	ctx.Request.Header.Set("Authorization", "Bearer "+restConfig.BearerToken)
	proxy := httputil.NewSingleHostReverseProxy(result)
	tr, err := rest.TransportFor(restConfig)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	proxy.Transport = tr
	proxy.ServeHTTP(ctx.Writer, ctx.Request)

	return
}

func (c Deploy) GetK8sNamespace(ctx *gin.Context) {
	type ParamsValidate struct {
		K8sConfig string `json:"k8s_config"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	client, err := logic.Deploy{}.GetK8sClient(params.K8sConfig)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	list, err := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	namespaces := make([]string, 0)
	for _, item := range list.Items {
		namespaces = append(namespaces, item.ObjectMeta.Name)
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

	client, err := logic.Deploy{}.GetK8sClient(params.K8sConfig)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	appList := make([]string, 0)
	if params.K8sControllerType == logic.K8sControllerTypeDeployments {
		list, err := client.AppsV1().Deployments(params.K8sNamespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			c.JsonResponseWithServerError(ctx, err)
			return
		}
		for _, item := range list.Items {
			appList = append(appList, item.ObjectMeta.Name)
		}
	} else if params.K8sControllerType == logic.K8sControllerTypeDaemonsets {
		list, err := client.AppsV1().DaemonSets(params.K8sNamespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			c.JsonResponseWithServerError(ctx, err)
			return
		}
		for _, item := range list.Items {
			appList = append(appList, item.ObjectMeta.Name)
		}
	} else if params.K8sControllerType == logic.K8sControllerTypeStatefulSet {
		list, err := client.AppsV1().StatefulSets(params.K8sNamespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			c.JsonResponseWithServerError(ctx, err)
			return
		}
		for _, item := range list.Items {
			appList = append(appList, item.ObjectMeta.Name)
		}
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

	client, err := logic.Deploy{}.GetK8sClient(params.K8sConfig)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	containers := make([]string, 0)
	containerList := make([]corev1.Container, 0)
	if params.K8sControllerType == logic.K8sControllerTypeDeployments {
		info, err := client.AppsV1().Deployments(params.K8sNamespace).Get(ctx, params.K8sAppName, metav1.GetOptions{})
		if err != nil {
			c.JsonResponseWithServerError(ctx, err)
			return
		}
		containerList = info.Spec.Template.Spec.Containers
	} else if params.K8sControllerType == logic.K8sControllerTypeDaemonsets {
		info, err := client.AppsV1().DaemonSets(params.K8sNamespace).Get(ctx, params.K8sAppName, metav1.GetOptions{})
		if err != nil {
			c.JsonResponseWithServerError(ctx, err)
			return
		}
		containerList = info.Spec.Template.Spec.Containers
	} else if params.K8sControllerType == logic.K8sControllerTypeStatefulSet {
		info, err := client.AppsV1().StatefulSets(params.K8sNamespace).Get(ctx, params.K8sAppName, metav1.GetOptions{})
		if err != nil {
			c.JsonResponseWithServerError(ctx, err)
			return
		}
		containerList = info.Spec.Template.Spec.Containers
	}
	for _, item := range containerList {
		containers = append(containers, item.Name)
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

	curRepository, _ := logic.Repository{}.GetByIdWithUser(params.RepositoryId, logic2.User{}.GetUser(ctx))
	if curRepository == nil {
		c.JsonResponseWithServerError(ctx, errors.New("repository 不存在"))
		return
	}

	err := dao.Q.RegistryRepositoryDeployRule.Create(&entity.RegistryRepositoryDeployRule{
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

	curRepository, _ := logic.Repository{}.GetByIdWithUser(params.RepositoryId, logic2.User{}.GetUser(ctx))
	if curRepository == nil {
		c.JsonResponseWithServerError(ctx, errors.New("repository 不存在"))
		return
	}

	curRule, _ := dao.Q.RegistryRepositoryDeployRule.Where(dao.Q.RegistryRepositoryDeployRule.ID.Eq(int32(params.Id))).First()
	if curRule == nil {
		c.JsonResponseWithServerError(ctx, errors.New("该规则不存在"))
		return
	}

	_, err := dao.Q.RegistryRepositoryDeployRule.Where(dao.Q.RegistryRepositoryDeployRule.ID.Eq(int32(params.Id))).Updates(entity.RegistryRepositoryDeployRule{
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

	info, err := dao.Q.RegistryRepositoryDeployRule.Where(dao.Q.RegistryRepositoryDeployRule.ID.Eq(int32(params.Id))).First()
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

	_, err := dao.Q.RegistryRepositoryDeployRule.Where(dao.Q.RegistryRepositoryDeployRule.ID.Eq(int32(params.Id))).Delete()
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

	curRepository, _ := logic.Repository{}.GetByIdWithUser(params.RepositoryId, logic2.User{}.GetUser(ctx))
	if curRepository == nil {
		c.JsonResponseWithServerError(ctx, errors.New("repository 不存在"))
		return
	}

	list, _ := dao.Q.RegistryRepositoryDeployRule.
		Where(dao.Q.RegistryRepositoryDeployRule.RepositoryID.Eq(int32(params.RepositoryId))).
		Order(dao.Q.RegistryRepositoryDeployRule.ID.Desc()).
		Find()
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

	list, total, err := dao.Q.RegistryRepositoryDeployRuleMatchLog.
		Where(dao.Q.RegistryRepositoryDeployRuleMatchLog.RuleID.Eq(int32(params.Id))).
		Order(dao.Q.RegistryRepositoryDeployRuleMatchLog.ID.Desc()).
		FindByPage((params.Page-1)*params.PageSize, params.PageSize)
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
