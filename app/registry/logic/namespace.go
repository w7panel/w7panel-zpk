package logic

import (
	"errors"
	"slices"

	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	"github.com/w7panel/w7panel-zpk/common/logic"
	"gorm.io/gen"
)

var (
	VisibleTypePrivate         = 1
	VisibleTypePublicWrite     = 2
	VisibleTypePublicRead      = 4
	VisibleTypeFollowNamespace = 3
)

var DefaultNamespace = "default"

type Namespace struct {
	logic.Logic
}

func (l Namespace) GetByIdWithUser(id int, user *entity.RegistryUser) (*entity.RegistryNamespace, error) {
	query := dao.Q.RegistryNamespace.Where(dao.RegistryNamespace.ID.Eq(int32(id)))
	if user != nil && !(logic.User{}.IsAdminUser(user)) {
		query = query.Where(dao.RegistryNamespace.UserID.Eq(user.ID))
	}
	return query.First()
}

func (l Namespace) GetByIdWithUserPermission(id int, user *entity.RegistryUser) (*entity.RegistryNamespace, error) {
	model, _ := dao.Q.RegistryNamespace.Where(dao.RegistryNamespace.ID.Eq(int32(id))).First()
	if model == nil {
		return nil, nil
	}
	if user != nil && !(logic.User{}.IsAdminUser(user)) && model.UserID != user.ID {
		exists := Permission{}.UserHasPermissionByResource(int(user.ID), model.Name, PermissionResourceTypeNamespace)
		if !exists {
			return nil, errors.New("user no permission")
		}
	}

	return model, nil
}

func (l Namespace) GetByName(name string) (*entity.RegistryNamespace, error) {
	return dao.Q.RegistryNamespace.Where(dao.RegistryNamespace.Name.Eq(name)).First()
}

func (l Namespace) ListSubNamespacesByNamespace(namespace string) ([]string, error) {
	subNamespaceMap := make(map[string]struct{})
	batchRows := make([]*entity.RegistryRepository, 0, 200)
	err := dao.Q.RegistryRepository.
		Select(dao.Q.RegistryRepository.Name, dao.Q.RegistryRepository.ID).
		Where(dao.Q.RegistryRepository.Namespace.Eq(namespace)).
		Where(dao.Q.RegistryRepository.Name.Like("%/%")).
		Order(dao.Q.RegistryRepository.ID.Asc()).
		FindInBatches(&batchRows, 200, func(tx gen.Dao, batch int) error {
			for _, repository := range batchRows {
				_, subNamespace := Repository{}.ParseRepositoryNameAndNamespace(repository.Name)
				if subNamespace == "" || subNamespace == DefaultNamespace {
					continue
				}
				subNamespaceMap[subNamespace] = struct{}{}
			}
			return nil
		})
	if err != nil {
		return nil, err
	}

	subNamespaces := make([]string, 0, len(subNamespaceMap))
	for subNamespace := range subNamespaceMap {
		subNamespaces = append(subNamespaces, subNamespace)
	}
	slices.Sort(subNamespaces)

	return subNamespaces, nil
}
