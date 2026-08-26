package respo

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/app/respo/command"
	"github.com/w7panel/w7panel-zpk/app/respo/http/controller"
	"github.com/w7panel/w7panel-zpk/app/respo/logic"
	logic2 "github.com/w7panel/w7panel-zpk/app/system/logic"
	"github.com/w7panel/w7panel-zpk/common/entity"
	"github.com/w7panel/w7panel-zpk/common/middleware"
	"github.com/w7panel/w7panel-zpk/common/types/registry"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/console"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	http_server "github.com/we7coreteam/w7-rangine-go/v2/src/http/server"
	"gorm.io/gorm"
)

type Provider struct {
}

func (provider *Provider) initDb() {
	runEnvType := facade.GetConfig().GetString("app.env")
	if runEnvType != "debug" && logic2.CurrentDBMode(facade.GetConfig()) != logic2.DBModeMySQL {
		db, err := facade.GetDbFactory().Channel("default")
		if err != nil {
			panic(err)
		}

		disableFK := db.Config.DisableForeignKeyConstraintWhenMigrating
		db.Config.DisableForeignKeyConstraintWhenMigrating = true
		// 同步数据库
		err = db.Migrator().AutoMigrate(
			&entity.Formula{},
			&entity.Tag{},
			&entity.TagFormula{},
			&entity.Version{},
			&entity.W7panelUser{},
		)
		db.Config.DisableForeignKeyConstraintWhenMigrating = disableFK
		if err != nil {
			panic(err)
		}
	}

	if err := provider.normalizeFormulaNames(); err != nil {
		panic(err)
	}
}

func (provider *Provider) normalizeFormulaNames() error {
	db, err := facade.GetDbFactory().Channel("default")
	if err != nil {
		return err
	}

	result := db.Model(&entity.Formula{}).
		Where("instr(name, ?) > 0", "_").
		Update("name", gorm.Expr("REPLACE(name, ?, ?)", "_", "-"))
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected > 0 {
		slog.Info("formula names normalized", "rows_affected", result.RowsAffected)
	}

	return nil
}

func (p Provider) registerEvent() {
	_ = facade.GetEvent().Subscribe(registry.RegistryRepositoryAfterPushedEvent, logic.Depot{}.OnRepositoryPushed)
}

