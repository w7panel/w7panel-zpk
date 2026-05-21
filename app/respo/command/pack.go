package command

import (
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/w7panel/w7panel-zpk/app/respo/logic"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/we7coreteam/w7-rangine-go/v2/src/console"
)

type Pack struct {
	console.Abstract
}

func (pack Pack) GetName() string {
	return "respo:pack"
}

func (pack Pack) GetDescription() string {
	return "respo command"
}

func (pack Pack) Handle(cmd *cobra.Command, args []string) {
	list, err := dao.Q.Formula.Find()
	if err != nil {
		panic(err)
		return
	}

	depot, _ := logic.NewDepot()
	for _, item := range list {
		formula, err := depot.GetFormula(item.Name, "", nil)
		if err != nil {
			slog.Error("GetFormula err", "name", item.Name, "err", err)
			continue
		}
		path, err := logic.PackFormulaToHelmAndPack(*formula, true)
		if err != nil {
			slog.Error("packFormulaToHelmAndPack err", "name", item.Name, "err", err)
			continue
		}

		slog.Info("packFormulaToHelmAndPack success", "name", item.Name, "path", path)
	}
}
