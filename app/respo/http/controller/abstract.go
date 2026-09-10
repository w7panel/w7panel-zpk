package controller

import (
	"github.com/w7panel/w7panel-zpk/app/respo/logic/formula"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/controller"
)

type Abstract struct {
	controller.Abstract
}

func (c Abstract) getDepot() *formula.Depot {
	depot, _ := formula.NewDepot()
	return depot
}
