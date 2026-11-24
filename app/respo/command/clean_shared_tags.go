package command

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"github.com/w7panel/w7panel-zpk/app/respo/logic/formula"
	systemlogic "github.com/w7panel/w7panel-zpk/app/system/logic"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/w7panel/w7panel-zpk/common/service/w7"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"github.com/we7coreteam/w7-rangine-go/v2/src/console"
	"oras.land/oras-go/v2/errdef"
)

type CleanSharedTags struct{ console.Abstract }

func (CleanSharedTags) GetName() string { return "respo:clean-shared-tags" }

func (CleanSharedTags) GetDescription() string {
	return "删除已迁移制品的旧 shared tag，保留版本 tag 和本地源文件"
}

func (command CleanSharedTags) Configure(cmd *cobra.Command) {
	cmd.Args = cobra.NoArgs
	cmd.Flags().String("identifie", "", "仅清理指定制品，省略则清理全部制品")
	cmd.Flags().Bool("dry-run", false, "仅检查并显示待清理 shared tag")
	cmd.RunE = command.run
}

func (command CleanSharedTags) Handle(cmd *cobra.Command, args []string) {
	if err := command.run(cmd, args); err != nil {
		panic(err)
	}
}

func (CleanSharedTags) run(cmd *cobra.Command, _ []string) error {
	if systemlogic.ShouldBlockWrites(facade.GetConfig()) {
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
	dryRun, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		return err
	}
	token := strings.TrimSpace(os.Getenv("W7PANEL_ACCESS_TOKEN"))
	query := dao.Q.Formula
	var lastID int32
	var found bool
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
			found, lastID = true, item.ID
			repositoryName := commonlogic.GetFormulaOciName(item.Name)
			repository, err := commonlogic.GetDefaultRemoteOci(repositoryName)
			if err != nil {
				return err
			}
			descriptor, err := repository.Resolve(cmd.Context(), legacySharedTag)
			if errors.Is(err, errdef.ErrNotFound) {
				fmt.Fprintf(cmd.OutOrStdout(), "%s:shared 不存在，跳过\n", repositoryName)
				continue
			}
			if err != nil {
				return err
			}
			files, err := readLegacySharedFiles(cmd.Context(), repository)
			if err != nil {
				return err
			}
			if err := requireSharedFilesMigrated(cmd.Context(), token, item, files); err != nil {
				return fmt.Errorf("制品 %s 不可清理: %w", item.Name, err)
			}
			if err := cleanSharedTag(cmd.Context(), repository, descriptor, dryRun); err != nil {
				return fmt.Errorf("清理 %s:shared 失败: %w", repositoryName, err)
			}
			if dryRun {
				fmt.Fprintf(cmd.OutOrStdout(), "待清理 %s:shared\n", repositoryName)
				continue
			}
			tag := dao.Q.RegistryRepositoryTag
			repo := dao.Q.RegistryRepository
			name, namespace := commonlogic.ParseRepositoryNameAndNamespace(repositoryName)
			_, err = tag.WithContext(cmd.Context()).Where(tag.Name.Eq(legacySharedTag), tag.WithContext(cmd.Context()).Columns(tag.RepositoryID).In(repo.WithContext(cmd.Context()).Select(repo.ID).Where(repo.Name.Eq(name), repo.Namespace.Eq(namespace)))).Delete()
			if err != nil {
				return fmt.Errorf("shared 已删除，本地 tag 记录清理失败: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "已清理 %s:shared\n", repositoryName)
		}
	}
	if !found && identifie != "" {
		return fmt.Errorf("制品 %q 不存在", identifie)
	}
	return nil
}

func requireSharedFilesMigrated(ctx context.Context, token string, item *entity.Formula, files map[string]string) error {
	if item.Readme == nil {
		return errors.New("请先运行 respo:migrate-documents 迁移简介和文档")
	}
	paths := []string{}
	for filename := range files {
		if filename == "readme.md" || filename == "docs/.order" {
			continue
		}
		if !strings.HasPrefix(filename, "docs/") || !strings.HasSuffix(strings.ToLower(filename), ".md") {
			return fmt.Errorf("存在未处理的旧资源 %q，保留 shared tag", filename)
		}
		paths = append(paths, strings.TrimPrefix(filename, "docs/"))
	}
	if len(paths) == 0 {
		return nil
	}
	if item.Setting == nil || item.Setting.DocumentRef == "" {
		return errors.New("旧文档尚未迁移")
	}
	tree, err := w7.DocumentSdk.Tree(ctx, token, item.Setting.DocumentRef)
	if err != nil {
		return err
	}
	existing := make(map[string]bool, len(tree.Files))
	for _, file := range tree.Files {
		existing[file.Path] = true
	}
	for _, path := range paths {
		if path == "index.md" || !existing[path] {
			return fmt.Errorf("旧文档 %q 未完成迁移", path)
		}
	}
	return nil
}

type sharedTagStore interface {
	Tags(context.Context, string, func([]string) error) error
	Resolve(context.Context, string) (ocispec.Descriptor, error)
	Delete(context.Context, ocispec.Descriptor) error
}

// Distribution deletes a manifest by digest, affecting every tag that refers to it.
func cleanSharedTag(ctx context.Context, store sharedTagStore, descriptor ocispec.Descriptor, dryRun bool) error {
	if err := store.Tags(ctx, "", func(tags []string) error {
		for _, tag := range tags {
			if tag == legacySharedTag {
				continue
			}
			other, err := store.Resolve(ctx, tag)
			if err != nil {
				return err
			}
			if other.Digest == descriptor.Digest {
				return fmt.Errorf("与 tag %q 共用 manifest，拒绝删除", tag)
			}
		}
		return nil
	}); err != nil {
		return err
	}
	if dryRun {
		return nil
	}
	if err := store.Delete(ctx, descriptor); err != nil && !errors.Is(err, errdef.ErrNotFound) {
		return err
	}
	return nil
}
