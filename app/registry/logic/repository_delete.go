package logic

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"sync"

	"golang.org/x/sync/errgroup"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
)

type repositoryTagStore interface {
	Tags(context.Context, string, func([]string) error) error
	Resolve(context.Context, string) (ocispec.Descriptor, error)
	Delete(context.Context, ocispec.Descriptor) error
}

const repositoryDeleteConcurrency = 4

var repositoryTagNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9_.-]{0,127}$`)

type RepositoryTagDeleteFailure struct {
	Tag   string `json:"tag"`
	Error string `json:"error"`
}

type RepositoryTagDeleteResult struct {
	DeletedTags []string                     `json:"deleted_tags"`
	FailedTags  []RepositoryTagDeleteFailure `json:"failed_tags"`
}

// A nil selection means all tags; an explicitly empty selection is a no-op.
// Resolve before deleting so shared manifests are deleted only once.
func deleteRepositoryTags(ctx context.Context, store repositoryTagStore, tags []string) (*RepositoryTagDeleteResult, error) {
	if tags == nil {
		tags = []string{}
		// Collect all pages before deleting: deleting while paging can skip tags.
		if err := store.Tags(ctx, "", func(page []string) error {
			tags = append(tags, page...)
			return nil
		}); err != nil && !clearImagesNotFound(err) {
			return nil, fmt.Errorf("读取仓库 tag 失败: %w", err)
		}
	}
	tags = slices.Clone(tags)
	slices.Sort(tags)
	tags = slices.Compact(tags)
	result := &RepositoryTagDeleteResult{
		DeletedTags: []string{}, FailedTags: []RepositoryTagDeleteFailure{},
	}
	errs := make([]error, len(tags))
	type manifestGroup struct {
		descriptor ocispec.Descriptor
		indexes    []int
	}
	manifests := map[string]*manifestGroup{}
	var mu sync.Mutex
	var resolveGroup errgroup.Group
	resolveGroup.SetLimit(repositoryDeleteConcurrency)
	for i, tag := range tags {
		resolveGroup.Go(func() error {
			desc, err := store.Resolve(ctx, tag)
			if clearImagesNotFound(err) {
				return nil
			}
			if err != nil {
				errs[i] = fmt.Errorf("读取 tag 失败: %w", err)
				return nil
			}
			mu.Lock()
			group := manifests[desc.Digest.String()]
			if group == nil {
				group = &manifestGroup{descriptor: desc}
				manifests[desc.Digest.String()] = group
			}
			group.indexes = append(group.indexes, i)
			mu.Unlock()
			return nil
		})
	}
	_ = resolveGroup.Wait()
	var deleteGroup errgroup.Group
	deleteGroup.SetLimit(repositoryDeleteConcurrency)
	for _, group := range manifests {
		deleteGroup.Go(func() error {
			if err := store.Delete(ctx, group.descriptor); err != nil && !clearImagesNotFound(err) {
				for _, i := range group.indexes {
					errs[i] = fmt.Errorf("删除 manifest 失败: %w", err)
				}
			}
			return nil
		})
	}
	_ = deleteGroup.Wait()
	for i, tag := range tags {
		if errs[i] != nil {
			result.FailedTags = append(result.FailedTags, RepositoryTagDeleteFailure{Tag: tag, Error: errs[i].Error()})
		} else {
			result.DeletedTags = append(result.DeletedTags, tag)
		}
	}
	return result, nil
}

// DeleteTags requires an explicit non-empty selection. It shares the repository
// lock with deletion and webhook processing; remote and database writes are not atomic.
func (l Repository) DeleteTags(ctx context.Context, repository *entity.RegistryRepository, tags []string) (*RepositoryTagDeleteResult, error) {
	if len(tags) == 0 {
		return nil, fmt.Errorf("请选择要删除的 tag")
	}
	for _, tag := range tags {
		if !repositoryTagNamePattern.MatchString(tag) {
			return nil, fmt.Errorf("tag 名称无效: %q", tag)
		}
	}
	name := commonlogic.BuildRepositoryName(repository.Name, repository.Namespace)
	unlock := lockRepositoryOperation(name)
	defer unlock()
	store, err := commonlogic.GetDefaultRemoteOci(name)
	if err != nil {
		return nil, err
	}
	result, err := deleteRepositoryTags(ctx, store, tags)
	if err != nil {
		return nil, err
	}
	if len(result.DeletedTags) > 0 {
		if _, err := dao.Q.RegistryRepositoryTag.WithContext(ctx).
			Where(dao.Q.RegistryRepositoryTag.RepositoryID.Eq(repository.ID), dao.Q.RegistryRepositoryTag.Name.In(result.DeletedTags...)).Delete(); err != nil {
			return nil, fmt.Errorf("远程 tag 已删除，但本地记录清理失败，请刷新后重试: %w", err)
		}
	}
	return result, nil
}

// Delete waits for remote deletion before removing local records. The two
// storage stages are not atomic; retaining records on failure allows retries.
func (l Repository) Delete(ctx context.Context, repository *entity.RegistryRepository) error {
	name := commonlogic.BuildRepositoryName(repository.Name, repository.Namespace)
	unlock := lockRepositoryOperation(name)
	defer unlock()
	store, err := commonlogic.GetDefaultRemoteOci(name)
	if err != nil {
		return err
	}
	result, err := deleteRepositoryTags(ctx, store, nil)
	if err != nil {
		return err
	}
	if len(result.FailedTags) > 0 {
		return fmt.Errorf("%d 个 tag 删除失败，已删除的 tag 不会恢复；%s: %s", len(result.FailedTags), result.FailedTags[0].Tag, result.FailedTags[0].Error)
	}
	return dao.Q.Transaction(func(tx *dao.Query) error {
		if _, err := tx.RegistryRepositoryTag.WithContext(ctx).Where(tx.RegistryRepositoryTag.RepositoryID.Eq(repository.ID)).Delete(); err != nil {
			return err
		}
		if _, err := tx.RegistryRepository.WithContext(ctx).Delete(repository); err != nil {
			return err
		}
		return (Permission{}).DelPermissionByResource(tx, name, PermissionResourceTypeRepository)
	})
}
