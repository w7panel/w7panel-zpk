package controller

import (
	"context"
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	workflow "github.com/w7panel/w7panel-zpk/app/workflow/logic"
	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/controller"
)

type Workflow struct {
	controller.Abstract
	Access func(*gin.Context, int32) error
}
type workflowParams struct {
	RepositoryID int32           `json:"repository_id" binding:"required,min=1"`
	Config       workflow.Config `json:"config"`
	RunID        string          `json:"run_id"`
	TaskID       string          `json:"task_id" binding:"max=40"`
	TailLines    int64           `json:"tail_lines" binding:"min=0,max=2000"`
}

func (c Workflow) authorize(ctx *gin.Context, params *workflowParams) bool {
	if !c.Validate(ctx, params) {
		return false
	}
	user := commonlogic.User{}.GetUser(ctx)
	if user == nil {
		c.JsonResponseWithServerError(ctx, errors.New("用户信息异常"))
		return false
	}
	if c.Access == nil {
		c.JsonResponseWithServerError(ctx, errors.New("未配置工作流访问校验"))
		return false
	}
	if err := c.Access(ctx, params.RepositoryID); err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return false
	}

	return true
}
func (c Workflow) Get(ctx *gin.Context) {
	var params workflowParams
	if !c.authorize(ctx, &params) {
		return
	}
	config, err := (workflow.Workflow{}).Get(params.RepositoryID)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	c.JsonResponseWithoutError(ctx, gin.H{"config": config})
}
func (c Workflow) Save(ctx *gin.Context) {
	var params workflowParams
	if !c.authorize(ctx, &params) {
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
	if err := (workflow.Workflow{}).Save(params.RepositoryID, params.Config); err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	c.JsonSuccessResponse(ctx)
}
func (c Workflow) Runs(ctx *gin.Context) {
	var params workflowParams
	if !c.authorize(ctx, &params) {
		return
	}
	runs, err := (workflow.Workflow{}).Runs(params.RepositoryID, params.TaskID)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	c.JsonResponseWithoutError(ctx, gin.H{"list": runs})
}
func (c Workflow) Logs(ctx *gin.Context) {
	var params workflowParams
	if !c.authorize(ctx, &params) {
		return
	}
	run, err := (workflow.Workflow{}).Status(params.RepositoryID, params.RunID)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	requestCtx, cancel := context.WithTimeout(ctx.Request.Context(), 30*time.Second)
	defer cancel()
	if params.TailLines == 0 {
		params.TailLines = 100
	}
	logs, err := (workflow.Workflow{}).Logs(requestCtx, run, params.TailLines)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	c.JsonResponseWithoutError(ctx, gin.H{"log": logs.Log, "archived": logs.Archived, "truncated": logs.Truncated, "status": run.ExecutionStatus})
}
