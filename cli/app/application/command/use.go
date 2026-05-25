package command

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/w7panel/w7panel-zpk/cli/app/application/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/src/console"
)

type Use struct {
	console.Abstract
}

func (c Use) GetName() string {
	return "use"
}

func (c Use) GetDescription() string {
	return "use zpk artifact"
}

func (c Use) Configure(cmd *cobra.Command) {
	cmd.Flags().StringP("name", "n", "", "artifact name or identifier")
	_ = cmd.MarkFlagRequired("name")
}

func (c Use) Handle(cmd *cobra.Command, args []string) {
	name, _ := cmd.Flags().GetString("name")
	name = strings.TrimSpace(name)
	registryInfo, err := logic.ParseFormulaRegistry(name)
	if err != nil {
		panic(err)
	}

	session, err := logic.LoadSession()
	if err != nil {
		panic(err)
	}
	if session.Host == "" || session.Username == "" || session.Password == "" {
		panic("please login before use")
	}

	session.Artifact = registryInfo.Artifact
	session.OciRegistry = registryInfo.OciRegistry
	session.OciRepository = registryInfo.OciRepository
	session.OciTag = registryInfo.OciTag
	session.Attachments = nil
	if err := logic.SaveSession(session); err != nil {
		panic(err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "using artifact: %s\n", registryInfo.Reference())
}
