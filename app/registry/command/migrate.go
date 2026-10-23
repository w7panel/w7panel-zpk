package command

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/w7panel/w7panel-zpk/app/registry/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/src/console"
)

type Migrate struct{ console.Abstract }

func (Migrate) GetName() string { return "registry:migrate" }
func (Migrate) GetDescription() string {
	return "逐 tag 迁移全部平台镜像及 OCI 制品"
}

func (m Migrate) Configure(cmd *cobra.Command) {
	cmd.Args = cobra.NoArgs
	f := cmd.Flags()
	f.String("source", "", "源仓库 host[:port]，不带协议或路径")
	f.String("destination", "", "目标仓库 host[:port]，同名 tag 会被覆盖")
	for _, side := range []string{"source", "destination"} {
		f.String(side+"-username", "", side+" 仓库账号，优先于环境变量")
		f.String(side+"-password", "", side+" 仓库密码，优先于环境变量")
		f.Bool(side+"-http", false, "使用 HTTP 访问"+side)
		f.String(side+"-ca", "", side+" 自签名 CA 文件")
	}
	f.Bool("dry-run", false, "仅枚举源仓库并解析 digest，不写目标")
	f.Int("attempts", 3, "每项最多尝试次数")
	f.Int("concurrency", 3, "单项图复制的并发数；始终复制全部平台")
	f.Duration("item-timeout", 2*time.Hour, "每次单项尝试的超时")
	f.String("report", "registry-migration-"+time.Now().Format("20060102-150405.000000000")+".jsonl", "报告路径，拒绝覆盖已有文件")
	_ = cmd.MarkFlagRequired("source")
	_ = cmd.MarkFlagRequired("destination")
	cmd.RunE = m.run
}

func migrationEndpoint(cmd *cobra.Command, side, env string) logic.MigrationEndpoint {
	f := cmd.Flags()
	address, _ := f.GetString(side)
	plainHTTP, _ := f.GetBool(side + "-http")
	ca, _ := f.GetString(side + "-ca")
	credential := func(field, suffix string) string {
		if f.Changed(side + "-" + field) {
			value, _ := f.GetString(side + "-" + field)
			return value
		}
		return os.Getenv(env + suffix)
	}
	return logic.MigrationEndpoint{Address: address, PlainHTTP: plainHTTP, CAFile: ca,
		Username: credential("username", "_USERNAME"), Password: credential("password", "_PASSWORD")}
}

func (m Migrate) Handle(cmd *cobra.Command, args []string) {
	if err := m.run(cmd, args); err != nil {
		panic(err)
	}
}

func (Migrate) run(cmd *cobra.Command, _ []string) (returnErr error) {
	f := cmd.Flags()
	dryRun, _ := f.GetBool("dry-run")
	attempts, _ := f.GetInt("attempts")
	concurrency, _ := f.GetInt("concurrency")
	timeout, _ := f.GetDuration("item-timeout")
	path, _ := f.GetString("report")
	report, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("创建迁移报告: %w", err)
	}
	defer func() {
		if err := report.Close(); returnErr == nil {
			returnErr = err
		}
	}()
	fmt.Fprintln(cmd.OutOrStdout(), "迁移报告："+path)
	return logic.MigrateRegistry(cmd.Context(), logic.MigrationOptions{
		Source: migrationEndpoint(cmd, "source", "SOURCE_REGISTRY"), Destination: migrationEndpoint(cmd, "destination", "DESTINATION_REGISTRY"),
		DryRun: dryRun, Attempts: attempts, Concurrency: concurrency, ItemTimeout: timeout,
		Report: report, Progress: cmd.OutOrStdout(),
	})
}
