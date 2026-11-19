package command

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	clicommon "github.com/w7panel/w7panel-zpk/cli/app/application/logic/common"
	"github.com/w7panel/w7panel-zpk/cli/app/application/logic/respo"
	"github.com/w7panel/w7panel-zpk/common/function"
	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"github.com/we7coreteam/w7-rangine-go/v2/src/console"
)

type App struct{ console.Abstract }

func (c App) GetName() string                          { return "app" }
func (c App) GetDescription() string                   { return "push, pull and update application packages" }
func (c App) Handle(cmd *cobra.Command, args []string) { _ = cmd.Help() }
func (c App) Configure(cmd *cobra.Command) {
	cmd.PersistentFlags().Bool("plain-http", false, "use HTTP for this registry")
	push := &cobra.Command{Use: "push [identifier[:version] | path]", Short: "push a complete local application directory as OCI", Args: cobra.MaximumNArgs(1), RunE: runAppPush}
	push.Flags().StringP("path", "p", ".", "application package directory")
	push.Flags().Bool("publish", false, "request version publication after pushing")
	pull := &cobra.Command{Use: "pull <identifier[:version]>", Short: "download an editable application directory", Args: cobra.ExactArgs(1), RunE: runAppPull}
	pull.Flags().StringP("path", "p", "", "new destination directory (defaults to identifier)")
	pull.Flags().String("host", "", "default pull registry (overrides login host)")
	update := &cobra.Command{Use: "update <backend|frontend|helm|manifest>", Short: "update a local application package", ValidArgs: []string{"backend", "frontend", "helm", "manifest"}, Args: cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs), RunE: runAppUpdate}
	update.Flags().StringP("path", "p", ".", "application package directory")
	update.Flags().String("app", "", "application identifier (defaults to main application)")
	update.Flags().String("dir", "", "replacement attachment directory")
	update.Flags().String("file", "", "replacement manifest or target file")
	update.Flags().String("target", "", "relative path inside the attachment")
	update.Flags().Bool("create", false, "allow creating a new target file")
	update.Flags().String("patch", "", "manifest merge patch or JSON Patch")
	update.Flags().String("patch-file", "", "YAML or JSON manifest patch file")
	update.Flags().String("type", "", "manifest patch type: merge (default) or json")
	update.MarkFlagsMutuallyExclusive("dir", "file", "patch", "patch-file")
	update.MarkFlagsOneRequired("dir", "file", "patch", "patch-file")
	cmd.AddCommand(push, pull, update)
}

func runAppUpdate(cmd *cobra.Command, args []string) error {
	directory, _ := cmd.Flags().GetString("path")
	app, _ := cmd.Flags().GetString("app")
	sourceDir, _ := cmd.Flags().GetString("dir")
	file, _ := cmd.Flags().GetString("file")
	target, _ := cmd.Flags().GetString("target")
	create, _ := cmd.Flags().GetBool("create")
	patch, _ := cmd.Flags().GetString("patch")
	patchFile, _ := cmd.Flags().GetString("patch-file")
	patchType, _ := cmd.Flags().GetString("type")
	options := respo.UpdateOptions{
		Directory: directory, App: app, Kind: args[0], Dir: sourceDir, File: file,
		Target: target, Create: create, Patch: patch, PatchFile: patchFile, PatchType: patchType,
	}
	if err := validateAppUpdate(options); err != nil {
		return err
	}
	changes, err := respo.Update(options)
	if err != nil {
		return err
	}
	for _, name := range changes {
		fmt.Fprintf(cmd.OutOrStdout(), "已更新 %s\n", name)
	}
	return nil
}

func validateAppUpdate(options respo.UpdateOptions) error {
	if options.Dir == "" && options.File == "" && options.Patch == "" && options.PatchFile == "" {
		return fmt.Errorf("--dir、--file、--patch、--patch-file 的输入内容不能为空")
	}
	if options.App != "" {
		if err := commonlogic.ValidateAppIdentifier(options.App); err != nil {
			return err
		}
	}
	patching := options.Patch != "" || options.PatchFile != ""
	if patching && options.Kind != "manifest" {
		return fmt.Errorf("--patch 和 --patch-file 仅适用于 manifest")
	}
	if options.PatchType != "" && (!patching || (options.PatchType != "merge" && options.PatchType != "json")) {
		return fmt.Errorf("--type 仅适用于 manifest 局部修改，可选 merge 或 json")
	}
	if options.Kind == "manifest" && (options.Dir != "" || options.Target != "" || options.Create) {
		return fmt.Errorf("manifest 不支持 --dir、--target 或 --create")
	}
	if options.Kind != "manifest" && options.File != "" && options.Target == "" {
		return fmt.Errorf("完整附件更新使用 --dir 指定解压目录；--file 必须配合 --target 更新单个文件")
	}
	if options.Dir != "" && options.Target != "" {
		return fmt.Errorf("--dir 与 --target 不能同时使用")
	}
	if options.Create && options.Target == "" {
		return fmt.Errorf("--create 必须配合 --target 使用")
	}
	if options.Target != "" {
		_, err := function.RelativePath(options.Target)
		return err
	}
	return nil
}

