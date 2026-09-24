package command

import (
	"github.com/spf13/cobra"
	"github.com/w7panel/w7-tradition-plugin/app/application/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/src/console"
)

type Restore struct{ console.Abstract }

func (c Restore) GetName() string { return "restore" }
func (c Restore) GetDescription() string {
	return "根据当前 OCI 制品重新应用受管文件"
}
func (c Restore) Configure(cmd *cobra.Command) {
	cmd.Flags().String("plugin-state-dir", "", "保存当前传统应用插件文件管理状态的目录")
	cmd.Flags().String("site-dir", "", "需要恢复受管文件的传统应用站点根目录")
	_ = cmd.MarkFlagRequired("plugin-state-dir")
	_ = cmd.MarkFlagRequired("site-dir")
}
func (c Restore) Handle(cmd *cobra.Command, args []string) {
	pluginStateDir, _ := cmd.Flags().GetString("plugin-state-dir")
	siteDir, _ := cmd.Flags().GetString("site-dir")
	result, err := logic.NewInstaller(pluginStateDir).Restore(siteDir)
	if err != nil {
		panic(err)
	}
	writeCommandJSON(cmd, result)
}
