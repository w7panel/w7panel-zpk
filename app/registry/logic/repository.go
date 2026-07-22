package logic

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest/manifestlist"
	"github.com/docker/distribution/manifest/ocischema"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	"github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/w7panel/w7panel-zpk/common/types/registry"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
)

type TagInfo struct {
	RepositoryName string
	TagName        string
	Digest         string
	Manifest       distribution.Manifest
	Platform       string
	Type           string
	CreatedAt      string
	UpdatedAt      string
	Size           int64
}

type Repository struct {
	logic.Logic
}

func (l Repository) GetByIdWithUser(id int, user *entity.RegistryUser) (*entity.RegistryRepository, error) {
	query := dao.Q.RegistryRepository.Where(dao.RegistryRepository.ID.Eq(int32(id)))
	if user != nil && !(logic.User{}.IsAdminUser(user)) {
		query = query.Where(dao.RegistryRepository.UserID.Eq(user.ID))
	}
	return query.First()
}

func (l Repository) GetByIdWithUserPermission(id int, user *entity.RegistryUser) (*entity.RegistryRepository, error) {
	model, _ := dao.Q.RegistryRepository.Where(dao.RegistryRepository.ID.Eq(int32(id))).First()
	if model == nil {
		return nil, nil
	}
	if user != nil && !(logic.User{}.IsAdminUser(user)) && model.UserID != user.ID {
		exists := Permission{}.UserHasPermissionByResource(int(user.ID), model.Namespace, PermissionResourceTypeNamespace)
		if !exists {
			return nil, errors.New("user no permission")
		}
	}

	return model, nil
}

func (l Repository) GetByNameAndNamespace(repositoryName string, namespace string) (*entity.RegistryRepository, error) {
	return dao.Q.RegistryRepository.Where(dao.RegistryRepository.Name.Eq(repositoryName)).Where(dao.RegistryRepository.Namespace.Eq(namespace)).First()
}

func (l Repository) OnRepositoryPrepareOperate(payload registry.RegistryRepositoryPayLoad) {
	if slices.Contains(payload.Scope.Actions, string(PermissionActionTypePush)) {
		repositoryName, namespace := logic.ParseRepositoryNameAndNamespace(payload.Scope.Name)

		userId := int32(0)
		if payload.User != nil {
			userId = payload.User.ID
		}

		namespaceModel, _ := Namespace{}.GetByName(namespace)
		if namespaceModel == nil {
			ignoreNamespaceConfig := facade.GetConfig().GetString("setting.registry.list_ignore_namespaces")
			ignoredNamespaces := strings.Split(ignoreNamespaceConfig, ",")
			if slices.Contains(ignoredNamespaces, namespace) {
				namespaceVisibleType := VisibleTypePrivate
				defaultNamespaceVisible := facade.GetConfig().GetInt("setting.registry.namespaces_default_visible." + namespace)
				if defaultNamespaceVisible > 0 {
					namespaceVisibleType = defaultNamespaceVisible
				}
				namespaceModel = &entity.RegistryNamespace{
					Name:        namespace,
					VisibleType: int32(namespaceVisibleType),
					UserID:      userId,
				}
				err := dao.Q.RegistryNamespace.Create(namespaceModel)
				if err != nil {
					slog.Error("fail to create registry namespace", "namespace", namespace, "error", err)
				}
			} else {
				slog.Info("ignore namespace", "namespace", namespace, "payload", payload)
				return
			}
		}

		repositoryModel, _ := Repository{}.GetByNameAndNamespace(repositoryName, namespace)
		if repositoryModel == nil {
			if userId == 0 && namespaceModel != nil {
				userId = namespaceModel.UserID
			}
			err := dao.Q.RegistryRepository.Create(&entity.RegistryRepository{
				Name:        repositoryName,
				Namespace:   namespace,
				VisibleType: int32(VisibleTypeFollowNamespace),
				Desc:        "",
				UserID:      userId,
			})
			if err != nil {
				slog.Error("fail to create registry repository", "namespace", namespace, "repository", repositoryName, "scope", payload.Scope, "error", err)
			}
		}
	}
}

func (l Repository) OnRepositoryPushed(payload registry.RegistryRepositoryWebHookPayLoad) {
	repositoryName, namespace := logic.ParseRepositoryNameAndNamespace(payload.Event.Target.Repository)
	repositoryModel, _ := Repository{}.GetByNameAndNamespace(repositoryName, namespace)
	if repositoryModel != nil {
		tag, _ := dao.Q.RegistryRepositoryTag.Where(dao.Q.RegistryRepositoryTag.RepositoryID.Eq(repositoryModel.ID)).
			Where(dao.Q.RegistryRepositoryTag.Name.Eq(payload.Event.Target.Tag)).First()
		if tag == nil {
			err := dao.Q.RegistryRepositoryTag.Create(&entity.RegistryRepositoryTag{
				RepositoryID: repositoryModel.ID,
				Name:         payload.Event.Target.Tag,
			})
			if err != nil {
				slog.Error("create registry repository tag", "payload", payload, "error", err)
			}
		} else {
			curTime := time.Now()
			_, err := dao.Q.RegistryRepositoryTag.Where(dao.Q.RegistryRepositoryTag.ID.Eq(tag.ID)).Updates(entity.RegistryRepositoryTag{
				LatestPushAt: &curTime,
			})
			if err != nil {
				slog.Error("update registry repository tag", "payload", payload, "error", err)
			}
		}
	}

	go facade.GetEvent().Publish(registry.RegistryRepositoryAfterPushedEvent, payload)
}

