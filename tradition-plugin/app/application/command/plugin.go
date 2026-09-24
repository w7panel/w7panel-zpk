package command

import (
	"github.com/spf13/cobra"
	"github.com/w7panel/w7-tradition-plugin/app/application/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/src/console"
)

type Plugin struct{ console.Abstract }
type PluginInstall struct{ console.Abstract }
type PluginUninstall struct{ console.Abstract }

func (c Plugin) GetName() string                          { return "plugin" }
func (c Plugin) GetDescription() string                   { return "安装或卸载受管插件" }
func (c Plugin) Handle(cmd *cobra.Command, args []string) { _ = cmd.Help() }
func (c Plugin) Configure(cmd *cobra.Command) {
	cmd.AddCommand(cobraCommand(new(PluginInstall)))
	cmd.AddCommand(cobraCommand(new(PluginUninstall)))
}

func (c PluginInstall) GetName() string        { return "install" }
func (c PluginInstall) GetDescription() string { return "安装受管插件" }
func (c PluginInstall) Configure(cmd *cobra.Command) {
	pluginCommonFlags(cmd)
	cmd.Flags().String("package-dir", "", "已解压的插件包目录，该目录内容将作为插件层")
	_ = cmd.MarkFlagRequired("package-dir")
}
func (c PluginInstall) Handle(cmd *cobra.Command, args []string) {
	pluginStateDir, siteDir, plugin := pluginFlagValues(cmd)
	packageDir, _ := cmd.Flags().GetString("package-dir")
	service := logic.NewInstaller(pluginStateDir)
	result, err := service.InstallPlugin(siteDir, packageDir, plugin, readPolicy(cmd))
	if err != nil {
		panic(err)
	}
	writeCommandJSON(cmd, result)
}

func (c PluginUninstall) GetName() string { return "uninstall" }
func (c PluginUninstall) GetDescription() string {
	return "卸载受管插件并恢复剩余层文件"
}
func (c PluginUninstall) Configure(cmd *cobra.Command) { pluginCommonFlags(cmd) }
func (c PluginUninstall) Handle(cmd *cobra.Command, args []string) {
	pluginStateDir, siteDir, plugin := pluginFlagValues(cmd)
	service := logic.NewInstaller(pluginStateDir)
	result, err := service.UninstallPlugin(siteDir, plugin, readPolicy(cmd))
	if err != nil {
		panic(err)
	}
	writeCommandJSON(cmd, result)
}

func pluginCommonFlags(cmd *cobra.Command) {
	cmd.Flags().String("plugin-state-dir", "", "保存当前传统应用插件文件管理状态的目录")
	cmd.Flags().String("site-dir", "", "传统应用实际运行的站点根目录")
	cmd.Flags().String("plugin", "", "插件唯一标识，必须与配置文件中的 identifie 完全一致")
	addPolicyFlag(cmd)
	_ = cmd.MarkFlagRequired("plugin-state-dir")
	_ = cmd.MarkFlagRequired("site-dir")
	_ = cmd.MarkFlagRequired("plugin")
}

func pluginFlagValues(cmd *cobra.Command) (string, string, string) {
	pluginStateDir, _ := cmd.Flags().GetString("plugin-state-dir")
	siteDir, _ := cmd.Flags().GetString("site-dir")
	plugin, _ := cmd.Flags().GetString("plugin")
	return pluginStateDir, siteDir, plugin
}
