package controller

import (
	"context"
	"errors"
	"time"

	"github.com/distribution/reference"
	"github.com/gin-gonic/gin"
	workflow "github.com/w7panel/w7panel-zpk/app/workflow/logic"
	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/controller"
	"gorm.io/gorm"
)

type Workflow struct {
	controller.Abstract
}
type workflowParams struct {
	ImageRepository string          `json:"image_repository" binding:"required,max=512"`
	Config          workflow.Config `json:"config"`
	RunID           string          `json:"run_id"`
	TaskID          string          `json:"task_id" binding:"max=40"`
	TailLines       int64           `json:"tail_lines" binding:"min=0,max=2000"`
	Image           string          `json:"image" binding:"max=512"`
	Page            int             `json:"page" binding:"min=0"`
	PageSize        int             `json:"page_size" binding:"min=0,max=100"`
}

func (c Workflow) repository(ctx *gin.Context, imageRepository string) (workflow.RepositoryConfig, error) {
	user := commonlogic.User{}.GetUser(ctx)
	config, err := (workflow.Workflow{}).SavedRepository(imageRepository)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		named, parseErr := reference.ParseNormalizedNamed(imageRepository)
		if parseErr != nil || !reference.IsNameOnly(named) || named.Name() != imageRepository {
			return workflow.RepositoryConfig{}, errors.New("请提供完整镜像仓库地址，不包含 tag 或 digest")
		}
		return workflow.RepositoryConfig{UserID: user.ID, Name: reference.Path(named), Image: workflow.ImageContext{Repository: named.Name(), Host: reference.Domain(named)}}, nil
	}
	if err != nil {
		return workflow.RepositoryConfig{}, err
	}
	if config.UserID != user.ID && !(commonlogic.User{}).IsAdminUser(user) {
		return workflow.RepositoryConfig{}, errors.New("无权访问此工作流")
	}
	return config, nil
}

func (c Workflow) authorize(ctx *gin.Context, params *workflowParams) (workflow.RepositoryConfig, bool) {
	if !c.Validate(ctx, params) {
		return workflow.RepositoryConfig{}, false
	}
	repository, err := c.repository(ctx, params.ImageRepository)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return workflow.RepositoryConfig{}, false
	}
	return repository, true
}
func (c Workflow) Get(ctx *gin.Context) {
	var params workflowParams
	if _, ok := c.authorize(ctx, &params); !ok {
		return
	}
	config, err := (workflow.Workflow{}).Get(params.ImageRepository)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	c.JsonResponseWithoutError(ctx, gin.H{"config": config})
}
func (c Workflow) Save(ctx *gin.Context) {
	var params workflowParams
	repository, ok := c.authorize(ctx, &params)
	if !ok {
		return
	}
	// Current-cluster deployment uses the service's cluster privileges and remains admin-only.
	user := commonlogic.User{}.GetUser(ctx)
	for _, task := range params.Config.Tasks {
		if task.Action == workflow.ActionDeploy && task.K8sConfig == "" && !(commonlogic.User{}).IsAdminUser(user) {
			c.JsonResponseWithServerError(ctx, errors.New("当前集群部署仅管理员可配置，请提供目标集群 KUBECONFIG"))
			return
		}
	}
	params.Config.Repository = repository
	if err := (workflow.Workflow{}).Save(params.ImageRepository, params.Config); err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	c.JsonSuccessResponse(ctx)
}
func (c Workflow) Runs(ctx *gin.Context) {
	var params workflowParams
	if _, ok := c.authorize(ctx, &params); !ok {
		return
	}
	if params.Page == 0 {
		params.Page = 1
	}
	if params.PageSize == 0 {
		params.PageSize = 10
	}
	runs, total, err := (workflow.Workflow{}).Runs(params.ImageRepository, params.TaskID, params.Page, params.PageSize)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	c.JsonResponseWithoutError(ctx, gin.H{"list": runs, "total": total, "page": params.Page, "limit": params.PageSize})
}
func (c Workflow) Logs(ctx *gin.Context) {
	var params workflowParams
	if _, ok := c.authorize(ctx, &params); !ok {
		return
	}
	if params.TaskID == "" {
		c.JsonResponseWithServerError(ctx, errors.New("请选择要查看日志的任务"))
		return
	}
	run, err := (workflow.Workflow{}).Status(params.ImageRepository, params.RunID)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	requestCtx, cancel := context.WithTimeout(ctx.Request.Context(), 30*time.Second)
	defer cancel()
	if params.TailLines == 0 {
		params.TailLines = 100
	}
	logs, err := (workflow.Workflow{}).Logs(requestCtx, run, params.TailLines, params.TaskID)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	c.JsonResponseWithoutError(ctx, gin.H{"log": logs.Log, "archived": logs.Archived, "truncated": logs.Truncated, "status": logs.Status})
}

func (c Workflow) Run(ctx *gin.Context) {
	var params workflowParams
	repository, ok := c.authorize(ctx, &params)
	if !ok {
		return
	}
	if params.Image == "" || params.TaskID == "" {
		c.JsonResponseWithServerError(ctx, errors.New("请填写镜像地址并选择任务"))
		return
	}
	named, err := reference.ParseNormalizedNamed(params.Image)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	if _, canonical := named.(reference.Canonical); canonical {
		c.JsonResponseWithServerError(ctx, errors.New("请使用带 tag 的镜像地址"))
		return
	}
	source := reference.TagNameOnly(named).(reference.NamedTagged)
	user := commonlogic.User{}.GetUser(ctx)
	event, err := (workflow.Workflow{}).ResolveManualImage(ctx.Request.Context(), source)
	if err == nil {
		event.Repository = repository.Name
		event.Image.Repository = repository.Image.Repository
		event.Image.Host = repository.Image.Host
		event.Image.PlainHTTP = repository.Image.PlainHTTP
	}
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	runID, err := (workflow.Workflow{}).TriggerManual(event, params.TaskID, (commonlogic.User{}).IsAdminUser(user))
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	c.JsonResponseWithoutError(ctx, gin.H{"run_id": runID})
}
