package application

import (
	"github.com/w7panel/w7panel-zpk/cli/app/application/command"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/console"
)

type Provider struct {
}

func (*Provider) Register(console console.Console) {
	console.RegisterCommand(new(command.Login))
	console.RegisterCommand(new(command.Use))
	console.RegisterCommand(new(command.Attach))
	console.RegisterCommand(new(command.Push))
}
