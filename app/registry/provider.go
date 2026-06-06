package registry

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/app/registry/http/controller"
	"github.com/w7panel/w7panel-zpk/app/registry/logic"
	respologic "github.com/w7panel/w7panel-zpk/app/system/logic"
	"github.com/w7panel/w7panel-zpk/common/accessor"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	"github.com/w7panel/w7panel-zpk/common/function"
	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/w7panel/w7panel-zpk/common/middleware"
	"github.com/w7panel/w7panel-zpk/common/types/registry"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	httpserver "github.com/we7coreteam/w7-rangine-go/v2/src/http/server"
)

type Provider struct {
}

func (p Provider) registerRoutes(httpServer *httpserver.Server) {
	httpServer.RegisterRouters(func(engine *gin.Engine) {
		root := engine.Group("/zpk/v2")

		root.Any("/registry/auth", controller.Auth{}.Auth)
		root.Any("/registry/webhook", controller.WebHook{}.Hook)

		api := root.Group("/api", middleware.Cors{}.Process, middleware.DenyDomainReq{}.Process, middleware.W7PanelUser{}.Process)
		api.POST("/permission/namespace/set", middleware.AdminUser{}.Process, controller.Permission{}.SetNamespacePermission)
		api.POST("/permission/namespace/get", middleware.AdminUser{}.Process, controller.Permission{}.GetNamespacePermission)

		api.POST("/namespace/add", controller.Namespace{}.Create)
		api.POST("/namespace/info", controller.Namespace{}.Info)
		api.POST("/namespace/list", controller.Namespace{}.List)
		api.POST("/namespace/sub_namespace/list", controller.Namespace{}.SubNamespaceList)
		api.POST("/namespace/edit", controller.Namespace{}.Edit)
		api.POST("/namespace/del", controller.Namespace{}.Delete)

		api.POST("/repository/add", controller.Repository{}.Create)
		api.POST("/repository/info", controller.Repository{}.Info)
		api.POST("/repository/list", controller.Repository{}.List)
		api.POST("/repository/edit", controller.Repository{}.Edit)
		api.POST("/repository/del", controller.Repository{}.Delete)
		api.POST("/repository/tags/list", controller.Repository{}.Tags)
		api.POST("/repository/tags/del", controller.Repository{}.DelTag)

		api.Any("/repository/deploy_rule/k8s/proxy/*path", controller.Deploy{}.K8sProxy)
		api.POST("/repository/deploy_rule/k8s/namespaces", controller.Deploy{}.GetK8sNamespace)
		api.POST("/repository/deploy_rule/k8s/apps", controller.Deploy{}.GetK8sApps)
		api.POST("/repository/deploy_rule/k8s/app-containers", controller.Deploy{}.GetK8sAppContainers)
		api.POST("/repository/deploy_rule/add", controller.Deploy{}.AddRule)
		api.POST("/repository/deploy_rule/info", controller.Deploy{}.QueryRule)
		api.POST("/repository/deploy_rule/list", controller.Deploy{}.ListRule)
		api.POST("/repository/deploy_rule/edit", controller.Deploy{}.EditRule)
		api.POST("/repository/deploy_rule/del", controller.Deploy{}.DelRule)
		api.POST("/repository/deploy_rule/deploy-log", controller.Deploy{}.RuleDeployLog)

		api.GET("/registry/info", controller.Info{}.RegistryInfo)
	})
}

func (p Provider) registerEvent() {
	_ = facade.GetEvent().Subscribe(registry.RegistryRepositoryPrepareEvent, logic.Repository{}.OnRepositoryPrepareOperate)
	_ = facade.GetEvent().Subscribe(registry.RegistryRepositoryPushedEvent, logic.Repository{}.OnRepositoryPushed)
	_ = facade.GetEvent().Subscribe(registry.RegistryRepositoryPulledEvent, logic.Repository{}.OnRepositoryPulled)
	_ = facade.GetEvent().Subscribe(registry.RegistryRepositoryAfterPushedEvent, logic.Deploy{}.OnRepositoryPushed)
	_ = facade.GetEvent().Subscribe(registry.ClearUserPermissionEvent, logic.Permission{}.OnClearUserPermissionEvent)
	_ = facade.GetEvent().Subscribe(registry.AddUserPermissionEvent, logic.Permission{}.OnAddUserPermissionEvent)
	_ = facade.GetEvent().Subscribe(registry.DeleteUserPermissionEvent, logic.Permission{}.OnDelUserPermissionEvent)
}

func (p Provider) initJwtCert() {
	certFileContent, err := os.ReadFile(facade.GetConfig().GetString("registry_cli.default.oauth.rsa_cert_path"))
	if err != nil {
		panic(err)
	}
	privateFileContent, err := os.ReadFile(facade.GetConfig().GetString("registry_cli.default.oauth.rsa_private_path"))
	if err != nil {
		panic(err)
	}
	publicKey, privateKey, _, err := function.LoadCertAndKey(certFileContent, privateFileContent)
	if err != nil {
		panic(err)
	}
	logic.PermissionPrivateKey = privateKey
	logic.PermissionPublicKey = publicKey
}

