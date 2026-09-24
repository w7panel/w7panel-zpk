package command

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/w7panel/w7-tradition-plugin/app/application/logic"
)

const maxPluginPriority = 1000

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

func addPolicyFlag(cmd *cobra.Command) {
	cmd.Flags().String("policy-file", "", "传统应用配置 JSON 文件，包含受管插件及其优先级")
	_ = cmd.MarkFlagRequired("policy-file")
}

func readPolicy(cmd *cobra.Command) logic.Policy {
	path, _ := cmd.Flags().GetString("policy-file")
	content, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	policy, err := decodePolicy(content)
	if err != nil {
		panic(err)
	}
	return policy
}

func decodePolicy(content []byte) (logic.Policy, error) {
	type policyItem struct {
		Identifie string `json:"identifie"`
		Priority  int    `json:"priority"`
	}
	var input struct {
		Platform struct {
			Tradition struct {
				Plugins []policyItem `json:"plugins"`
			} `json:"tradition"`
		} `json:"platform"`
	}
	if err := json.Unmarshal(content, &input); err != nil {
		return logic.Policy{}, err
	}
	plugins := make(map[string]int, len(input.Platform.Tradition.Plugins))
	for _, item := range input.Platform.Tradition.Plugins {
		if item.Identifie == "" || item.Priority < 0 || item.Priority > maxPluginPriority {
			return logic.Policy{}, fmt.Errorf("invalid plugin priority for %q", item.Identifie)
		}
		plugins[item.Identifie] = item.Priority
	}
	return logic.Policy{Plugins: plugins}, nil
}

func writeCommandJSON(cmd *cobra.Command, value any) {
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		panic(err)
	}
}
