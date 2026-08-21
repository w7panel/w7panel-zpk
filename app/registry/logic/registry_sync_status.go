package logic

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/docker/distribution/manifest/ocischema"
	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/w7panel/w7panel-zpk/common/service/registry/client"
	registrytypes "github.com/w7panel/w7panel-zpk/common/service/registry/types"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
)

const defaultSyncStatusNamespace = "w7-sync"
const syncStatusAnnotationPrefix = "com.w7.registry.sync."

// TagSyncStatus is the user-facing progress of an image being copied into the cache registry.
type TagSyncStatus struct {
	TagName   string    `json:"tag_name"`
	Status    string    `json:"status"`
	Completed int       `json:"completed"`
	Total     int       `json:"total"`
	Progress  int       `json:"progress"`
	Error     string    `json:"error,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

type RegistrySyncStatus struct{}

func syncStatusLocation(namespace, repository, reference string) (string, string) {
	namespace = strings.Trim(namespace, "/")
	if namespace == "" {
		namespace = defaultSyncStatusNamespace
	}
	sum := sha256.Sum256([]byte(repository + "\x00" + reference))
	return namespace + "/" + repository, "sync-" + hex.EncodeToString(sum[:])
}

func (RegistrySyncStatus) ListStatuses(repositoryName string) ([]TagSyncStatus, error) {
	namespace := facade.GetConfig().GetString("setting.registry.sync_status_namespace")
	statusRepository, _ := syncStatusLocation(namespace, repositoryName, "")
	registryClient := commonlogic.GetDefaultRegistryClient()
	statusTags, err := registryClient.ListTags(statusRepository)
	if err != nil {
		if registrytypes.IsCode(err, registrytypes.NotFoundCode) {
			return []TagSyncStatus{}, nil
		}
		return nil, err
	}
	statuses := make([]TagSyncStatus, 0, len(statusTags))
	for _, statusTag := range statusTags {
		status, err := readSyncStatus(registryClient, statusRepository, statusTag)
		if err != nil {
			slog.Warn("读取单个镜像同步状态失败", "err", err, "repository", statusRepository, "tag", statusTag)
			continue
		}
		if (status.Status == "syncing" || status.Status == "failed") && status.TagName != "" {
			statuses = append(statuses, *status)
		}
	}
	return statuses, nil
}

func readSyncStatus(registryClient client.Client, statusRepository, statusTag string) (*TagSyncStatus, error) {
	manifest, _, err := registryClient.PullManifest(statusRepository, statusTag)
	if err != nil {
		return nil, err
	}

	ociManifest, ok := manifest.(*ocischema.DeserializedManifest)
	if !ok {
		return nil, fmt.Errorf("unexpected sync status manifest type %T", manifest)
	}
	annotations := ociManifest.Annotations
	annotation := func(name string) string { return annotations[syncStatusAnnotationPrefix+name] }
	completed, err := strconv.Atoi(annotation("completed"))
	if err != nil {
		return nil, fmt.Errorf("invalid completed annotation: %w", err)
	}
	total, err := strconv.Atoi(annotation("total"))
	if err != nil {
		return nil, fmt.Errorf("invalid total annotation: %w", err)
	}
	progress, err := strconv.Atoi(annotation("progress"))
	if err != nil {
		return nil, fmt.Errorf("invalid progress annotation: %w", err)
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, annotation("updated_at"))
	if err != nil {
		return nil, fmt.Errorf("invalid updated_at annotation: %w", err)
	}
	return &TagSyncStatus{
		TagName: annotation("reference"), Status: annotation("status"), Completed: completed, Total: total, Progress: progress,
		Error: annotation("error"), UpdatedAt: updatedAt,
	}, nil
}