func (provider *Provider) Register(httpServer *http_server.Server, console console.Console) {
	provider.initDb()

	provider.registerEvent()

	console.RegisterCommand(new(command.Pack))
	console.RegisterCommand(new(command.Sqlite))
	console.RegisterCommand(new(command.ResetTags))

	// 注册一些路由
	httpServer.RegisterRouters(func(engine *gin.Engine) {
		root := engine.Group("/zpk")
		root.GET("/", controller.Home{}.Index)

		root.GET("/live", func(ctx *gin.Context) {
			ctx.JSON(200, gin.H{
				"message": "ok",
			})
		})

		cors := root.Group("/", middleware.Cors{}.Process)

		group := cors.Group("/respo", middleware.Cors{}.Process)
		group.Match([]string{"POST", "OPTIONS"}, "/add", middleware.DenyDomainReq{}.Process, middleware.Auth{}.Process, controller.Formula{}.Add)
		group.Match([]string{"POST", "OPTIONS"}, "/delete", middleware.DenyDomainReq{}.Process, middleware.Auth{}.Process, controller.Formula{}.Delete)
		group.Match([]string{"POST", "OPTIONS"}, "/manifest/file", middleware.DenyDomainReq{}.Process, middleware.Auth{}.Process, controller.FormulaAttach{}.SaveManifestFile)
		group.Match([]string{"POST", "OPTIONS"}, "/manifest/path-tree", middleware.DenyDomainReq{}.Process, middleware.Auth{}.Process, controller.FormulaAttach{}.ManifestFiles)
		group.Match([]string{"POST", "OPTIONS"}, "/share-file/file", middleware.DenyDomainReq{}.Process, middleware.Auth{}.Process, controller.FormulaAttach{}.SaveSharedFile)
		group.Match([]string{"POST", "OPTIONS"}, "/share-file/path-tree", middleware.DenyDomainReq{}.Process, middleware.Auth{}.Process, controller.FormulaAttach{}.SharedFiles)
		group.Match([]string{"GET", "OPTIONS"}, "/list", middleware.Auth{CanSkip: true}.Process, middleware.ConsoleUser{CanSkip: true}.Process, controller.Formula{}.List)
		group.Match([]string{"GET", "OPTIONS"}, "/detail/:id", controller.Formula{}.Detail)
		group.Match([]string{"GET", "OPTIONS"}, "/v2/detail/:id/:version", controller.Formula{}.Detail)
		group.Match([]string{"GET", "OPTIONS"}, "/info/:id", middleware.Auth{CanSkip: true}.Process, middleware.CloudAccessToken{}.Process, middleware.W7PanelUser{}.Process, middleware.ConsoleUser{CanSkip: true}.Process, controller.Formula{}.Info)
		group.Match([]string{"GET", "OPTIONS"}, "/info/:id/:cid", middleware.Auth{CanSkip: true}.Process, middleware.CloudAccessToken{}.Process, middleware.W7PanelUser{}.Process, middleware.ConsoleUser{CanSkip: true}.Process, controller.Formula{}.Info)
		group.Match([]string{"GET", "OPTIONS"}, "/v2/info/:id/:version", middleware.Auth{CanSkip: true}.Process, middleware.CloudAccessToken{}.Process, middleware.W7PanelUser{}.Process, middleware.ConsoleUser{CanSkip: true}.Process, controller.Formula{}.Info)
		group.Match([]string{"GET", "OPTIONS"}, "/v2/info/:id/:version/:cid", middleware.Auth{CanSkip: true}.Process, middleware.CloudAccessToken{}.Process, middleware.W7PanelUser{}.Process, middleware.ConsoleUser{CanSkip: true}.Process, controller.Formula{}.Info)
		group.Match([]string{"POST", "OPTIONS"}, "/icon", middleware.DenyDomainReq{}.Process, middleware.Auth{}.Process, controller.FormulaAttach{}.EditIcon)
		group.Match([]string{"POST", "OPTIONS"}, "/status", middleware.DenyDomainReq{}.Process, middleware.Auth{}.Process, controller.Formula{}.Status)
		group.Match([]string{"POST", "OPTIONS"}, "/setting/get", middleware.DenyDomainReq{}.Process, middleware.Auth{}.Process, controller.FormulaSetting{}.Get)
		group.Match([]string{"POST", "OPTIONS"}, "/setting/set", middleware.DenyDomainReq{}.Process, middleware.Auth{}.Process, controller.FormulaSetting{}.Set)

		//设置商品价格属性
		group.Match([]string{"POST", "OPTIONS"}, "/goods/can-upgrade-versions", middleware.DenyDomainReq{}.Process, middleware.Auth{}.Process, controller.FormulaGoods{}.GetCanFeeUpgradeVersions)
		//发布商品
		group.Match([]string{"POST", "OPTIONS"}, "/goods/set-service-fee", middleware.DenyDomainReq{}.Process, middleware.Auth{}.Process, controller.FormulaGoods{}.SetServiceFee)
		group.Match([]string{"POST", "OPTIONS"}, "/goods/cross-upgrade-formulas", middleware.DenyDomainReq{}.Process, middleware.Auth{}.Process, controller.FormulaGoods{}.GetCrossUpgradeFormulaCandidates)
		group.Match([]string{"POST", "OPTIONS"}, "/goods/set-cross-upgrade-formulas", middleware.DenyDomainReq{}.Process, middleware.Auth{}.Process, controller.FormulaGoods{}.SetCrossUpgradeFormulas)
		group.Match([]string{"POST", "OPTIONS"}, "/goods/audit-status", middleware.DenyDomainReq{}.Process, middleware.Auth{}.Process, middleware.ConsoleUser{}.Process, controller.FormulaGoods{}.GetGoodsAuditStatus)
		group.Match([]string{"POST", "OPTIONS"}, "/goods/labels", middleware.Auth{}.Process, middleware.ConsoleUser{}.Process, controller.FormulaGoods{}.GetGoodsLabels)
		//从云端同步应用
		group.Match([]string{"POST", "OPTIONS"}, "/cloud-app/notapp/list", middleware.DenyDomainReq{}.Process, middleware.Auth{}.Process, middleware.ConsoleUser{}.Process, controller.CloudApp{}.NotAppList)
		group.Match([]string{"POST", "OPTIONS"}, "/cloud-app/notapp/unpack", middleware.DenyDomainReq{}.Process, middleware.Auth{}.Process, middleware.ConsoleUser{}.Process, controller.CloudApp{}.UnPackNotApp)

		//安装成功
		group.Match([]string{"POST", "OPTIONS"}, "/install/complete-notify", controller.Formula{}.InstallComplete)
		group.Match([]string{"POST", "OPTIONS"}, "/uninstall/complete-notify", controller.Formula{}.UnInstallComplete)

		// 版本相关
		group.Match([]string{"POST", "OPTIONS"}, "/publish", middleware.DenyDomainReq{}.Process, middleware.Auth{}.Process, controller.Version{}.Publish)
		group.Match([]string{"POST", "OPTIONS"}, "/version-list", middleware.Auth{}.Process, controller.Version{}.GetList)
		group.Match([]string{"POST", "OPTIONS"}, "/version-add", middleware.DenyDomainReq{}.Process, middleware.Auth{}.Process, controller.Version{}.Add)
		group.Match([]string{"POST", "OPTIONS"}, "/version-unpublish", middleware.DenyDomainReq{}.Process, middleware.Auth{}.Process, controller.Version{}.Unpublish)
		// 制品压缩包管理
		group.Match([]string{"POST", "OPTIONS"}, "/get-zip-file-list", middleware.Auth{}.Process, controller.Attach{}.GetBackendZipFileList)
		group.Match([]string{"POST", "OPTIONS"}, "/get-zip-file-content", middleware.Auth{}.Process, controller.Attach{}.GetBackendZipFileContent)
		group.Match([]string{"GET", "OPTIONS"}, "/attach/frontend/:identifie/:version/*path", controller.Attach{}.GetFrontendZipFileContent)

		cors.Match([]string{"POST", "OPTIONS"}, "/zip/upload", middleware.DenyDomainReq{}.Process, middleware.Auth{}.Process, controller.Attach{}.Upload)
		cors.GET("/zip/download/:id", controller.Attach{}.Download)
		cors.GET("/zip/icon/:path", controller.FormulaAttach{}.GetIcon)
		cors.GET("/zpk/zip/icon/:path", controller.FormulaAttach{}.GetIcon)

		// 标签
		cors.Match([]string{"POST", "OPTIONS"}, "/respo/tag/add", middleware.DenyDomainReq{}.Process, middleware.Auth{}.Process, controller.Tag{}.Add)
		cors.Match([]string{"POST", "OPTIONS"}, "/respo/tag/delete", middleware.DenyDomainReq{}.Process, middleware.Auth{}.Process, controller.Tag{}.Delete)
		cors.Match([]string{"POST", "OPTIONS"}, "/respo/tag/list", controller.Tag{}.List)

		cors.Match([]string{"GET", "OPTIONS"}, "/static/*path", controller.Static{}.File)

		cors.Match([]string{"POST", "OPTIONS"}, "/helm/chart/list", middleware.DenyDomainReq{}.Process, middleware.Auth{}.Process, controller.Helm{}.GetHelmRepositoryCharts)
		cors.Match([]string{"POST", "OPTIONS"}, "/helm/chart/version/list", middleware.DenyDomainReq{}.Process, middleware.Auth{}.Process, controller.Helm{}.GetHelmRepositoryChartVersions)

		group.Match([]string{"POST", "OPTIONS"}, "/open-api/formula/base-info", controller.Formula{}.BaseInfo)
	})

	// 初始化本地仓库
	err := logic.RegisterDepot()
	if err != nil {
		panic(err)
	}

	err = logic.Tag{}.ResetTags()
	if err != nil {
		slog.Error("tag reset fail", "err", err)
	}

	depot, _ := logic.NewDepot()
	err = depot.InitDepotEnv()
	if err != nil {
		panic(err)
	}
	go depot.PackLoop()

	logic.InitFormulaPublishLoop()
	logic.RecoverFormulaPublishTasks()
}
