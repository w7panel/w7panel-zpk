package command

import (
	"github.com/spf13/cobra"
	"github.com/w7panel/w7-tradition-plugin/app/application/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/src/console"
)

type App struct{ console.Abstract }
type AppUpdate struct{ console.Abstract }

func (c App) GetName() string                          { return "app" }
func (c App) GetDescription() string                   { return "管理传统应用更新" }
func (c App) Handle(cmd *cobra.Command, args []string) { _ = cmd.Help() }
func (c App) Configure(cmd *cobra.Command) {
	cmd.AddCommand(cobraCommand(new(AppUpdate)))
}

func (c AppUpdate) GetName() string { return "update" }
func (c AppUpdate) GetDescription() string {
	return "更新原应用文件并重新应用受管插件"
}
func (c AppUpdate) Configure(cmd *cobra.Command) {
	cmd.Flags().String("data-dir", "", "保存 OCI 数据和安装状态的持久化目录")
	cmd.Flags().String("site-dir", "", "传统应用实际运行的站点根目录")
	cmd.Flags().String("package-dir", "", "已解压的新传统应用包目录，用于刷新原应用文件")
	addPolicyFlag(cmd)
	_ = cmd.MarkFlagRequired("data-dir")
	_ = cmd.MarkFlagRequired("site-dir")
	_ = cmd.MarkFlagRequired("package-dir")
}
func (c AppUpdate) Handle(cmd *cobra.Command, args []string) {
	dataDir, _ := cmd.Flags().GetString("data-dir")
	siteDir, _ := cmd.Flags().GetString("site-dir")
	packageDir, _ := cmd.Flags().GetString("package-dir")
	service := logic.NewInstaller(dataDir)
	result, err := service.UpdateApplication(siteDir, packageDir, readPolicy(cmd))
	if err != nil {
		panic(err)
	}
	writeCommandJSON(cmd, result)
}
