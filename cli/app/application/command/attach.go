package command

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/w7panel/w7panel-zpk/cli/app/application/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/src/console"
)

type Attach struct {
	console.Abstract
}

type AttachAdd struct {
	console.Abstract
}

func (c Attach) GetName() string {
	return "attach"
}

func (c Attach) GetDescription() string {
	return "manage artifact attachments"
}

func (c Attach) Handle(cmd *cobra.Command, args []string) {
	_ = cmd.Help()
}

func (c Attach) Configure(cmd *cobra.Command) {
	cmd.AddCommand(cobraCommand(new(AttachAdd)))
}

func (c AttachAdd) GetName() string {
	return "add"
}

func (c AttachAdd) GetDescription() string {
	return "add artifact package"
}

func (c AttachAdd) Configure(cmd *cobra.Command) {
	cmd.Flags().StringP("path", "p", "", "local package path")
	cmd.Flags().StringP("type", "t", "", "attachment type: frontend, backend, helm")
	cmd.Flags().StringP("sub_artifact", "a", "", "sub artifact identifie, empty means main artifact")
	_ = cmd.MarkFlagRequired("path")
	_ = cmd.MarkFlagRequired("type")
}

func (c AttachAdd) Handle(cmd *cobra.Command, args []string) {
	path, _ := cmd.Flags().GetString("path")
	attachType, _ := cmd.Flags().GetString("type")
	subArtifact, _ := cmd.Flags().GetString("sub_artifact")

	path = strings.TrimSpace(path)
	subArtifact = strings.Trim(strings.TrimSpace(subArtifact), "/")
	if path == "" {
		panic("local package path is required")
	}
	normalizedType, err := logic.NormalizeAttachType(strings.TrimSpace(attachType))
	if err != nil {
		panic(err)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		panic(err)
	}
	info, err := os.Stat(absPath)
	if err != nil {
		panic(err)
	}
	if info.IsDir() {
		panic("attachment path must be a file")
	}

	session, err := logic.LoadSession()
	if err != nil {
		panic(err)
	}
	if session.Artifact == "" {
		panic("please run use before attach")
	}
	if subArtifact == "" {
		subArtifact = session.Artifact
	}
	subArtifact = strings.ReplaceAll(subArtifact, "_", "-")

	replaced := false
	for index, item := range session.Attachments {
		if item.Type == normalizedType && item.Artifact == subArtifact {
			session.Attachments[index] = logic.Attachment{
				Path:     absPath,
				Type:     normalizedType,
				AddedAt:  time.Now(),
				Artifact: subArtifact,
			}
			replaced = true
			break
		}
	}
	if !replaced {
		session.Attachments = append(session.Attachments, logic.Attachment{
			Path:     absPath,
			Type:     normalizedType,
			AddedAt:  time.Now(),
			Artifact: subArtifact,
		})
	}

	if err := logic.SaveSession(session); err != nil {
		panic(err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "attached %s as %s for %s\n", absPath, normalizedType, subArtifact)
}

func cobraCommand(command interface {
	GetName() string
	GetDescription() string
	Configure(cmd *cobra.Command)
	Handle(cmd *cobra.Command, args []string)
}) *cobra.Command {
	cmd := &cobra.Command{
		Use:   command.GetName(),
		Short: command.GetDescription(),
		Run:   command.Handle,
	}
	command.Configure(cmd)
	return cmd
}
