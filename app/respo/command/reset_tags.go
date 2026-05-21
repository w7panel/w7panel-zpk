package command

import (
	"github.com/spf13/cobra"
	"github.com/w7panel/w7panel-zpk/app/respo/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/src/console"
)

type ResetTags struct {
	console.Abstract
}

func (c ResetTags) GetName() string {
	return "reset:tags"
}

func (c ResetTags) GetDescription() string {
	return "reset tags"
}

func (c ResetTags) Handle(cmd *cobra.Command, args []string) {
	err := logic.Tag{}.ResetTags()
	if err != nil {
		panic(err)
	}
}
