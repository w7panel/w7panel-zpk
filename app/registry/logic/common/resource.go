package common

import (
	"context"
	"errors"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	userlogic "github.com/w7panel/w7panel-zpk/common/logic"
	"gorm.io/gorm"
)

type Resource struct{}

func resourceDB(ctx context.Context, query *dao.Query) *gorm.DB {
	return query.RegistryRepository.WithContext(ctx).UnderlyingDB().Session(&gorm.Session{NewDB: true}).WithContext(ctx)
}
func resourceKey(resource ResourcePermissionResource) entity.RegistryRepository {
	name, namespace := userlogic.ParseRepositoryNameAndNamespace(resource.Value)
	if resource.Type == string(PermissionResourceTypeNamespace) {
		name = resource.Value
		namespace = ""
	}
	return entity.RegistryRepository{ResourceType: resource.Type, Name: name, Namespace: namespace}
}
func (Resource) Get(ctx context.Context, resource ResourcePermissionResource) (*entity.RegistryRepository, error) {
	var row entity.RegistryRepository
	err := resourceDB(ctx, dao.Q).Where(resourceKey(resource)).First(&row).Error
	return &row, err
}

// Set changes resource visibility without replacing user grants or ownership.
func (Resource) Set(ctx context.Context, resource ResourcePermissionResource, userID int32, visibleType int32) error {
	return dao.Q.Transaction(func(tx *dao.Query) error {
		db := resourceDB(ctx, tx)
		row := resourceKey(resource)
		err := db.Where(row).First(&row).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			row.UserID = userID
			row.VisibleType = visibleType
			return db.Create(&row).Error
		}
		return db.Model(&row).Select("VisibleType").Updates(entity.RegistryRepository{VisibleType: visibleType}).Error
	})
}
func (Resource) Delete(ctx context.Context, resource ResourcePermissionResource) error {
	return dao.Q.Transaction(func(tx *dao.Query) error {
		if err := resourceDB(ctx, tx).Where(resourceKey(resource)).Delete(&entity.RegistryRepository{}).Error; err != nil {
			return err
		}
		return (ResourcePermission{}).DelPermissionByResource(tx, resource.Value, PermissionResourceType(resource.Type))
	})
}

// Prepare resolves public inheritance. User grant inheritance stays in CheckResource.
func (l Resource) Prepare(ctx context.Context, resource ResourcePermissionResource) (ResourcePermissionResource, error) {
	row, err := l.Get(ctx, resource)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return resource, err
	}
	if err == nil && row.VisibleType != VisibleTypeFollowNamespace {
		resource.PublicActions = PublicActionsForVisibleType(int(row.VisibleType))
		return resource, nil
	}
	_, namespace := userlogic.ParseRepositoryNameAndNamespace(resource.Value)
	if resource.Type == string(PermissionResourceTypeNamespace) {
		namespace = resource.Value
	}
	space, err := (Namespace{}).GetByName(namespace)
	if err != nil {
		return resource, err
	}
	resource.PublicActions = PublicActionsForVisibleType(int(space.VisibleType))
	return resource, nil
}
func (l Resource) Check(ctx context.Context, user *entity.RegistryUser, resource ResourcePermissionResource, actions []string) (ResourcePermissionDecision, error) {
	prepared, err := l.Prepare(ctx, resource)
	if err != nil {
		return ResourcePermissionDecision{}, err
	}
	return (ResourcePermission{}).CheckResource(ctx, user, prepared, actions)
}
func renameResources(ctx context.Context, tx *dao.Query, oldName, newName string) error {
	db := resourceDB(ctx, tx)
	var rows []entity.RegistryRepository
	if err := db.Where(&entity.RegistryRepository{Namespace: oldName}).Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if err := db.Model(&row).Select("Namespace").Updates(entity.RegistryRepository{Namespace: newName}).Error; err != nil {
			return err
		}
	}
	return nil
}
func hasResources(ctx context.Context, tx *dao.Query, namespace string) (bool, error) {
	var count int64
	err := resourceDB(ctx, tx).Model(&entity.RegistryRepository{}).Where(&entity.RegistryRepository{Namespace: namespace}).Count(&count).Error
	return count > 0, err
}

func (l Resource) GetByID(ctx context.Context, id int32, resourceType string, user *entity.RegistryUser, ownerOnly bool) (*entity.RegistryRepository, error) {
	query := dao.Q.RegistryRepository
	row, err := query.WithContext(ctx).Where(query.ID.Eq(id), query.ResourceType.Eq(resourceType)).First()
	if err != nil {
		return nil, err
	}
	if user == nil || (userlogic.User{}).IsAdminUser(user) || row.UserID == user.ID {
		return row, nil
	}
	if !ownerOnly && (ResourcePermission{}).UserHasPermissionByResource(int(user.ID), row.Namespace, PermissionResourceTypeNamespace) {
		return row, nil
	}
	return nil, errors.New("user no permission")
}
func (Resource) Save(ctx context.Context, row *entity.RegistryRepository, oldValue string) error {
	return dao.Q.Transaction(func(tx *dao.Query) error {
		if err := tx.RegistryRepository.WithContext(ctx).Save(row); err != nil {
			return err
		}
		newValue := userlogic.BuildRepositoryName(row.Name, row.Namespace)
		if oldValue != "" && oldValue != newValue {
			grants := tx.RegistryUserPermission
			_, err := grants.WithContext(ctx).Where(grants.ResourceType.Eq(row.ResourceType), grants.ResourceValue.Eq(oldValue)).Update(grants.ResourceValue, newValue)
			return err
		}
		return nil
	})
}

func (Resource) GetByNameAndNamespace(ctx context.Context, resourceType, name, namespace string) (*entity.RegistryRepository, error) {
	query := dao.Q.RegistryRepository
	return query.WithContext(ctx).Where(query.ResourceType.Eq(resourceType), query.Name.Eq(name), query.Namespace.Eq(namespace)).First()
}
