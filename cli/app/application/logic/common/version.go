package common

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/w7panel/w7panel-zpk/common/function"
	"oras.land/oras-go/v2/errdef"
	"oras.land/oras-go/v2/registry/remote"
)

var ErrNoVersions = errors.New("仓库没有符合版本规则的版本")

// IsVersion accepts the existing three numeric components with an optional v.
func IsVersion(value string) bool {
	parts := strings.Split(strings.TrimPrefix(value, "v"), ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if part == "" || strings.Trim(part, "0123456789") != "" {
			return false
		}
		if _, err := strconv.Atoi(part); err != nil {
			return false
		}
	}
	return true
}

func ListVersions(ctx context.Context, repository *remote.Repository) ([]string, error) {
	tags := []string{}
	err := repository.Tags(ctx, "", func(page []string) error {
		for _, tag := range page {
			if IsVersion(tag) {
				tags = append(tags, tag)
			}
		}
		return nil
	})
	if errors.Is(err, errdef.ErrNotFound) {
		return tags, nil
	}
	if err != nil {
		return nil, err
	}
	function.SortTagsDesc(tags)
	return tags, nil
}

// NextVersion ignores non-version tags and retains the original patch rule.
func NextVersion(tags []string) (string, error) {
	versions := make([]string, 0, len(tags))
	for _, tag := range tags {
		if IsVersion(tag) {
			versions = append(versions, tag)
		}
	}
	if len(versions) == 0 {
		return "1.0.0", nil
	}
	function.SortTagsDesc(versions)
	latest := versions[0]
	parts := strings.Split(latest, ".")
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return "", fmt.Errorf("版本 %q 无法自动递增，请显式指定版本: %w", latest, err)
	}
	if patch == math.MaxInt {
		return "", fmt.Errorf("版本 %q 无法自动递增，请显式指定版本", latest)
	}
	parts[2] = strconv.Itoa(patch + 1)
	return strings.Join(parts, "."), nil
}

func ResolveVersion(ctx context.Context, repository *remote.Repository, requested string) (string, v1.Descriptor, error) {
	if requested == "" || requested == "latest" {
		tags, err := ListVersions(ctx, repository)
		if err != nil {
			return "", v1.Descriptor{}, err
		}
		if len(tags) == 0 {
			return "", v1.Descriptor{}, ErrNoVersions
		}
		requested = tags[0]
	}
	descriptor, err := repository.Resolve(ctx, requested)
	return requested, descriptor, err
}
