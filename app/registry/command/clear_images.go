package command

import (
	"errors"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/w7panel/w7panel-zpk/app/registry/logic"
	systemlogic "github.com/w7panel/w7panel-zpk/app/system/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"github.com/we7coreteam/w7-rangine-go/v2/src/console"
)

type ClearImages struct{ console.Abstract }

func (ClearImages) GetName() string { return "registry:clear-images" }
func (ClearImages) GetDescription() string {
	return "清空指定 namespace 下全部镜像 tag，并清理当前 ZPK 数据库中的 namespace 和镜像数据"
}
func (c ClearImages) Configure(cmd *cobra.Command) {
	cmd.Args = cobra.NoArgs
	f := cmd.Flags()
	f.String("registry", "", "仓库 host[:port]，不带协议或路径")
	f.String("namespace", "", "目标 namespace，包含其下子路径")
	f.String("username", "", "仓库账号，默认 REGISTRY_USERNAME")
	f.String("password", "", "仓库密码，默认 REGISTRY_PASSWORD")
	f.Bool("http", false, "使用 HTTP 访问内网仓库")
	f.String("ca", "", "HTTPS 自签名 CA 文件")
	f.Bool("dry-run", false, "仅列出待删除 tag、digest 和数据库记录")
	f.Duration("timeout", time.Minute, "单次请求（含分页枚举）的超时")
	_ = cmd.MarkFlagRequired("registry")
	_ = cmd.MarkFlagRequired("namespace")
	cmd.RunE = c.run
}
func (c ClearImages) Handle(cmd *cobra.Command, args []string) {
	if err := c.run(cmd, args); err != nil {
		panic(err)
	}
}
func (ClearImages) run(cmd *cobra.Command, _ []string) error {
	if systemlogic.ShouldBlockWrites(facade.GetConfig()) {
		return errors.New("数据库迁移进行中，禁止清空镜像")
	}
	db, err := facade.GetDbFactory().Channel("default")
	if err != nil {
		return err
	}
	f := cmd.Flags()
	address, _ := f.GetString("registry")
	namespace, _ := f.GetString("namespace")
	plainHTTP, _ := f.GetBool("http")
	ca, _ := f.GetString("ca")
	dryRun, _ := f.GetBool("dry-run")
	timeout, _ := f.GetDuration("timeout")
	credential := func(name, env string) string {
		if f.Changed(name) {
			value, _ := f.GetString(name)
			return value
		}
		return os.Getenv(env)
	}
	return logic.ClearImagesAndDatabase(cmd.Context(), db, logic.ClearImagesOptions{
		Endpoint:  logic.MigrationEndpoint{Address: address, Username: credential("username", "REGISTRY_USERNAME"), Password: credential("password", "REGISTRY_PASSWORD"), PlainHTTP: plainHTTP, CAFile: ca},
		Namespace: namespace, DryRun: dryRun, Timeout: timeout, Progress: cmd.OutOrStdout(),
	})
}
