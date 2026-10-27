package logic

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/docker/distribution/reference"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/w7panel/w7panel-zpk/common/entity"
	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
	"gorm.io/gorm"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
)

type ClearImagesOptions struct {
	Endpoint  MigrationEndpoint
	Namespace string
	DryRun    bool
	Timeout   time.Duration
	Progress  io.Writer
}

// ClearImages follows the version-list deletion contract: resolve tags
// and delete their manifests by digest. Shared tags are deleted together.
func ClearImages(ctx context.Context, options ClearImagesOptions) error {
	if options.Namespace == "" || options.Namespace != strings.TrimSpace(options.Namespace) {
		return errors.New("namespace 不能为空或包含首尾空白")
	}
	if _, err := reference.WithName(options.Namespace + "/namespace-check"); err != nil || strings.ContainsAny(options.Namespace, ":@") {
		return errors.New("namespace 必须是有效的仓库路径，不含地址、tag 或 digest")
	}
	if options.Timeout <= 0 {
		return errors.New("timeout 必须大于 0")
	}
	r, transport, err := newMigrationRegistry(options.Endpoint)
	if err != nil {
		return err
	}
	defer transport.CloseIdleConnections()
	if options.Progress == nil {
		options.Progress = io.Discard
	}
	return clearImages(ctx, r, options)
}

func clearImages(ctx context.Context, r *remote.Registry, options ClearImagesOptions) error {
	var names []string
	err := func() error {
		requestCtx, cancel := context.WithTimeout(ctx, options.Timeout)
		defer cancel()
		return r.Repositories(requestCtx, "", func(page []string) error {
			for _, name := range page {
				if strings.HasPrefix(name, options.Namespace+"/") {
					names = append(names, name)
				}
			}
			return nil
		})
	}()
	if err != nil {
		return fmt.Errorf("枚举仓库失败: %w", err)
	}
	sort.Strings(names)
	type deletion struct {
		name       string
		descriptor v1.Descriptor
	}
	var pending []deletion
	tagCount := 0
	// Finish discovery before deleting anything, including tags sharing a digest.
	for _, name := range names {
		repo, err := r.Repository(ctx, name)
		if err != nil {
			return err
		}
		var tags []string
		requestCtx, cancel := context.WithTimeout(ctx, options.Timeout)
		err = repo.Tags(requestCtx, "", func(page []string) error { tags = append(tags, page...); return nil })
		cancel()
		if err != nil {
			return fmt.Errorf("列出 %s 的 tag 失败: %w", name, err)
		}
		sort.Strings(tags)
		seen := map[string]bool{}
		for _, tag := range tags {
			requestCtx, cancel := context.WithTimeout(ctx, options.Timeout)
			desc, err := repo.Resolve(requestCtx, tag)
			cancel()
			if err != nil {
				return fmt.Errorf("解析 %s:%s 失败: %w", name, tag, err)
			}
			tagCount++
			fmt.Fprintf(options.Progress, "%s:%s -> %s\n", name, tag, desc.Digest)
			if !seen[desc.Digest.String()] {
				seen[desc.Digest.String()] = true
				pending = append(pending, deletion{name, desc})
			}
		}
	}
	fmt.Fprintf(options.Progress, "发现 %d 个仓库、%d 个 tag、%d 个 manifest（仅账号可见范围）。\n", len(names), tagCount, len(pending))
	if options.DryRun {
		return nil
	}
	failed, deleted := 0, 0
	for _, item := range pending {
		if err := ctx.Err(); err != nil {
			return err
		}
		requestCtx, cancel := context.WithTimeout(ctx, options.Timeout)
		// Match the existing client: DELETE only the resolved manifest, without
		// ORAS referrer-index maintenance or any blob operations.
		scheme := "https"
		if r.PlainHTTP {
			scheme = "http"
		}
		requestCtx = auth.AppendRepositoryScope(requestCtx, registry.Reference{Registry: r.Reference.Registry, Repository: item.name}, auth.ActionPull, auth.ActionDelete)
		req, err := http.NewRequestWithContext(requestCtx, http.MethodDelete, scheme+"://"+r.Reference.Registry+"/v2/"+item.name+"/manifests/"+item.descriptor.Digest.String(), nil)
		if err == nil {
			var resp *http.Response
			resp, err = r.Client.Do(req)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode != http.StatusAccepted {
					err = fmt.Errorf("Registry 返回 HTTP %d", resp.StatusCode)
				}
			}
		}
		cancel()
		if err != nil {
			failed++
			fmt.Fprintf(options.Progress, "删除失败 %s@%s: %v\n", item.name, item.descriptor.Digest, err)
		} else {
			deleted++
			fmt.Fprintf(options.Progress, "已删除 %s@%s\n", item.name, item.descriptor.Digest)
		}
	}
	fmt.Fprintf(options.Progress, "完成：manifest 删除成功 %d，失败 %d。\n", deleted, failed)
	if failed > 0 {
		return fmt.Errorf("%d 个 manifest 删除失败，已成功的删除不会回滚", failed)
	}
	return nil
}

