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

func (p Provider) Register(server *httpserver.Server) {
	if facade.GetConfig().GetString("app.env") != "debug" && !systemlogic.ShouldBlockWrites(facade.GetConfig()) {
		db, err := facade.GetDbFactory().Channel("default")
		if err != nil {
			panic(err)
		}
		if err = p.initDatabase(db); err != nil {
			panic(err)
		}
		if err = (logic.Workflow{}).MigrateWorkflowRules(); err != nil {
			panic(err)
		}
	}
	server.RegisterRouters(func(engine *gin.Engine) {
		api := engine.Group("/zpk/v2/api/workflow", middleware.Cors{}.Process, middleware.DenyDomainReq{}.Process, middleware.Auth{}.Process)
		workflow := controller.Workflow{}
		api.POST("/run", workflow.Run)
		api.POST("/get", workflow.Get)
		api.POST("/save", workflow.Save)
		api.POST("/runs", workflow.Runs)
		api.POST("/logs", workflow.Logs)
		cluster := engine.Group("/zpk/v2/api/repository/deploy_rule/k8s", middleware.Cors{}.Process, middleware.DenyDomainReq{}.Process, middleware.Auth{}.Process)
		deploy := controller.Deploy{}
		cluster.Any("/proxy/*path", middleware.AdminUser{}.Process, deploy.K8sProxy)
		cluster.POST("/namespaces", deploy.GetK8sNamespace)
		cluster.POST("/apps", deploy.GetK8sApps)
		cluster.POST("/app-containers", deploy.GetK8sAppContainers)
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
		{&entity.RegistryRepositoryDeployRule{}, []string{"RegistryUsername", "RegistryPassword", "UserID", "RepositoryName", "ImageRepository", "ImageHost", "PlainHTTP", "WorkflowTaskKey", "ExecutionOrder", "TaskAction", "ImageSource", "SourceBuildTaskKey", "DockerfileInstructions", "OutputTagSuffix", "TaskName"}},
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
	indexes := []string{"workflow_run_task", "RepositoryID", "TriggerRepository", "ExecutionStatus", "BuildOutputTag"}
	for _, index := range indexes {
		if !db.Migrator().HasIndex(log, index) {
			if err := db.Migrator().CreateIndex(log, index); err != nil {
				return err
			}
		}
	}
	return nil
}
