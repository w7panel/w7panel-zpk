package command

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/errdef"
	"strings"

	"github.com/spf13/cobra"
	systemlogic "github.com/w7panel/w7panel-zpk/app/system/logic"
	"github.com/w7panel/w7panel-zpk/common/accessor"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"github.com/we7coreteam/w7-rangine-go/v2/src/console"
	"gorm.io/gorm"
)

type MigrateNamespace struct{ console.Abstract }

func (MigrateNamespace) GetName() string { return "respo:migrate-namespace" }
func (MigrateNamespace) GetDescription() string {
	return "迁移旧制品权限及全部版本到 app，保留旧数据"
}
func (command MigrateNamespace) Configure(cmd *cobra.Command) {
	cmd.Args = cobra.NoArgs
	cmd.Flags().String("identifie", "", "仅迁移指定制品，省略则迁移全部")
	cmd.RunE = command.run
}
func (command MigrateNamespace) Handle(cmd *cobra.Command, args []string) {
	if err := command.run(cmd, args); err != nil {
		panic(err)
	}
}
func (MigrateNamespace) run(cmd *cobra.Command, _ []string) error {
	if systemlogic.ShouldBlockWrites(facade.GetConfig()) {
		return errors.New("数据库迁移期间禁止写入")
	}
	if facade.GetConfig().GetString("setting.depot.oci_namespace") != "app" {
		return errors.New("请先将 DEPOT_OCI_NAMESPACE 配置为 app")
	}
	if err := ensureAppNamespace(cmd.Context()); err != nil {
		return err
	}
	identifie, err := cmd.Flags().GetString("identifie")
	if err != nil {
		return err
	}
	query := dao.Q.Formula
	filter := query.WithContext(cmd.Context())
	if identifie != "" {
		filter = filter.Where(query.Name.Eq(strings.ReplaceAll(identifie, "_", "-")))
	}
	items, err := filter.Find()
	if err != nil {
		return err
	}
	failed := 0
	for _, item := range items {
		if err := migrateNamespacePermission(cmd.Context(), item); err != nil {
			if cmd.Context().Err() != nil {
				return cmd.Context().Err()
			}
			failed++
			fmt.Fprintf(cmd.ErrOrStderr(), "迁移 %s 权限失败，跳过: %v\n", item.Name, err)
			continue
		}
		versions := dao.Q.Version
		rows, err := versions.WithContext(cmd.Context()).Where(versions.FormulaID.Eq(item.ID)).Order(versions.ID.Desc()).Find()
		if err != nil {
			return err
		}
		for _, version := range rows {
			if err := migrateOciVersion(cmd.Context(), item.Name, version.Name); err != nil {
				if errors.Is(err, commonlogic.OciManifestNotFoundErr) {
					fmt.Fprintf(cmd.OutOrStdout(), "%s:%s: 无 OCI 数据，跳过\n", item.Name, version.Name)
					continue
				}
				if cmd.Context().Err() != nil {
					return cmd.Context().Err()
				}
				failed++
				fmt.Fprintf(cmd.ErrOrStderr(), "迁移 %s:%s 失败，跳过: %v\n", item.Name, version.Name, err)
				continue
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s:%s: app 版本就绪\n", item.Name, version.Name)
		}
	}
	if failed > 0 {
		return fmt.Errorf("迁移完成，%d 项失败，请查看日志后重试", failed)
	}
	return nil
}

func ensureAppNamespace(ctx context.Context) error {
	query := dao.Q.RegistryNamespace
	_, err := query.WithContext(ctx).Where(query.Name.Eq("app")).First()
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return query.WithContext(ctx).Create(&entity.RegistryNamespace{Name: "app", VisibleType: 1})
}
func migrateNamespacePermission(ctx context.Context, item *entity.Formula) error {
	name := strings.ToLower(strings.ReplaceAll(item.Name, "-", "_"))
	return dao.Q.Transaction(func(tx *dao.Query) error {
		query := tx.RegistryRepository
		target, err := query.WithContext(ctx).Where(query.ResourceType.Eq("repository"), query.Namespace.Eq("app"), query.Name.Eq(name)).First()
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if target == nil {
			visible := int32(4)
			old, err := query.WithContext(ctx).Where(query.ResourceType.Eq("repository"), query.Namespace.Eq("zpk_oci"), query.Name.Eq(name)).First()
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if old != nil {
				visible = old.VisibleType
				if visible == 3 {
					space, err := tx.RegistryNamespace.WithContext(ctx).Where(tx.RegistryNamespace.Name.Eq("zpk_oci")).First()
					if err != nil {
						return err
					}
					visible = space.VisibleType
				}
			}
			target = &entity.RegistryRepository{Name: name, Namespace: "app", UserID: item.UserID, VisibleType: visible}
			if err := query.WithContext(ctx).Create(target); err != nil {
				return err
			}
		}
		grants := tx.RegistryUserPermission
		rows, err := grants.WithContext(ctx).Where(grants.ResourceType.Eq("repository"), grants.ResourceValue.Eq("zpk_oci/"+name)).Find()
		if err != nil {
			return err
		}
		owner := accessor.PermissionActionOption{"pull", "push"}
		rows = append(rows, &entity.RegistryUserPermission{UserID: item.UserID, Action: &owner})
		for _, row := range rows {
			_, err := grants.WithContext(ctx).Where(grants.ResourceType.Eq("repository"), grants.ResourceValue.Eq("app/"+name), grants.UserID.Eq(row.UserID)).First()
			if err == nil {
				continue
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if err := grants.WithContext(ctx).Create(&entity.RegistryUserPermission{UserID: row.UserID, ResourceType: "repository", ResourceValue: "app/" + name, Action: row.Action}); err != nil {
				return err
			}
		}
		return nil
	})
}

// migrateOciVersion copies a missing legacy version into app.
func migrateOciVersion(ctx context.Context, name, tag string) error {
	name = strings.ToLower(strings.ReplaceAll(name, "-", "_"))
	source, err := commonlogic.GetDefaultRemoteOci("zpk_oci/" + name)
	if err != nil {
		return err
	}
	target, err := commonlogic.GetDefaultRemoteOci("app/" + name)
	if err != nil {
		return err
	}
	return copyLegacyOciVersion(ctx, source, target, tag, "zpk_oci/"+name)
}

func copyLegacyOciVersion(ctx context.Context, source, target oras.Target, tag, sourceName string) error {
	if _, err := target.Resolve(ctx, tag); err == nil {
		return nil
	} else if !errors.Is(err, errdef.ErrNotFound) {
		return err
	}
	descriptor, raw, err := oras.FetchBytes(ctx, source, tag, oras.DefaultFetchBytesOptions)
	if errors.Is(err, errdef.ErrNotFound) {
		return commonlogic.OciManifestNotFoundErr
	}
	if err != nil {
		return err
	}
	var manifest ocispec.Manifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return err
	}
	if manifest.Annotations == nil {
		manifest.Annotations = map[string]string{}
	}
	// Copying old CLI versions must not trigger publication or clear edited files.
	delete(manifest.Annotations, "w7.cc/auto_publish")
	manifest.Annotations["w7.cc/migrated-from"] = sourceName
	migrated, err := json.Marshal(manifest)
	if err != nil {
		return err
	}
	if err := oras.CopyGraph(ctx, source, target, descriptor, oras.DefaultCopyGraphOptions); err != nil {
		return err
	}
	migratedDescriptor := content.NewDescriptorFromBytes(descriptor.MediaType, migrated)
	if err := target.Push(ctx, migratedDescriptor, bytes.NewReader(migrated)); err != nil && !errors.Is(err, errdef.ErrAlreadyExists) {
		return err
	}
	return target.Tag(ctx, migratedDescriptor, tag)
}