// ClearImagesAndDatabase cleans the configured ZPK database only after all
// remote deletes succeed. The remote and database stages are not atomic.
func ClearImagesAndDatabase(ctx context.Context, db *gorm.DB, options ClearImagesOptions) error {
	if options.Namespace == commonlogic.DefaultNamespace {
		return errors.New("default namespace 不可删除")
	}
	if options.Namespace == "" || strings.TrimSpace(options.Namespace) != options.Namespace {
		return errors.New("namespace 无效")
	}
	if _, err := reference.WithName(options.Namespace + "/namespace-check"); err != nil || strings.ContainsAny(options.Namespace, ":@") {
		return errors.New("namespace 无效")
	}
	if options.Progress == nil {
		options.Progress = io.Discard
	}
	// Read before remote deletion so database/configuration errors fail early.
	if err := clearImagesDatabase(ctx, db, options.Namespace, true, options.Progress); err != nil {
		return err
	}
	if err := ClearImages(ctx, options); err != nil {
		return err
	}
	if options.DryRun {
		return nil
	}
	if err := clearImagesDatabase(ctx, db, options.Namespace, false, options.Progress); err != nil {
		return fmt.Errorf("远程 tag 已删除，但数据库清理失败，可重试命令：%w", err)
	}
	return nil
}

func clearImagesDatabase(ctx context.Context, db *gorm.DB, namespace string, dryRun bool, out io.Writer) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var namespaces []entity.RegistryNamespace
		if err := tx.Unscoped().Find(&namespaces).Error; err != nil {
			return err
		}
		var repositories []entity.RegistryRepository
		// The first path segment is stored as Namespace, remaining paths as Name.
		root, _, _ := strings.Cut(namespace, "/")
		if err := tx.Unscoped().Where(&entity.RegistryRepository{Namespace: root}).Find(&repositories).Error; err != nil {
			return err
		}
		for _, repo := range repositories {
			name := commonlogic.BuildRepositoryName(repo.Name, repo.Namespace)
			if !strings.HasPrefix(name, namespace+"/") {
				continue
			}
			fmt.Fprintf(out, "数据库镜像：%s (id=%d)\n", name, repo.ID)
			if dryRun {
				continue
			}
			var ruleIDs []int32
			if err := tx.Model(&entity.RegistryRepositoryDeployRule{}).
				Where(&entity.RegistryRepositoryDeployRule{RepositoryID: repo.ID}).
				Pluck("id", &ruleIDs).Error; err != nil {
				return err
			}
			if len(ruleIDs) > 0 {
				if err := tx.Where(map[string]interface{}{"rule_id": ruleIDs}).Delete(&entity.RegistryRepositoryDeployRuleMatchLog{}).Error; err != nil {
					return err
				}
			}
			if err := tx.Where(&entity.RegistryRepositoryDeployRule{RepositoryID: repo.ID}).Delete(&entity.RegistryRepositoryDeployRule{}).Error; err != nil {
				return err
			}
			if err := tx.Where(&entity.RegistryRepositoryTag{RepositoryID: repo.ID}).Delete(&entity.RegistryRepositoryTag{}).Error; err != nil {
				return err
			}
			if err := tx.Where(&entity.RegistryUserPermission{ResourceType: "repository", ResourceValue: name}).Delete(&entity.RegistryUserPermission{}).Error; err != nil {
				return err
			}
			if err := tx.Delete(&repo).Error; err != nil {
				return err
			}
		}
		for _, ns := range namespaces {
			if ns.Name != namespace && !strings.HasPrefix(ns.Name, namespace+"/") {
				continue
			}
			fmt.Fprintf(out, "数据库 namespace：%s (id=%d)\n", ns.Name, ns.ID)
			if dryRun {
				continue
			}
			if err := tx.Where(&entity.RegistryUserPermission{ResourceType: "namespace", ResourceValue: ns.Name}).Delete(&entity.RegistryUserPermission{}).Error; err != nil {
				return err
			}
			if err := tx.Delete(&ns).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