func (p Provider) initSuperAdmin() {
	first, _ := dao.Q.RegistryUser.Where(dao.Q.RegistryUser.Username.Eq(facade.GetConfig().GetString("registry_cli.default.username"))).First()
	if first == nil {
		user := entity.RegistryUser{
			Username:   facade.GetConfig().GetString("registry_cli.default.username"),
			Desc:       "super admin",
			Password:   logic2.User{}.MakeUserPassword(facade.GetConfig().GetString("registry_cli.default.password")),
			Type:       logic2.UserTypeSuperAdminForRegistry,
			ExpireDays: 0,
		}
		err := dao.Q.RegistryUser.Create(&user)
		if err != nil {
			panic(err)
		}
	}
}

func (p Provider) initDefaultNamespace() {
	defaultNamespace, _ := dao.Q.RegistryNamespace.Where(dao.Q.RegistryNamespace.Name.Eq(logic2.DefaultNamespace)).First()
	if defaultNamespace == nil {
		namespace := entity.RegistryNamespace{
			Name:        logic2.DefaultNamespace,
			VisibleType: int32(logic.VisibleTypePrivate),
		}
		err := dao.Q.RegistryNamespace.Create(&namespace)
		if err != nil {
			panic(err)
		}
	}
}

func (p Provider) initDb() {
	runEnvType := facade.GetConfig().GetString("app.env")
	if runEnvType != "debug" && respologic.CurrentDBMode(facade.GetConfig()) != respologic.DBModeMySQL {
		db, err := facade.GetDbFactory().Channel("default")
		if err != nil {
			panic(err)
		}

		disableFK := db.Config.DisableForeignKeyConstraintWhenMigrating
		db.Config.DisableForeignKeyConstraintWhenMigrating = true
		// 同步数据库
		err = db.Migrator().AutoMigrate(
			&entity.RegistryUser{},
			&entity.RegistryRepository{},
			&entity.RegistryNamespace{},
			&entity.RegistryUserPermission{},
			&entity.RegistryRepositoryDeployRule{},
			&entity.RegistryRepositoryTag{},
			&entity.RegistryRepositoryDeployRuleMatchLog{},
		)
		db.Config.DisableForeignKeyConstraintWhenMigrating = disableFK
		if err != nil {
			panic(err)
		}
	}

	initLockFile := filepath.Join(facade.GetConfig().GetString("setting.db_dir"), ".initdb.lock")
	if _, err := os.Stat(initLockFile); os.IsNotExist(err) {
		if err := p.syncRegistryUserPermissions(); err != nil {
			panic(err)
		}
		if err := os.WriteFile(initLockFile, []byte("done"), 0o644); err != nil {
			panic(err)
		}
	} else if err != nil {
		panic(err)
	}
}

func (provider *Provider) syncRegistryUserPermissions() error {
	ociNamespace := facade.GetConfig().GetString("setting.depot.oci_namespace")
	if ociNamespace == "" {
		return nil
	}

	actions := accessor.PermissionActionOption{
		"push",
		"pull",
	}

	users, err := dao.Q.RegistryUser.Find()
	if err != nil {
		return err
	}
	userIDs := make([]int32, 0, len(users))
	for _, user := range users {
		userIDs = append(userIDs, user.ID)
	}
	if len(userIDs) > 0 {
		newPermissions := make([]*entity.RegistryUserPermission, 0)
		for _, user := range users {
			newPermissions = append(newPermissions, &entity.RegistryUserPermission{
				UserID:        user.ID,
				ResourceType:  "namespace",
				ResourceValue: ociNamespace,
				Action:        &actions,
			})
		}
		if len(newPermissions) > 0 {
			if err := dao.Q.RegistryUserPermission.CreateInBatches(newPermissions, 100); err != nil {
				return err
			}
		}
	}

	formulas, err := dao.Q.Formula.Find()
	if err != nil {
		return err
	}
	repositoryPermissions := make([]*entity.RegistryUserPermission, 0)
	for _, formula := range formulas {
		if formula.UserID <= 0 || strings.TrimSpace(formula.Name) == "" {
			continue
		}
		repositoryPermissions = append(repositoryPermissions, &entity.RegistryUserPermission{
			UserID:        formula.UserID,
			ResourceType:  "repository",
			ResourceValue: ociNamespace + "/" + strings.ReplaceAll(formula.Name, "-", "_"),
			Action:        &actions,
		})
	}
	if len(repositoryPermissions) == 0 {
		return nil
	}
	return dao.Q.RegistryUserPermission.CreateInBatches(repositoryPermissions, 100)
}

func (p Provider) Register(httpServer *httpserver.Server) {
	p.initDb()

	p.registerRoutes(httpServer)
	p.registerEvent()

	p.initJwtCert()

	p.initDefaultNamespace()
	p.initSuperAdmin()
}
