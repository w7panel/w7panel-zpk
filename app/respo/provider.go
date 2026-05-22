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
			&entity.Order{},
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
		group.Match([]string{"POST", "OPTIONS"}, "/add", middleware.DenyDomainReq{}.Process, middleware.W7PanelUser{}.Process, controller.Formula{}.Add)
		group.Match([]string{"POST", "OPTIONS"}, "/delete", middleware.DenyDomainReq{}.Process, middleware.W7PanelUser{}.Process, controller.Formula{}.Delete)
		group.Match([]string{"POST", "OPTIONS"}, "/file", middleware.DenyDomainReq{}.Process, middleware.W7PanelUser{}.Process, controller.FormulaAttach{}.SaveFile)
		group.Match([]string{"POST", "OPTIONS"}, "/path-tree", middleware.DenyDomainReq{}.Process, middleware.W7PanelUser{}.Process, controller.FormulaAttach{}.Files)
		group.Match([]string{"POST", "OPTIONS"}, "/publish", middleware.DenyDomainReq{}.Process, middleware.W7PanelUser{}.Process, controller.Formula{}.Publish)
		group.Match([]string{"GET", "OPTIONS"}, "/list", middleware.W7PanelUser{CanSkip: true, NoAutoCreateUser: true}.Process, middleware.ConsoleUser{CanSkip: true}.Process, controller.Formula{}.List)
		group.Match([]string{"GET", "OPTIONS"}, "/detail/:id", controller.Formula{}.Detail)
		group.Match([]string{"GET", "OPTIONS"}, "/v2/detail/:id/:version", controller.Formula{}.Detail)
		group.Match([]string{"GET", "OPTIONS"}, "/info/:id", middleware.W7PanelUser{CanSkip: true, NoAutoCreateUser: true}.Process, middleware.ConsoleUser{CanSkip: true}.Process, controller.Formula{}.Info)
		group.Match([]string{"GET", "OPTIONS"}, "/info/:id/:cid", middleware.W7PanelUser{CanSkip: true, NoAutoCreateUser: true}.Process, middleware.ConsoleUser{CanSkip: true}.Process, controller.Formula{}.Info)
		group.Match([]string{"GET", "OPTIONS"}, "/v2/info/:id/:version", middleware.W7PanelUser{CanSkip: true}.Process, middleware.ConsoleUser{CanSkip: true}.Process, controller.Formula{}.Info)
		group.Match([]string{"GET", "OPTIONS"}, "/v2/info/:id/:version/:cid", middleware.W7PanelUser{CanSkip: true}.Process, middleware.ConsoleUser{CanSkip: true}.Process, controller.Formula{}.Info)
		group.Match([]string{"POST", "OPTIONS"}, "/icon", middleware.DenyDomainReq{}.Process, middleware.W7PanelUser{}.Process, controller.FormulaAttach{}.EditIcon)
		group.Match([]string{"POST", "OPTIONS"}, "/status", middleware.DenyDomainReq{}.Process, middleware.W7PanelUser{}.Process, controller.Formula{}.Status)

		group.Match([]string{"GET", "OPTIONS"}, "/user/info", middleware.W7PanelUser{CanSkip: true, NoAutoCreateUser: true}.Process, middleware.ConsoleUser{CanSkip: true}.Process, controller.User{}.Info)

		//设置商品价格属性
		group.Match([]string{"POST", "OPTIONS"}, "/goods/can-upgrade-versions", middleware.DenyDomainReq{}.Process, middleware.W7PanelUser{}.Process, controller.FormulaGoods{}.GetCanFeeUpgradeVersions)
		//发布商品
		group.Match([]string{"POST", "OPTIONS"}, "/goods/set-service-fee", middleware.DenyDomainReq{}.Process, middleware.W7PanelUser{}.Process, controller.FormulaGoods{}.SetServiceFee)
		group.Match([]string{"POST", "OPTIONS"}, "/goods/publish", middleware.DenyDomainReq{}.Process, middleware.W7PanelUser{}.Process, middleware.ConsoleUser{}.Process, controller.FormulaGoods{}.PublishGoods)
		group.Match([]string{"GET", "OPTIONS"}, "/goods/info/:id", controller.FormulaGoods{}.GoodsInfo)
		group.Match([]string{"POST", "OPTIONS"}, "/goods/labels", middleware.W7PanelUser{}.Process, middleware.ConsoleUser{}.Process, controller.FormulaGoods{}.GetGoodsLabels)
		group.Match([]string{"POST", "OPTIONS"}, "/attach/upload-img", middleware.DenyDomainReq{}.Process, middleware.W7PanelUser{}.Process, middleware.ConsoleUser{}.Process, controller.CloudAttach{}.UploadImg)

		//从云端同步应用
		group.Match([]string{"POST", "OPTIONS"}, "/cloud-app/notapp/list", middleware.DenyDomainReq{}.Process, middleware.W7PanelUser{}.Process, middleware.ConsoleUser{}.Process, controller.CloudApp{}.NotAppList)
		group.Match([]string{"POST", "OPTIONS"}, "/cloud-app/notapp/unpack", middleware.DenyDomainReq{}.Process, middleware.W7PanelUser{}.Process, middleware.ConsoleUser{}.Process, controller.CloudApp{}.UnPackNotApp)

		group.Match([]string{"POST", "OPTIONS"}, "/order/pay", middleware.W7PanelUser{}.Process, middleware.ConsoleUser{}.Process, controller.Order{}.Pay)
		group.Match([]string{"POST", "OPTIONS"}, "/order/info", middleware.W7PanelUser{}.Process, middleware.ConsoleUser{}.Process, controller.Order{}.Query)
		group.Match([]string{"POST", "OPTIONS"}, "/order/list", middleware.W7PanelUser{}.Process, middleware.ConsoleUser{}.Process, controller.Order{}.List)
		group.Match([]string{"POST", "OPTIONS"}, "/order/pay-notify", middleware.W7App{}.Process, controller.Order{}.PayNotify)
		group.Match([]string{"POST", "OPTIONS"}, "/order/refund-notify", middleware.W7App{}.Process, controller.Order{}.RefundNotify)

		//安装成功
		group.Match([]string{"POST", "OPTIONS"}, "/install/complete-notify", controller.Formula{}.InstallComplete)
		group.Match([]string{"POST", "OPTIONS"}, "/uninstall/complete-notify", controller.Formula{}.UnInstallComplete)

		// 版本相关
		group.Match([]string{"POST", "OPTIONS"}, "/version-list", middleware.W7PanelUser{}.Process, controller.Version{}.GetList)
		group.Match([]string{"POST", "OPTIONS"}, "/version-add", middleware.DenyDomainReq{}.Process, middleware.W7PanelUser{}.Process, controller.Version{}.Add)
		//group.Match([]string{"POST", "OPTIONS"}, "/version-publish", middleware.W7PanelUser{}.Process, controller.Version{}.Publish)
		// 制品压缩包管理
		group.Match([]string{"POST", "OPTIONS"}, "/get-zip-file-list", middleware.W7PanelUser{}.Process, controller.Zip{}.GetZipFileList)
		group.Match([]string{"POST", "OPTIONS"}, "/get-zip-file-content", middleware.W7PanelUser{}.Process, controller.Zip{}.GetZipFileContent)

		cors.Match([]string{"POST", "OPTIONS"}, "/zip/upload", middleware.DenyDomainReq{}.Process, middleware.W7PanelUser{}.Process, controller.Zip{}.Upload)
		cors.GET("/zip/download/:id", controller.Zip{}.Download)
		cors.GET("/zip/icon/:path", controller.FormulaAttach{}.GetIcon)
		cors.GET("/zpk/zip/icon/:path", controller.FormulaAttach{}.GetIcon)

		// 标签
		cors.Match([]string{"POST", "OPTIONS"}, "/respo/tag/add", middleware.DenyDomainReq{}.Process, middleware.W7PanelUser{}.Process, controller.Tag{}.Add)
		cors.Match([]string{"POST", "OPTIONS"}, "/respo/tag/delete", middleware.DenyDomainReq{}.Process, middleware.W7PanelUser{}.Process, controller.Tag{}.Delete)
		cors.Match([]string{"POST", "OPTIONS"}, "/respo/tag/list", controller.Tag{}.List)

		cors.Match([]string{"GET", "OPTIONS"}, "/static/*path", controller.Static{}.File)

		group.Match([]string{"POST", "OPTIONS"}, "/add-to-official", middleware.DenyDomainReq{}.Process, middleware.W7PanelUser{}.Process, middleware.AdminUser{}.Process, controller.FormulaRemote{}.PushToOfficialZpkStore)

		group.Match([]string{"POST", "OPTIONS"}, "/audit/list", middleware.DenyDomainReq{}.Process, middleware.W7PanelUser{}.Process, middleware.AdminUser{}.Process, controller.FormulaAudit{}.List)
		group.Match([]string{"POST", "OPTIONS"}, "/audit/audit", middleware.DenyDomainReq{}.Process, middleware.W7PanelUser{}.Process, middleware.AdminUser{}.Process, controller.FormulaAudit{}.Audit)

		cors.Match([]string{"POST", "OPTIONS"}, "/helm/chart/list", middleware.DenyDomainReq{}.Process, middleware.W7PanelUser{}.Process, controller.Helm{}.GetHelmRepositoryCharts)
		cors.Match([]string{"POST", "OPTIONS"}, "/helm/chart/version/list", middleware.DenyDomainReq{}.Process, middleware.W7PanelUser{}.Process, controller.Helm{}.GetHelmRepositoryChartVersions)

		openApiGroup := group.Group("/open-api", middleware.W7AppV2{}.Process)
		{
			openApiGroup.Match([]string{"POST", "OPTIONS"}, "/add-from-remote", middleware.W7AppUser{}.Process, controller.FormulaRemote{}.AddRemote)
			openApiGroup.Match([]string{"POST", "OPTIONS"}, "/audit/notify", controller.FormulaRemote{}.OfficialZpkStoreAuditNotify)

			openApiGroup.Match([]string{"POST", "OPTIONS"}, "/formula/info", controller.Formula{}.Info)
		}
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
