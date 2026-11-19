package command

import (
	"context"
	"time"

	"github.com/spf13/cobra"
	clicommon "github.com/w7panel/w7panel-zpk/cli/app/application/logic/common"
	imagelogic "github.com/w7panel/w7panel-zpk/cli/app/application/logic/image"
	"github.com/we7coreteam/w7-rangine-go/v2/src/console"
)

type Image struct{ console.Abstract }

func (c Image) GetName() string                          { return "image" }
func (c Image) GetDescription() string                   { return "push container images through Docker" }
func (c Image) Handle(cmd *cobra.Command, args []string) { _ = cmd.Help() }
func (c Image) Configure(cmd *cobra.Command) {
	push := &cobra.Command{Use: "push <local-image[:tag]>", Short: "authenticate, tag and push a local Docker image", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		session, err := clicommon.LoadSession()
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Minute)
		defer cancel()
		return imagelogic.PushImage(ctx, args[0], *session, cmd.OutOrStdout(), cmd.ErrOrStderr())
	}}
	cmd.AddCommand(push)
}
