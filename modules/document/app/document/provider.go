// Package document is an independent document subsystem. It imports no host application code.
package document

import (
	"context"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/modules/document/app/document/http/controller"
	"github.com/w7panel/w7panel-zpk/modules/document/app/document/logic"
	"github.com/w7panel/w7panel-zpk/modules/document/common/dao"
	"github.com/w7panel/w7panel-zpk/modules/document/common/entity"
	"github.com/w7panel/w7panel-zpk/modules/document/common/middleware"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	httpserver "github.com/we7coreteam/w7-rangine-go/v2/src/http/server"
	"gorm.io/gorm/clause"
)

type Provider struct{}

func (p *Provider) initDb() {
	if facade.GetConfig().GetBool("document.read_only") {
		return
	}
	db, err := facade.GetDbFactory().Channel("default")
	if err != nil {
		panic(err)
	}
	if err := db.AutoMigrate(&entity.DocumentCategory{}, &entity.DocumentLibrary{}, &entity.DocumentFile{}); err != nil {
		panic(err)
	}
	if err := dao.Q.DocumentCategory.WithContext(context.Background()).Clauses(clause.OnConflict{DoNothing: true}).Create(&entity.DocumentCategory{Identifie: "applications", Name: "制品文档", Revision: 1}); err != nil {
		panic(err)
	}
}

func (p *Provider) Register(httpServer *httpserver.Server) {
	p.initDb()
	httpServer.RegisterRouters(p.registerRoutes)
}

func (p *Provider) registerRoutes(engine *gin.Engine) {
	root := engine.Group("/zpk/document", sessions.Sessions(logic.SessionName, service.SessionStore))
	root.POST("/login", controller.Login{}.FromPanel)
	read := root.Group("", middleware.Auth{Optional: true}.Process)
	read.POST("/file/list", controller.File{}.List)
	read.POST("/file/read", controller.File{}.Read)
	read.GET("/download", controller.Download{}.Export)
	auth := root.Group("", middleware.Auth{}.Process)
	auth.POST("/user", controller.Login{}.CurrentUser)
	categoryRoutes := auth.Group("/category")
	categoryRoutes.POST("/list", controller.Category{}.List)
	categoryRoutes.POST("/create", controller.Category{}.Create)
	categoryRoutes.POST("/edit", controller.Category{}.Edit)
	categoryRoutes.POST("/delete", controller.Category{}.Delete)
	auth.POST("/library/list", controller.Library{}.List)
	auth.POST("/library/create", controller.Library{}.Create)
	auth.POST("/library/initialize", controller.Library{}.Initialize)
	auth.POST("/library/edit", controller.Library{}.Edit)
	auth.POST("/library/delete", controller.Library{}.Delete)
	auth.POST("/file/save", controller.File{}.Save)
	auth.POST("/file/rename", controller.File{}.Rename)
	auth.POST("/file/delete", controller.File{}.Delete)
	auth.POST("/file/order", controller.File{}.Order)
}
