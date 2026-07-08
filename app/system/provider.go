package system

import (
	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/app/system/http/controller"
	"github.com/w7panel/w7panel-zpk/common/middleware"
	http_server "github.com/we7coreteam/w7-rangine-go/v2/src/http/server"
)

type Provider struct {
}

func (provider *Provider) Register(httpServer *http_server.Server) {
	httpServer.RegisterRouters(func(engine *gin.Engine) {
		root := engine.Group("/zpk")

		cors := root.Group("/", middleware.Cors{}.Process, middleware.Auth{}.Process)
		oidc := cors.Group("/system/oidc", middleware.Cors{}.Process, middleware.DenyDomainReq{}.Process)
		oidc.Match([]string{"POST", "OPTIONS"}, "/w7panel/login", controller.OIDC{}.LoginFromW7panel)

		group := cors.Group("/system", middleware.Cors{}.Process, middleware.DenyDomainReq{}.Process, middleware.Auth{}.Process, middleware.AdminUser{}.Process)

		group.Match([]string{"GET", "OPTIONS"}, "/util/db/switch/status", controller.DBSwitch{}.Status)
		group.Match([]string{"POST", "OPTIONS"}, "/util/db/switch/mysql/test", controller.DBSwitch{}.TestMySQL)
		group.Match([]string{"POST", "OPTIONS"}, "/util/db/switch/mysql/run", controller.DBSwitch{}.SwitchToMySQL)
		group.Match([]string{"GET", "OPTIONS"}, "/util/registry/storage/config/get", controller.RegistryStorage{}.GetConfig)
		group.Match([]string{"POST", "OPTIONS"}, "/util/registry/storage/s3/test", controller.RegistryStorage{}.TestS3Connection)
		group.Match([]string{"POST", "OPTIONS"}, "/util/registry/storage/config/update", controller.RegistryStorage{}.UpdateConfig)

		systemApi := root.Group("/v2/api", middleware.Cors{}.Process, middleware.DenyDomainReq{}.Process, middleware.Auth{}.Process)
		systemApi.POST("/user/add", middleware.AdminUser{}.Process, controller.UserManager{}.Create)
		systemApi.POST("/user/info", middleware.AdminUser{}.Process, controller.UserManager{}.Info)
		systemApi.POST("/user/list", middleware.AdminUser{}.Process, controller.UserManager{}.List)
		systemApi.POST("/user/edit", middleware.AdminUser{}.Process, controller.UserManager{}.Edit)
		systemApi.POST("/user/del", middleware.AdminUser{}.Process, controller.UserManager{}.Delete)

		systemApi.Match([]string{"GET", "OPTIONS"}, "/user/cur-user/info", controller.User{}.CurUserInfo)
		systemApi.Match([]string{"POST", "OPTIONS"}, "/user/cur-user/edit", controller.User{}.EditUser)

	})
}
