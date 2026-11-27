// Package document is an independent document subsystem. It imports no host application code.
package document

import (
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/w7panel/w7panel-zpk/modules/document/app/document/http/controller"
	"github.com/w7panel/w7panel-zpk/modules/document/app/document/http/middleware"
	"github.com/w7panel/w7panel-zpk/modules/document/common/entity"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	httpserver "github.com/we7coreteam/w7-rangine-go/v2/src/http/server"
)

type Provider struct{}

func (p *Provider) registerValidator() {
	engine := binding.Validator.Engine().(*validator.Validate)
	identifiePattern := regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,95}$`)
	rules := map[string]validator.Func{
		"document_identifie": func(field validator.FieldLevel) bool {
			return identifiePattern.MatchString(field.Field().String())
		},
		"document_ref": func(field validator.FieldLevel) bool {
			namespace, identifie, found := strings.Cut(field.Field().String(), "/")
			return found && identifiePattern.MatchString(namespace) && identifiePattern.MatchString(identifie)
		},
		"document_path": func(field validator.FieldLevel) bool {
			value := field.Field().String()
			return value != "" && len(value) <= 240 && !strings.ContainsAny(value, "\\\x00\r\n") && !path.IsAbs(value) && path.Clean(value) == value && value != "." && value != ".." && !strings.HasPrefix(value, "../")
		},
		"notblank": func(field validator.FieldLevel) bool {
			return strings.TrimSpace(field.Field().String()) != ""
		},
		"maxbytes": func(field validator.FieldLevel) bool {
			limit, err := strconv.Atoi(field.Param())
			return err == nil && len(field.Field().String()) <= limit
		},
	}
	for name, rule := range rules {
		if err := engine.RegisterValidation(name, rule); err != nil {
			panic(err)
		}
	}
}

func (p *Provider) initDb() {
	db, err := facade.GetDbFactory().Channel("default")
	if err != nil {
		panic(err)
	}
	if err := db.AutoMigrate(&entity.DocumentLibrary{}, &entity.DocumentFile{}); err != nil {
		panic(err)
	}
}

func (p *Provider) Register(httpServer *httpserver.Server) {
	p.registerValidator()
	p.initDb()
	httpServer.RegisterRouters(p.registerRoutes)
}

func (p *Provider) registerRoutes(engine *gin.Engine) {
	root := engine.Group("/doc")
	root.POST("/login", controller.Login{}.FromPanel)
	read := root.Group("", middleware.OptionalAuth{}.Process)
	read.POST("/file/list", controller.File{}.List)
	read.POST("/file/read", controller.File{}.Read)
	read.GET("/download", controller.Download{}.Export)
	auth := root.Group("", middleware.Auth{}.Process)
	auth.POST("/user", controller.Login{}.CurrentUser)
	auth.POST("/download-url", controller.Download{}.URL)
	auth.POST("/library/list", controller.Library{}.List)
	auth.POST("/library/info", controller.Library{}.Info)
	auth.POST("/library/create", controller.Library{}.Create)
	auth.POST("/library/edit", controller.Library{}.Edit)
	auth.POST("/library/delete", controller.Library{}.Delete)
	auth.POST("/file/save", controller.File{}.Save)
	auth.POST("/file/rename", controller.File{}.Rename)
	auth.POST("/file/delete", controller.File{}.Delete)
	auth.POST("/file/order", controller.File{}.Order)

	openapi := root.Group("/openapi", middleware.App{}.Process, middleware.AppUser{}.Process)
	openapi.POST("/file/save", controller.File{}.SaveFromApp)
	openapi.POST("/library/info", controller.Library{}.Info)
	openapi.POST("/library/create", controller.Library{}.Create)
	openapi.POST("/file/list", controller.File{}.List)
	openapi.POST("/file/read", controller.File{}.Read)
	openapi.POST("/file/delete", controller.File{}.Delete)
	openapi.POST("/file/order", controller.File{}.Order)
}
