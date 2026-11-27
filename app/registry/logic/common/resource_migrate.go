package common

import (
	"errors"
	"github.com/w7panel/w7panel-zpk/common/entity"
	userlogic "github.com/w7panel/w7panel-zpk/common/logic"
	"gorm.io/gorm"
)

// MigrateLegacy backfills the existing table once per row, without overriding new settings.
func (Resource) MigrateLegacy(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var grants []entity.RegistryUserPermission
		if err := tx.Where(&entity.RegistryUserPermission{ResourceType: "document_library"}).Order("id").Find(&grants).Error; err != nil {
			return err
		}
		resources := map[string]*entity.RegistryRepository{}
		for _, grant := range grants {
			row := resources[grant.ResourceValue]
			if row == nil {
				name, namespace := userlogic.ParseRepositoryNameAndNamespace(grant.ResourceValue)
				row = &entity.RegistryRepository{ResourceType: grant.ResourceType, Name: name, Namespace: namespace, VisibleType: VisibleTypePrivate}
				resources[grant.ResourceValue] = row
			}
			if grant.UserID == 0 && grant.Action != nil {
				if ValidatePermissionActions([]string{"push"}, []string(*grant.Action)) {
					row.VisibleType = VisibleTypePublicWrite
				} else if ValidatePermissionActions([]string{"pull"}, []string(*grant.Action)) {
					row.VisibleType = VisibleTypePublicRead
				}
			}
			if row.UserID == 0 && grant.UserID > 0 && grant.Action != nil && ValidatePermissionActions([]string{"push"}, []string(*grant.Action)) {
				row.UserID = grant.UserID
			}
		}
		for _, row := range resources {
			var current entity.RegistryRepository
			err := tx.Where(&entity.RegistryRepository{ResourceType: row.ResourceType, Namespace: row.Namespace, Name: row.Name}).First(&current).Error
			if err == nil {
				continue
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if err := tx.Create(row).Error; err != nil {
				return err
			}
		}
		return tx.Where("resource_type = ? AND user_id = ?", "document_library", 0).Delete(&entity.RegistryUserPermission{}).Error
	})
}
