package service

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/permission"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/storage"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"gorm.io/gorm"
)

type SDK struct {
	DB          *gorm.DB
	Storage     *storage.Local
	Permissions *permission.Client
}

func InitSDK(config *viper.Viper) (*SDK, error) {
	if err := os.MkdirAll(filepath.Dir(config.GetString("database.default.db_name")), 0700); err != nil {
		return nil, err
	}
	store, err := storage.NewLocal(config.GetString("document.storage_path"))
	if err != nil {
		return nil, err
	}
	db, err := facade.GetDbFactory().Channel("default")
	if err != nil {
		store.Close()
		return nil, err
	}
	return &SDK{
		DB:          db,
		Storage:     store,
		Permissions: permission.New(config.GetString("permission.base_url")),
	}, nil
}

func (s *SDK) Close() error {
	if s.Storage == nil {
		return nil
	}
	err := s.Storage.Close()
	s.Storage = nil
	return err
}
