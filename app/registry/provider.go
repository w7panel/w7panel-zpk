package registry

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/app/registry/command"
	commoncontroller "github.com/w7panel/w7panel-zpk/app/registry/http/controller/common"
	"github.com/w7panel/w7panel-zpk/app/registry/http/controller/openapi"
	controller "github.com/w7panel/w7panel-zpk/app/registry/http/controller/repository"

	registrycommon "github.com/w7panel/w7panel-zpk/app/registry/logic/common"
	registryrepository "github.com/w7panel/w7panel-zpk/app/registry/logic/repository"
	respologic "github.com/w7panel/w7panel-zpk/app/system/logic"
	"github.com/w7panel/w7panel-zpk/common/accessor"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	"github.com/w7panel/w7panel-zpk/common/function"
	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/w7panel/w7panel-zpk/common/middleware"
	"github.com/w7panel/w7panel-zpk/common/types/registry"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/console"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	httpserver "github.com/we7coreteam/w7-rangine-go/v2/src/http/server"
)

type Provider struct {
}

func (p Provider) registerRoutes(httpServer *httpserver.Server) {
	httpServer.RegisterRouters(func(engine *gin.Engine) {
		root := engine.Group("/zpk/v2")

		root.Any("/registry/auth", commoncontroller.Auth{}.Auth)
		root.Any("/registry/webhook", controller.WebHook{}.Hook)

		permissionService := root.Group("/openapi/permission", middleware.App{}.Process)
		permissionService.POST("/add", openapi.ResourcePermission{}.Add)
		permissionService.POST("/delete", openapi.ResourcePermission{}.Delete)
		permissionService.POST("/clear", openapi.ResourcePermission{}.Clear)

		resourceService := root.Group("/openapi/resource", middleware.App{}.Process)
		resourceService.POST("/get", openapi.Resource{}.Get)
		resourceService.POST("/set", openapi.Resource{}.Set)
		resourceService.POST("/check", openapi.Resource{}.Check)
		resourceService.POST("/delete", openapi.Resource{}.Delete)

		namespaceService := root.Group("/openapi/namespace", middleware.App{}.Process)
		namespaceService.POST("/get", openapi.NamespaceService{}.Get)
		namespaceService.POST("/create", openapi.NamespaceService{}.Create)
		namespaceService.POST("/rename", openapi.NamespaceService{}.Rename)
		namespaceService.POST("/delete", openapi.NamespaceService{}.Delete)

		namespaces := root.Group("/api/namespace", middleware.Cors{}.Process, middleware.DenyDomainReq{}.Process, middleware.Auth{}.Process)
		namespaces.POST("/add", commoncontroller.Namespace{}.Create)
		namespaces.POST("/info", commoncontroller.Namespace{}.Info)
		namespaces.POST("/list", commoncontroller.Namespace{}.List)
		namespaces.POST("/sub_namespace/list", commoncontroller.Namespace{}.SubNamespaceList)
		namespaces.POST("/edit", commoncontroller.Namespace{}.Edit)
		namespaces.POST("/del", commoncontroller.Namespace{}.Delete)

		api := root.Group("/api", middleware.Cors{}.Process, middleware.DenyDomainReq{}.Process, middleware.Auth{}.Process)
		api.POST("/permission/namespace/set", middleware.AdminUser{}.Process, commoncontroller.NamespacePermission{}.SetNamespacePermission)
		api.POST("/permission/namespace/get", middleware.AdminUser{}.Process, commoncontroller.NamespacePermission{}.GetNamespacePermission)

		api.POST("/resource/create", commoncontroller.Resource{}.Create)
		api.POST("/resource/info", commoncontroller.Resource{}.Info)
		api.POST("/resource/list", commoncontroller.Resource{}.List)
		api.POST("/repository/edit", controller.Repository{}.Edit)
		api.POST("/repository/del", controller.Repository{}.Delete)
		api.POST("/repository/tags/list", controller.Repository{}.Tags)
		api.POST("/repository/tags/del", controller.Repository{}.DelTag)
		api.POST("/repository/tags/batch-del", controller.Repository{}.BatchDelTags)

		api.GET("/registry/info", commoncontroller.Info{}.RegistryInfo)
	})
}

func (p Provider) registerEvent() {
	_ = facade.GetEvent().Subscribe(registry.RegistryRepositoryPrepareEvent, registryrepository.Repository{}.OnRepositoryPrepareOperate)
	_ = facade.GetEvent().Subscribe(registry.RegistryRepositoryPushedEvent, registryrepository.Repository{}.OnRepositoryPushed)
	_ = facade.GetEvent().Subscribe(registry.RegistryRepositoryPulledEvent, registryrepository.Repository{}.OnRepositoryPulled)
	_ = facade.GetEvent().Subscribe(registry.RegistryRepositoryAfterPushedEvent, registryrepository.Deploy{}.OnRepositoryPushed)
	_ = facade.GetEvent().Subscribe(registry.UpdateRepositoryEvent, registryrepository.Repository{}.OnUpdateRepositoryEvent)
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
	registrycommon.PermissionPrivateKey = privateKey
	registrycommon.PermissionPublicKey = publicKey
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
			VisibleType: int32(registrycommon.VisibleTypePrivate),
		}
		err := dao.Q.RegistryNamespace.Create(&namespace)
		if err != nil {
			panic(err)
		}
	}
}

func (p Provider) initDb() {
	runEnvType := facade.GetConfig().GetString("app.env")
	if runEnvType != "debug" && !respologic.ShouldBlockWrites(facade.GetConfig()) {
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
			&entity.RegistryRepositoryTag{},
		)
		db.Config.DisableForeignKeyConstraintWhenMigrating = disableFK
		if err != nil {
			panic(err)
		}
		if err := (registrycommon.Resource{}).MigrateLegacy(db); err != nil {
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

func (p Provider) Register(httpServer *httpserver.Server, console console.Console) {
	console.RegisterCommand(new(command.Migrate))
	console.RegisterCommand(new(command.ClearImages))

	p.initDb()

	p.registerRoutes(httpServer)
	p.registerEvent()

	p.initJwtCert()

	p.initDefaultNamespace()
	p.initSuperAdmin()
}
