package application

import (
	"github.com/w7panel/w7-tradition-plugin/app/application/command"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/console"
)

type Provider struct {
}

func (*Provider) Register(console console.Console) {
	console.RegisterCommand(new(command.App))
	console.RegisterCommand(new(command.Plugin))
	console.RegisterCommand(new(command.Restore))
}