func (l Repository) OnRepositoryPulled(payload registry.RegistryRepositoryWebHookPayLoad) {
	repositoryName, namespace := logic.ParseRepositoryNameAndNamespace(payload.Event.Target.Repository)
	repositoryModel, _ := Repository{}.GetByNameAndNamespace(repositoryName, namespace)
	if repositoryModel != nil {
		pullNum := repositoryModel.PullNum + 1
		_, err := dao.Q.RegistryRepository.Where(dao.Q.RegistryRepository.ID.Eq(repositoryModel.ID)).Update(dao.Q.RegistryRepository.PullNum, pullNum)
		if err != nil {
			slog.Error("fail to update registry repository", "namespace", namespace, "repository", repositoryName, "payload", payload, "error", err)
		}
	}
}

func (l Repository) GetRepositoryTagInfo(repositoryName string, tag string, manifest distribution.Manifest) (*TagInfo, error) {
	registryClient := logic.GetDefaultRegistryClient()

	tagType := "Docker-Image"
	platformArch := ""
	createdAt := ""
	updatedAt := ""
	size := int64(0)

	// 1. 处理 Manifest List (多架构镜像)
	manifestList, ok := manifest.(*manifestlist.DeserializedManifestList)
	if ok {
		tagType = "ManifestList"
		var sb strings.Builder
		firstManifest := false

		for _, item := range manifestList.Manifests {
			// 过滤 unknown 架构
			if item.Platform.Architecture != "" && item.Platform.Architecture != "unknown" {
				if sb.Len() > 0 {
					sb.WriteString(",")
				}
				sb.WriteString(item.Platform.Architecture)
				manifestTmp, _, errTmp := registryClient.PullManifest(repositoryName, item.Digest.String())
				if errTmp == nil && manifestTmp != nil {
					switch m := manifestTmp.(type) {
					case *ocischema.DeserializedManifest:
						size += m.Config.Size
						for _, layer := range m.Layers {
							size += layer.Size
						}
					case *schema2.DeserializedManifest:
						size += m.Config.Size
						for _, layer := range m.Layers {
							size += layer.Size
						}
					}

					// 选取第一个有效的具体 manifest 用于提取详细创建时间等信息
					if !firstManifest {
						manifest = manifestTmp // 替换为具体架构的 manifest 继续后续处理
						firstManifest = true
					}
				}
			}
		}
		platformArch = sb.String()
	}

	// 2. 统一处理：下载 Config Blob 获取详细信息 (适用于 OCI 和 Schema2)
	// 无论是 ocischema.DeserializedManifest 还是 schema2.DeserializedManifest，都有 Config.Digest
	var configDigest string
	switch m := manifest.(type) {
	case *ocischema.DeserializedManifest:
		tagType = "Oci"
		configDigest = m.Config.Digest.String()
		if size == 0 {
			size = m.Config.Size
			for _, layer := range m.Layers {
				size += layer.Size
			}
		}
		// 尝试从 Annotations 获取创建时间
		if m.Annotations != nil {
			createdAt = m.Annotations["org.opencontainers.image.created"]
			updatedAt = m.Annotations["org.opencontainers.image.updated"]
		}
	case *schema2.DeserializedManifest:
		tagType = "Docker-Image"
		configDigest = m.Config.Digest.String()
		if size == 0 {
			size = m.Config.Size
			for _, layer := range m.Layers {
				size += layer.Size
			}
		}
		// Schema V2 通常在 Manifest 里没有 created 注解，需在 Config 里找
	default:
		// 其他类型可能不支持或不需要处理
		slog.Warn("未知的 manifest 类型", "type", fmt.Sprintf("%T", manifest))
	}

	// 如果获取到了 config digest，则拉取 blob 解析 architecture 和 os
	if configDigest != "" {
		_, reader, err := registryClient.PullBlob(repositoryName, configDigest)
		if err != nil {
			return nil, err
		} else {
			defer reader.Close()
			data, err := io.ReadAll(reader)
			if err != nil {
				return nil, err
			} else {
				// 定义一个结构体来接收我们关心的字段，或者继续使用 map
				// 使用 map 比较灵活，但要注意类型断言
				configInfo := make(map[string]interface{})
				if err := json.Unmarshal(data, &configInfo); err != nil {
					return nil, err
				} else {
					// 从 Config JSON 根节点获取 architecture 和 os
					if arch, ok := configInfo["architecture"].(string); ok && arch != "" {
						// 如果之前是 ManifestList 且已经收集了架构列表，这里可以覆盖或补充单个架构信息
						// 如果之前 platformArch 为空（例如直接拉取的单架构镜像），则赋值
						if platformArch == "" {
							platformArch = arch
						}
					}
					// 获取创建时间 (如果 manifest annotations 里没有)
					if createdAt == "" {
						if created, ok := configInfo["created"].(string); ok {
							createdAt = created
						}
					}
					if updatedAt == "" {
						if updated, ok := configInfo["updated"].(string); ok {
							updatedAt = updated
						}
					}
				}
			}
		}
	}

	return &TagInfo{
		TagName:   tag,
		Type:      tagType,
		Manifest:  manifest,
		Platform:  platformArch,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		Size:      size,
	}, nil
}