func runAppPush(cmd *cobra.Command, args []string) error {
	directory, _ := cmd.Flags().GetString("path")
	reference := ""
	if len(args) > 0 {
		reference = args[0]
		if info, err := os.Stat(reference); err == nil && info.IsDir() {
			if cmd.Flags().Changed("path") {
				return fmt.Errorf("目录位置参数和 --path 不能同时指定")
			}
			directory, reference = reference, ""
		}
	}
	if strings.ContainsAny(reference, "/\\") {
		return fmt.Errorf("app push 仅接受应用标识[:版本]或本地目录，推送仓库使用 login 保存的默认仓库")
	}
	if _, version, ok := strings.Cut(reference, ":"); ok && version != "" && version != "latest" && !clicommon.IsVersion(version) {
		return fmt.Errorf("应用版本格式应为 1.2.3 或 v1.2.3")
	}
	if reference == "" {
		identifier, err := respo.ReadAppIdentifier(directory)
		if err != nil {
			return err
		}
		reference = identifier
	}
	session, err := clicommon.LoadSession()
	if err != nil {
		return err
	}
	plainHTTP, _ := cmd.Flags().GetBool("plain-http")
	namespace := facade.GetConfig().GetString("setting.depot.oci_namespace")
	if namespace == "" {
		namespace = "zpk_oci"
	}
	target, err := clicommon.ResolveTarget(reference, session.Host, namespace, plainHTTP)
	if err != nil {
		return err
	}
	identifier := target.Identifie
	target.Identifie = strings.ReplaceAll(identifier, "_", "-")
	target.Repository = strings.TrimSuffix(target.Repository, identifier) + strings.ReplaceAll(target.Identifie, "-", "_")
	if err = commonlogic.ValidateAppIdentifier(target.Identifie); err != nil {
		return err
	}
	credential, err := session.RegistryCredential(target.Registry)
	if err != nil {
		return err
	}
	if credential.Username == "" {
		return fmt.Errorf("请先执行 zpk login --host %s", target.Registry)
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Minute)
	defer cancel()
	publish, _ := cmd.Flags().GetBool("publish")
	_, err = respo.Push(ctx, target, *session, directory, publish, cmd.OutOrStdout())
	return err
}
func runAppPull(cmd *cobra.Command, args []string) error {
	session, err := clicommon.LoadSession()
	if err != nil {
		return err
	}
	host, _ := cmd.Flags().GetString("host")
	if host == "" {
		host = session.Host
	}
	plainHTTP, _ := cmd.Flags().GetBool("plain-http")
	namespace := facade.GetConfig().GetString("setting.depot.oci_namespace")
	if namespace == "" {
		namespace = "zpk_oci"
	}
	target, err := clicommon.ResolveTarget(args[0], host, namespace, plainHTTP)
	if err != nil {
		return err
	}
	identifier := target.Identifie
	target.Identifie = strings.ReplaceAll(identifier, "_", "-")
	target.Repository = strings.TrimSuffix(target.Repository, identifier) + strings.ReplaceAll(target.Identifie, "-", "_")
	if err = commonlogic.ValidateAppIdentifier(target.Identifie); err != nil {
		return err
	}
	if target.Version != "" && target.Version != "latest" && !clicommon.IsVersion(target.Version) {
		return fmt.Errorf("应用版本格式应为 1.2.3 或 v1.2.3")
	}
	directory, _ := cmd.Flags().GetString("path")
	if directory == "" {
		directory = target.Identifie
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Minute)
	defer cancel()
	result, err := respo.Pull(ctx, target, *session, directory)
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "已拉取 %s:%s → %s\n%s\n", result.Reference, result.Version, directory, result.Digest)
	return nil
}
