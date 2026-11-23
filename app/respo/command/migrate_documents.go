package command

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/w7panel/w7panel-zpk/app/respo/logic/formula"
	systemlogic "github.com/w7panel/w7panel-zpk/app/system/logic"
	"github.com/w7panel/w7panel-zpk/common/accessor"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	documentsdk "github.com/w7panel/w7panel-zpk/common/service/document"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"github.com/we7coreteam/w7-rangine-go/v2/src/console"
)

type MigrateDocuments struct{ console.Abstract }

func (MigrateDocuments) GetName() string { return "respo:migrate-documents" }

func (MigrateDocuments) GetDescription() string {
	return "一次性将制品旧文档复制到独立文档系统，保留原文件"
}

func (command MigrateDocuments) Configure(cmd *cobra.Command) {
	cmd.Args = cobra.NoArgs
	cmd.Flags().String("identifie", "", "仅迁移指定制品，省略则迁移全部制品")
	cmd.RunE = command.run
}

func (command MigrateDocuments) Handle(cmd *cobra.Command, args []string) {
	if err := command.run(cmd, args); err != nil {
		panic(err)
	}
}

func (MigrateDocuments) run(cmd *cobra.Command, _ []string) error {
	config := facade.GetConfig()
	if systemlogic.ShouldBlockWrites(config) {
		return errors.New("数据库迁移进行中，当前已禁止写操作")
	}
	token := strings.TrimSpace(os.Getenv("W7PANEL_ACCESS_TOKEN"))
	if token == "" {
		return errors.New("请通过 W7PANEL_ACCESS_TOKEN 环境变量提供创始人的面板访问凭据")
	}
	identifie, err := cmd.Flags().GetString("identifie")
	if err != nil {
		return err
	}
	if identifie != "" {
		identifie = strings.ReplaceAll(strings.TrimSpace(identifie), "_", "-")
		if err := formula.ValidateIdentifie(identifie); err != nil {
			return err
		}
	}
	client := documentsdk.New(config.GetString("setting.document.base_url"))
	user, err := client.CurrentUser(cmd.Context(), token)
	if err != nil {
		return err
	}
	if user.ID <= 0 || !user.Admin {
		return errors.New("文档迁移需要创始人权限")
	}
	depot, err := formula.NewDepot()
	if err != nil {
		return err
	}
	query := dao.Q.Formula
	var lastID int32
	total := 0
	for {
		rows := query.WithContext(cmd.Context()).Where(query.ID.Gt(lastID))
		if identifie != "" {
			rows = rows.Where(query.Name.Eq(identifie))
		}
		batch, err := rows.Order(query.ID).Limit(100).Find()
		if err != nil {
			return err
		}
		if len(batch) == 0 {
			break
		}
		for _, item := range batch {
			loaded, err := depot.GetFormula(item.Name, "", nil)
			if err != nil {
				return fmt.Errorf("读取制品 %s 失败: %w", item.Name, err)
			}
			assets, err := depot.GetAssetFiles(loaded)
			if err != nil {
				return fmt.Errorf("读取制品 %s 旧文档失败: %w", item.Name, err)
			}
			if err := migrateFormulaDocuments(cmd.Context(), client, token, item, assets); err != nil {
				return fmt.Errorf("迁移制品 %s 失败（可重试，已复制正文不会覆盖）: %w", item.Name, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "制品 %s 文档迁移完成\n", item.Name)
			total++
			lastID = item.ID
		}
	}
	if total == 0 && identifie != "" {
		return fmt.Errorf("制品 %q 不存在", identifie)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "文档迁移完成，共 %d 个制品\n", total)
	return nil
}

// Migration stays in this one-off command rather than the normal request flow.
func migrateFormulaDocuments(ctx context.Context, client *documentsdk.Client, token string, item *entity.Formula, assets map[string]string) error {
	reference := ""
	if item.Setting != nil {
		reference = item.Setting.DocumentRef
	}
	if reference == "" {
		title := item.Title
		if title == "" {
			title = item.Name
		}
		var err error
		reference, err = client.Initialize(ctx, token, item.Name, title, item.UserID)
		if err != nil {
			return err
		}
	}
	filenames := make([]string, 0)
	for filename := range assets {
		if strings.HasPrefix(filename, "docs/") && strings.HasSuffix(strings.ToLower(filename), ".md") {
			filenames = append(filenames, filename)
		}
	}
	sort.Strings(filenames)
	if len(filenames) > 0 {
		tree, err := client.Tree(ctx, token, reference)
		if err != nil {
			return err
		}
		existing := make(map[string]bool, len(tree.Files))
		for _, file := range tree.Files {
			existing[file.Path] = true
		}
		for _, filename := range filenames {
			if existing[filename] {
				continue
			}
			title := strings.TrimSuffix(strings.TrimPrefix(filename, "docs/"), path.Ext(filename))
			if err := client.Save(ctx, token, reference, filename, title, assets[filename], 0); err != nil {
				return err
			}
		}
		tree, err = client.Tree(ctx, token, reference)
		if err != nil {
			return err
		}
		existing = make(map[string]bool, len(tree.Files))
		for _, file := range tree.Files {
			existing[file.Path] = true
		}
		order := make([]string, 0, len(tree.Files))
		added := make(map[string]bool, len(tree.Files))
		appendPath := func(filename string) {
			if existing[filename] && !added[filename] {
				order = append(order, filename)
				added[filename] = true
			}
		}
		appendPath("index.md")
		for _, filename := range strings.Split(assets["docs/.order"], ",") {
			appendPath(strings.TrimSpace(filename))
		}
		for _, filename := range filenames {
			appendPath(filename)
		}
		for _, file := range tree.Files {
			appendPath(file.Path)
		}
		if err := client.Order(ctx, token, reference, order, tree.Revision); err != nil {
			return err
		}
	}
	// GetFormula may initialize BaseInfo; keep the latest settings when adding the reference.
	query := dao.Q.Formula
	current, err := query.WithContext(ctx).Where(query.ID.Eq(item.ID)).First()
	if err != nil {
		return err
	}
	setting := accessor.FormulaSettingOption{}
	if current.Setting != nil {
		setting = *current.Setting
	}
	if setting.DocumentRef != "" {
		if setting.DocumentRef != reference {
			return errors.New("制品文档标识已变更，请检查后重试")
		}
		return nil
	}
	setting.DocumentRef = reference
	_, err = query.WithContext(ctx).Where(query.ID.Eq(item.ID)).Select(query.Setting).Updates(entity.Formula{Setting: &setting})
	return err
}
