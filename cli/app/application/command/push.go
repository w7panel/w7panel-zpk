package command

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/w7panel/w7panel-zpk/cli/app/application/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/src/console"
)

type Push struct {
	console.Abstract
}

func (c Push) GetName() string {
	return "push"
}

func (c Push) GetDescription() string {
	return "pack attachments as OCI artifact and push"
}

func (c Push) Configure(cmd *cobra.Command) {
}

func (c Push) Handle(cmd *cobra.Command, args []string) {
	session, err := logic.LoadSession()
	if err != nil {
		panic(err)
	}
	if session.Artifact == "" {
		panic("please run use before push")
	}
	if len(session.Attachments) == 0 {
		panic("please attach at least one package before push")
	}
	if err := logic.PackFormulaToOci(*session); err != nil {
		panic(err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "pushed formula %s\n", session.Artifact)
}
