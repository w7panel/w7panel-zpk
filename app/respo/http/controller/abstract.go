package controller

import (
	"github.com/w7panel/w7panel-zpk/app/respo/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/controller"
)

type Abstract struct {
	controller.Abstract
}

func (c Abstract) getDepot() *logic.Depot {
	depot, _ := logic.NewDepot()
	return depot
}
