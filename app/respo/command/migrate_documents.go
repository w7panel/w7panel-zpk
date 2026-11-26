package command

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	respologic "github.com/w7panel/w7panel-zpk/app/respo/logic"
	"github.com/w7panel/w7panel-zpk/app/respo/logic/formula"
	systemlogic "github.com/w7panel/w7panel-zpk/app/system/logic"
	"github.com/w7panel/w7panel-zpk/common/accessor"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/w7panel/w7panel-zpk/common/service/w7"
	documentsdk "github.com/w7panel/w7panel-zpk/common/service/w7/document"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"github.com/we7coreteam/w7-rangine-go/v2/src/console"
	"gorm.io/gen/field"
	"oras.land/oras-go/v2/registry/remote"
)

type MigrateDocuments struct{ console.Abstract }

func (MigrateDocuments) GetName() string { return "respo:migrate-documents" }

func (MigrateDocuments) GetDescription() string {
	return "迁移制品旧文档到文档系统、旧简介到数据库，保留源数据"
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
			assets, err := readLegacyFormulaAssets(cmd.Context(), depot.GetBasePath(), item.Name)
			if err != nil {
				return fmt.Errorf("读取制品 %s 旧文档失败: %w", item.Name, err)
			}
			if err := migrateFormulaDocuments(cmd.Context(), item, assets); err != nil {
				return fmt.Errorf("迁移制品 %s 失败（可重试，已复制正文不会覆盖）: %w", item.Name, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "制品 %s 文档及简介迁移完成\n", item.Name)
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
func migrateFormulaDocuments(ctx context.Context, item *entity.Formula, assets map[string]string) error {
	reference := ""
	if item.Setting != nil {
		reference = item.Setting.DocumentRef
	}
	if reference == "" {
		reference = "app/" + item.Name
		if err := w7.DocumentSdk.CreateCategory(ctx, item.UserID, documentsdk.CreateCategory{Identifie: "app", Name: "制品文档"}); err != nil {
			var responseError *documentsdk.ResponseError
			if !errors.As(err, &responseError) || responseError.StatusCode != http.StatusConflict {
				return err
			}
		}
		owner, err := (commonlogic.User{}).GetById(int(item.UserID))
		if err != nil {
			return err
		}
		title := item.Title
		if title == "" {
			title = item.Name
		}
		if err := w7.DocumentSdk.Create(ctx, item.UserID, documentsdk.CreateLibrary{Ref: reference, Name: title, OwnerID: item.UserID, OwnerName: owner.Username, Protected: 1}); err != nil {
			var responseError *documentsdk.ResponseError
			if !errors.As(err, &responseError) || responseError.StatusCode != http.StatusConflict {
				return err
			}
		}
	}
	tree, err := w7.DocumentSdk.Tree(ctx, item.UserID, reference)
	if err != nil {
		return err
	}
	existing := make(map[string]bool, len(tree.Files))
	for _, file := range tree.Files {
		existing[file.Path] = true
	}
	if !existing["index.md"] {
		depot, err := formula.NewDepot()
		if err != nil {
			return err
		}
		application, err := depot.GetFormula(item.Name, "", nil)
		if err != nil {
			return err
		}
		if application.Readme == nil {
			readme := assets["readme.md"]
			application.Readme = &readme
		}
		setting := accessor.FormulaSettingOption{}
		if application.Setting != nil {
			setting = *application.Setting
		}
		setting.DocumentRef = reference
		application.Setting = &setting
		if err := (respologic.FormulaDocument{}).SyncHomepage(ctx, application); err != nil {
			return err
		}
		existing["index.md"] = true
	}
	filenames := make([]string, 0)
	for filename := range assets {
		if strings.HasPrefix(filename, "docs/") && strings.HasSuffix(strings.ToLower(filename), ".md") {
			filenames = append(filenames, filename)
		}
	}
	sort.Strings(filenames)
	if len(filenames) > 0 {
		for _, filename := range filenames {
			documentPath := strings.TrimPrefix(filename, "docs/")
			if existing[documentPath] {
				continue
			}
			title := strings.TrimSuffix(documentPath, path.Ext(documentPath))
			if err := w7.DocumentSdk.Save(ctx, item.UserID, reference, documentPath, title, assets[filename]); err != nil {
				return err
			}
		}
		tree, err = w7.DocumentSdk.Tree(ctx, item.UserID, reference)
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
			appendPath(strings.TrimPrefix(strings.TrimSpace(filename), "docs/"))
		}
		for _, filename := range filenames {
			appendPath(strings.TrimPrefix(filename, "docs/"))
		}
		for _, file := range tree.Files {
			appendPath(file.Path)
		}
		if err := w7.DocumentSdk.Order(ctx, item.UserID, reference, order); err != nil {
			return err
		}
	}
	// Preserve settings and introductions edited since the migration started.
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
	}
	setting.DocumentRef = reference
	updates := entity.Formula{Setting: &setting}
	fields := []field.Expr{query.Setting}
	if current.Readme == nil {
		readme := assets["readme.md"]
		updates.Readme = &readme
		fields = append(fields, query.Readme)
	}
	_, err = query.WithContext(ctx).Where(query.ID.Eq(item.ID)).Select(fields...).Updates(updates)
	return err
}

const legacySharedTag = "shared"

// Only this one-off migration reads the retired shared storage.
func readLegacyFormulaAssets(ctx context.Context, basePath, name string) (map[string]string, error) {
	repository, err := commonlogic.GetDefaultRemoteOci(commonlogic.GetFormulaOciName(name))
	if err != nil {
		return nil, err
	}
	files, err := readLegacySharedFiles(ctx, repository)
	if err != nil {
		return nil, err
	}
	root := filepath.Join(basePath, formula.GetFormulaRelativeDir(name, 0), "files")
	err = filepath.WalkDir(root, func(filename string, entry fs.DirEntry, err error) error {
		if errors.Is(err, fs.ErrNotExist) && filename == root {
			return nil
		}
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("旧资源包含非普通文件: %s", filename)
		}
		content, err := os.ReadFile(filename)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, filename)
		if err == nil {
			files[filepath.ToSlash(relative)] = string(content)
		}
		return err
	})
	return files, err
}

func readLegacySharedFiles(ctx context.Context, repository *remote.Repository) (map[string]string, error) {
	files := map[string]string{}
	manifest, err := commonlogic.GetOciManifest(ctx, repository, legacySharedTag)
	if errors.Is(err, commonlogic.OciManifestNotFoundErr) {
		return files, nil
	}
	if err != nil {
		return nil, err
	}
	err = commonlogic.UnPackOciToLocal(ctx, repository, manifest, []string{commonlogic.MediaTypeFilesJson}, func(_, _ string, reader io.Reader) error {
		var layerFiles map[string]string
		if err := json.NewDecoder(reader).Decode(&layerFiles); err != nil {
			return err
		}
		for filename, content := range layerFiles {
			files[filename] = content
		}
		return nil
	})
	return files, err
}
