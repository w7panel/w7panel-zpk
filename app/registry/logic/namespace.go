package logic

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	"github.com/w7panel/w7panel-zpk/common/logic"
	"gorm.io/gen"
	"gorm.io/gorm"
)

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
		exists := ResourcePermission{}.UserHasPermissionByResource(int(user.ID), model.Name, PermissionResourceTypeNamespace)
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
				_, subNamespace := logic.ParseRepositoryNameAndNamespace(repository.Name)
				if subNamespace == "" || subNamespace == logic.DefaultNamespace {
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

var ErrNamespaceConflict = errors.New("命名空间已存在或仍有资源")
var ErrNamespaceProtected = errors.New("默认命名空间不可修改或删除")

// CreateShared creates system-owned namespaces without a user login dependency.
func (l Namespace) CreateShared(ctx context.Context, name string, visibleType int32, visible *bool) error {
	query := dao.Q.RegistryNamespace
	if _, err := query.WithContext(ctx).Unscoped().Where(query.Name.Eq(name)).First(); err == nil {
		return ErrNamespaceConflict
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return dao.Q.Transaction(func(tx *dao.Query) error {
		namespace := &entity.RegistryNamespace{Name: name, VisibleType: visibleType}
		query := tx.RegistryNamespace
		if err := query.WithContext(ctx).Create(namespace); err != nil {
			return err
		}
		if visible != nil && !*visible {
			_, err := query.WithContext(ctx).Where(query.ID.Eq(namespace.ID)).Update(query.Visible, false)
			return err
		}
		return nil
	})
}

func (l Namespace) RenameShared(ctx context.Context, id int32, name string) error {
	return dao.Q.Transaction(func(tx *dao.Query) error {
		query := tx.RegistryNamespace
		namespace, err := query.WithContext(ctx).Where(query.ID.Eq(id)).First()
		if err != nil || namespace.Name == name {
			return err
		}
		if namespace.Name == logic.DefaultNamespace {
			return ErrNamespaceProtected
		}
		if count, err := query.WithContext(ctx).Unscoped().Where(query.Name.Eq(name)).Count(); err != nil {
			return err
		} else if count != 0 {
			return ErrNamespaceConflict
		}
		if err := l.renamePermissions(ctx, tx, namespace.Name, name); err != nil {
			return err
		}
		repository := tx.RegistryRepository
		if _, err := repository.WithContext(ctx).Where(repository.Namespace.Eq(namespace.Name)).Update(repository.Namespace, name); err != nil {
			return err
		}
		_, err = query.WithContext(ctx).Where(query.ID.Eq(id)).Update(query.Name, name)
		return err
	})
}

func (l Namespace) DeleteShared(ctx context.Context, id int32) error {
	return dao.Q.Transaction(func(tx *dao.Query) error {
		query := tx.RegistryNamespace
		namespace, err := query.WithContext(ctx).Where(query.ID.Eq(id)).First()
		if err != nil {
			return err
		}
		if namespace.Name == logic.DefaultNamespace {
			return ErrNamespaceProtected
		}
		repository := tx.RegistryRepository
		if count, err := repository.WithContext(ctx).Where(repository.Namespace.Eq(namespace.Name)).Count(); err != nil {
			return err
		} else if count != 0 {
			return ErrNamespaceConflict
		}
		if used, err := l.hasResourcePermissions(ctx, tx, namespace.Name); err != nil {
			return err
		} else if used {
			return ErrNamespaceConflict
		}
		if _, err := query.WithContext(ctx).Where(query.ID.Eq(id)).Delete(); err != nil {
			return err
		}
		permissions := tx.RegistryUserPermission
		_, err = permissions.WithContext(ctx).Where(permissions.ResourceType.Eq("namespace"), permissions.ResourceValue.Eq(namespace.Name)).Delete()
		return err
	})
}

func (l Namespace) renamePermissions(ctx context.Context, tx *dao.Query, oldName, newName string) error {
	query := tx.RegistryUserPermission
	if _, err := query.WithContext(ctx).Where(query.ResourceType.Eq("namespace"), query.ResourceValue.Eq(oldName)).Update(query.ResourceValue, newName); err != nil {
		return err
	}
	rows, err := query.WithContext(ctx).Where(query.ResourceType.Neq("namespace"), query.ResourceValue.Like(oldName+"/%")).Find()
	if err != nil {
		return err
	}
	for _, row := range rows {
		if strings.HasPrefix(row.ResourceValue, oldName+"/") {
			value := newName + strings.TrimPrefix(row.ResourceValue, oldName)
			if _, err := query.WithContext(ctx).Where(query.ID.Eq(row.ID)).Update(query.ResourceValue, value); err != nil {
				return err
			}
		}
	}
	return nil
}

func (l Namespace) hasResourcePermissions(ctx context.Context, tx *dao.Query, name string) (bool, error) {
	query := tx.RegistryUserPermission
	rows, err := query.WithContext(ctx).Where(query.ResourceType.Neq("namespace"), query.ResourceValue.Like(name+"/%")).Find()
	if err != nil {
		return false, err
	}
	for _, row := range rows {
		if strings.HasPrefix(row.ResourceValue, name+"/") {
			return true, nil
		}
	}
	return false, nil
}
