package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/app/system/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/controller"
)

type DBSwitch struct {
	controller.Abstract
}

type DBSwitchMySQLTestParams struct {
	Host     string `form:"host" json:"host" binding:"required"`
	Port     string `form:"port" json:"port" binding:"omitempty"`
	UserName string `form:"user_name" json:"user_name" binding:"required"`
	Password string `form:"password" json:"password" binding:"omitempty"`
	DBName   string `form:"db_name" json:"db_name"`
	Charset  string `form:"charset" json:"charset" binding:"omitempty"`
	Prefix   string `form:"prefix" json:"prefix" binding:"omitempty"`
}

func (c DBSwitch) Status(ctx *gin.Context) {
	manager := logic.NewDBSwitchManager()
	state, err := manager.GetState()
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}

	c.JsonResponseWithoutError(ctx, gin.H{
		"mode":          state.Mode,
		"switch_status": state.SwitchStatus,
		"error":         state.Error,
		"started_at":    state.StartedAt,
		"finished_at":   state.FinishedAt,
		"mysql":         state.MySQL,
		"can_switch":    state.SwitchStatus != logic.DBSwitchStatusRunning,
	})
}

func (c DBSwitch) TestMySQL(ctx *gin.Context) {
	params := DBSwitchMySQLTestParams{}
	if !c.Validate(ctx, &params) {
		return
	}

	err := logic.TestMySQLConnection(logic.DBSwitchMySQLConfig{
		Host:     params.Host,
		Port:     params.Port,
		UserName: params.UserName,
		Password: params.Password,
	})
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}

	c.JsonSuccessResponse(ctx)
}

func (c DBSwitch) SwitchToMySQL(ctx *gin.Context) {
	params := DBSwitchMySQLTestParams{}
	if !c.Validate(ctx, &params) {
		return
	}

	params.DBName = "w7-cd-artifact"
	params.Prefix = "ims_"
	params.Charset = "utf8mb4"

	manager := logic.NewDBSwitchManager()
	if err := manager.StartSwitchToMySQL(logic.DBSwitchMySQLConfig{
		Host:     params.Host,
		Port:     params.Port,
		UserName: params.UserName,
		Password: params.Password,
		DBName:   params.DBName,
		Charset:  params.Charset,
		Prefix:   params.Prefix,
	}); err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}

	c.JsonSuccessResponse(ctx)
}
