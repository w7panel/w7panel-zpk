// Package document is an independent document subsystem. It imports no host application code.
package document

import (
	"context"
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/modules/document/app/document/http/controller"
	"github.com/w7panel/w7panel-zpk/modules/document/app/document/logic"
	"github.com/w7panel/w7panel-zpk/modules/document/common/dao"
	"github.com/w7panel/w7panel-zpk/modules/document/common/entity"
	"github.com/w7panel/w7panel-zpk/modules/document/common/middleware"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/permission"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/storage"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	httpserver "github.com/we7coreteam/w7-rangine-go/v2/src/http/server"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Options struct {
	DB          *gorm.DB
	Storage     storage.Store
	Permissions permission.Service
	// Migrate controls schema and default category initialization.
	Migrate bool
}

type Provider struct {
	Logic *logic.Document
	sdk   *service.SDK
}

func New(options Options) (*Provider, error) {
	if options.DB == nil || options.Storage == nil || options.Permissions == nil {
		return nil, errors.New("文档系统配置不完整")
	}
	query := dao.Use(options.DB)
	if options.Migrate {
		if err := options.DB.AutoMigrate(&entity.DocumentCategory{}, &entity.DocumentLibrary{}, &entity.DocumentFile{}, &entity.DocumentDownloadLink{}); err != nil {
			return nil, err
		}
		if err := query.DocumentCategory.WithContext(context.Background()).Clauses(clause.OnConflict{DoNothing: true}).Create(&entity.DocumentCategory{Identifie: "applications", Name: "制品文档", UserID: 0, Revision: 1}); err != nil {
			return nil, err
		}
	}
	return &Provider{Logic: logic.New(query, options.Permissions, options.Storage)}, nil
}

func (p *Provider) initDependencies() error {
	config := facade.GetConfig()
	sdk, err := service.InitSDK(config)
	if err != nil {
		return err
	}
	provider, err := New(Options{
		DB:          sdk.DB,
		Storage:     sdk.Storage,
		Permissions: sdk.Permissions,
		Migrate:     !config.GetBool("document.read_only"),
	})
	if err != nil {
		sdk.Close()
		return err
	}
	*p = *provider
	p.sdk = sdk
	return nil
}

func (p *Provider) Register(httpServer *httpserver.Server) {
	if p.Logic == nil {
		if err := p.initDependencies(); err != nil {
			panic(err)
		}
	}
	httpServer.RegisterRouters(p.registerRoutes)
}

// Close releases storage owned by standalone initialization. Injected storage belongs to its caller.
func (p *Provider) Close() error {
	if p.sdk == nil {
		return nil
	}
	err := p.sdk.Close()
	p.sdk = nil
	return err
}

func (p *Provider) registerRoutes(engine *gin.Engine) {
	root := engine.Group("/zpk/document")
	abstract := controller.Abstract{Logic: p.Logic}
	library := controller.Library{Abstract: abstract}
	file := controller.File{Abstract: abstract}
	download := controller.Download{Abstract: abstract}
	permissions := controller.Permission{Abstract: abstract}
	category := controller.Category{Logic: &logic.Category{Q: p.Logic.Q, Permissions: p.Logic.Permissions}}
	login := controller.Login{Logic: &logic.Login{Permissions: p.Logic.Permissions}}
	root.GET("/download", download.Public)
	root.POST("/login", login.FromPanel)
	auth := root.Group("", middleware.Auth{Permissions: p.Logic.Permissions}.Process)
	auth.POST("/user", login.CurrentUser)
	categoryRoutes := auth.Group("/category")
	categoryRoutes.POST("/permission/set", category.SetPermission)
	categoryRoutes.POST("/list", category.List)
	categoryRoutes.POST("/create", category.Create)
	categoryRoutes.POST("/edit", category.Edit)
	categoryRoutes.POST("/delete", category.Delete)
	auth.POST("/library/list", library.List)
	auth.POST("/library/create", library.Create)
	auth.POST("/library/initialize", library.Initialize)
	auth.POST("/library/edit", library.Edit)
	auth.POST("/library/delete", library.Delete)
	auth.POST("/file/list", file.List)
	auth.POST("/file/read", file.Read)
	auth.POST("/file/save", file.Save)
	auth.POST("/file/rename", file.Rename)
	auth.POST("/file/delete", file.Delete)
	auth.POST("/file/order", file.Order)
	auth.POST("/permission/set", permissions.Set)
	auth.POST("/link/create", download.CreateLink)
	auth.POST("/link/list", download.Links)
	auth.POST("/link/revoke", download.RevokeLink)
	auth.POST("/download", download.Export)
}
