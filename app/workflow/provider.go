package workflow

import (
	"github.com/gin-gonic/gin"
	systemlogic "github.com/w7panel/w7panel-zpk/app/system/logic"
	"github.com/w7panel/w7panel-zpk/app/workflow/http/controller"
	"github.com/w7panel/w7panel-zpk/app/workflow/logic"
	"github.com/w7panel/w7panel-zpk/common/entity"
	"github.com/w7panel/w7panel-zpk/common/middleware"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	httpserver "github.com/we7coreteam/w7-rangine-go/v2/src/http/server"
	"gorm.io/gorm"
)

type Provider struct{}

// Register accepts caller-owned access checks.
func (p Provider) Register(server *httpserver.Server, access func(*gin.Context, int32) error) {
	if facade.GetConfig().GetString("app.env") != "debug" && !systemlogic.ShouldBlockWrites(facade.GetConfig()) {
		db, err := facade.GetDbFactory().Channel("default")
		if err != nil {
			panic(err)
		}
		if err = p.initDatabase(db); err != nil {
			panic(err)
		}
	}
	server.RegisterRouters(func(engine *gin.Engine) {
		api := engine.Group("/zpk/v2/api/repository", middleware.Cors{}.Process, middleware.DenyDomainReq{}.Process, middleware.Auth{}.Process)
		workflow := controller.Workflow{Access: access}
		deploy := controller.Deploy{Access: access}
		api.POST("/workflow/get", workflow.Get)
		api.POST("/workflow/save", workflow.Save)
		api.POST("/workflow/runs", workflow.Runs)
		api.POST("/workflow/logs", workflow.Logs)
		api.Any("/deploy_rule/k8s/proxy/*path", middleware.AdminUser{}.Process, deploy.K8sProxy)
		api.POST("/deploy_rule/k8s/namespaces", deploy.GetK8sNamespace)
		api.POST("/deploy_rule/k8s/apps", deploy.GetK8sApps)
		api.POST("/deploy_rule/k8s/app-containers", deploy.GetK8sAppContainers)
		api.POST("/deploy_rule/add", deploy.AddRule)
		api.POST("/deploy_rule/info", deploy.QueryRule)
		api.POST("/deploy_rule/list", deploy.ListRule)
		api.POST("/deploy_rule/edit", deploy.EditRule)
		api.POST("/deploy_rule/del", deploy.DelRule)
		api.POST("/deploy_rule/deploy-log", deploy.RuleDeployLog)
	})
	logic.InitExecutor()
}

// initDatabase adds workflow columns to the existing tables. It deliberately
// leaves historical columns intact. Registry metadata is migrated by its own provider.
func (Provider) initDatabase(db *gorm.DB) error {
	tables := []struct {
		model  interface{}
		fields []string
	}{
		{&entity.RegistryRepositoryDeployRule{}, []string{"WorkflowTaskKey", "ExecutionOrder", "TaskAction", "ImageSource", "SourceBuildTaskKey", "DockerfileInstructions", "OutputTagSuffix", "TaskName"}},
		{&entity.RegistryRepositoryDeployRuleMatchLog{}, []string{"WorkflowRunID", "WorkflowTaskKey", "RepositoryID", "TriggerRepository", "TriggerTag", "TriggerDigest", "ExecutionStatus", "FailureReason", "EncryptedConfigSnapshot", "UpdatedAt", "BuildOutputTag", "BuildOutputDigest"}},
	}
	for _, table := range tables {
		if !db.Migrator().HasTable(table.model) {
			if err := db.AutoMigrate(table.model); err != nil {
				return err
			}
			continue
		}
		for _, field := range table.fields {
			if !db.Migrator().HasColumn(table.model, field) {
				if err := db.Migrator().AddColumn(table.model, field); err != nil {
					return err
				}
			}
		}
	}
	log := &entity.RegistryRepositoryDeployRuleMatchLog{}
	indexes := []string{"workflow_run_task", "RepositoryID", "ExecutionStatus", "BuildOutputTag"}
	for _, index := range indexes {
		if !db.Migrator().HasIndex(log, index) {
			if err := db.Migrator().CreateIndex(log, index); err != nil {
				return err
			}
		}
	}
	return nil
}
