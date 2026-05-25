package command

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/w7panel/w7panel-zpk/cli/app/application/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/src/console"
)

type List struct {
	console.Abstract
}

func (c List) GetName() string {
	return "list"
}

func (c List) GetDescription() string {
	return "list versions of current artifact image"
}

func (c List) Configure(cmd *cobra.Command) {
}

func (c List) Handle(cmd *cobra.Command, args []string) {
	session, err := logic.LoadSession()
	if err != nil {
		panic(err)
	}
	if session.Artifact == "" {
		panic("please run use before list")
	}
	if session.Host == "" || session.Username == "" || session.Password == "" {
		panic("please login before list")
	}

	tags, err := logic.ListRemoteTags(*session)
	if err != nil {
		panic(err)
	}
	if len(tags) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "no versions found")
		return
	}
	for _, tag := range tags {
		fmt.Fprintln(cmd.OutOrStdout(), tag)
	}
}
