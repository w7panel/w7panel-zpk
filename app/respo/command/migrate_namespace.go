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
	"time"

	"github.com/mattn/go-sqlite3"

	"github.com/spf13/cobra"
	systemlogic "github.com/w7panel/w7panel-zpk/app/system/logic"
	"github.com/w7panel/w7panel-zpk/common/accessor"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"github.com/we7coreteam/w7-rangine-go/v2/src/console"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	// Snapshot existing targets before push callbacks can create repository records.
	newTargets := make(map[int32]bool, len(items))
	for _, item := range items {
		query := dao.Q.RegistryRepository
		name := strings.ToLower(strings.ReplaceAll(item.Name, "-", "_"))
		_, err := query.WithContext(cmd.Context()).Where(query.ResourceType.Eq("repository"), query.Namespace.Eq("app"), query.Name.Eq(name)).First()
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		newTargets[item.ID] = errors.Is(err, gorm.ErrRecordNotFound)
	}
	failed := 0
	fmt.Fprintln(cmd.OutOrStdout(), "阶段 1/2：复制全部 OCI 版本")
	for _, item := range items {
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
	fmt.Fprintln(cmd.OutOrStdout(), "阶段 2/2：迁移资源及权限")
	const batchSize = 100
	for start := 0; start < len(items); start += batchSize {
		end := min(start+batchSize, len(items))
		if err := migrateNamespacePermission(cmd.Context(), items[start:end], newTargets); err != nil {
			return fmt.Errorf("第 %d 批权限迁移失败，本批已回滚，前面批次已保留（OCI 复制失败 %d 项），可重新执行命令: %w", start/batchSize+1, failed, err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "第 %d 批权限已提交：%d/%d 个制品\n", start/batchSize+1, end, len(items))
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

// SQLite may fail immediately when a read transaction upgrades to a write.
// Retry only the current batch transaction; previously committed batches stay intact.
func migrateNamespacePermission(ctx context.Context, items []*entity.Formula, newTargets map[int32]bool) error {
	const attempts = 5
	var err error
	for attempt := 0; attempt < attempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		err = migrateNamespacePermissionOnce(ctx, items, newTargets)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		var sqliteErr sqlite3.Error
		if !errors.As(err, &sqliteErr) || (sqliteErr.Code != sqlite3.ErrBusy && sqliteErr.Code != sqlite3.ErrLocked) || attempt == attempts-1 {
			return err
		}
		timer := time.NewTimer(time.Duration(attempt+1) * 200 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return err
}

func migrateNamespacePermissionOnce(ctx context.Context, items []*entity.Formula, newTargets map[int32]bool) error {
	if len(items) == 0 {
		return nil
	}
	return dao.Q.Transaction(func(tx *dao.Query) error {
		db := tx.RegistryRepository.WithContext(ctx).UnderlyingDB().Session(&gorm.Session{NewDB: true}).WithContext(ctx)
		permissionDB := tx.RegistryUserPermission.WithContext(ctx).UnderlyingDB().Session(&gorm.Session{NewDB: true}).WithContext(ctx)
		const batchSize = 100
		repositories := map[string]*entity.RegistryRepository{}
		type grantKey struct {
			resource string
			userID   int32
		}
		existing := map[grantKey]bool{}
		legacy := map[string][]*entity.RegistryUserPermission{}
		// Read in bounded batches, then prepare all writes in memory.
		for start := 0; start < len(items); start += batchSize {
			end := min(start+batchSize, len(items))
			names := make([]string, 0, end-start)
			values := make([]string, 0, 2*(end-start))
			for _, item := range items[start:end] {
				name := strings.ToLower(strings.ReplaceAll(item.Name, "-", "_"))
				names = append(names, name)
				values = append(values, "app/"+name, "zpk_oci/"+name)
			}
			var rows []*entity.RegistryRepository
			if err := db.Where("resource_type = ? AND namespace IN ? AND name IN ?", "repository", []string{"app", "zpk_oci"}, names).Find(&rows).Error; err != nil {
				return err
			}
			for _, row := range rows {
				repositories[row.Namespace+"/"+row.Name] = row
			}
			var grants []*entity.RegistryUserPermission
			if err := permissionDB.Where("resource_type = ? AND resource_value IN ?", "repository", values).Find(&grants).Error; err != nil {
				return err
			}
			for _, grant := range grants {
				if strings.HasPrefix(grant.ResourceValue, "app/") {
					existing[grantKey{grant.ResourceValue, grant.UserID}] = true
				} else {
					legacy[grant.ResourceValue] = append(legacy[grant.ResourceValue], grant)
				}
			}
		}
		var resources []*entity.RegistryRepository
		var grants []*entity.RegistryUserPermission
		var inheritedVisible *int32
		for _, item := range items {
			name := strings.ToLower(strings.ReplaceAll(item.Name, "-", "_"))
			value := "app/" + name
			target := repositories[value]
			if target == nil || newTargets[item.ID] {
				visible := int32(4)
				if old := repositories["zpk_oci/"+name]; old != nil {
					visible = old.VisibleType
					if visible == 3 {
						if inheritedVisible == nil {
							space, err := tx.RegistryNamespace.WithContext(ctx).Where(tx.RegistryNamespace.Name.Eq("zpk_oci")).First()
							if err != nil {
								return err
							}
							inheritedVisible = &space.VisibleType
						}
						visible = *inheritedVisible
					}
				}
				if target == nil {
					target = &entity.RegistryRepository{ResourceType: "repository", Name: name, Namespace: "app"}
				}
				// Correct records created by push callbacks without changing other metadata.
				target.UserID, target.VisibleType = item.UserID, visible
				resources = append(resources, target)
			}
			owner := accessor.PermissionActionOption{"pull", "push"}
			rows := append([]*entity.RegistryUserPermission{}, legacy["zpk_oci/"+name]...)
			rows = append(rows, &entity.RegistryUserPermission{UserID: item.UserID, Action: &owner})
			for _, row := range rows {
				key := grantKey{value, row.UserID}
				if existing[key] {
					continue
				}
				existing[key] = true
				grants = append(grants, &entity.RegistryUserPermission{UserID: row.UserID, ResourceType: "repository", ResourceValue: value, Action: row.Action})
			}
		}
		if len(resources) > 0 {
			if err := db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "id"}}, DoUpdates: clause.AssignmentColumns([]string{"user_id", "visible_type"})}).CreateInBatches(resources, batchSize).Error; err != nil {
				return err
			}
		}
		if len(grants) > 0 {
			if err := permissionDB.CreateInBatches(grants, batchSize).Error; err != nil {
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
