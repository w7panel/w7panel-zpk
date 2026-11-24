package service

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"github.com/w7panel/w7panel-zpk/modules/document/common/dao"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/permission"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/storage"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/zpk"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
)

var ZPKSdk *zpk.Client
var DocumentStorage storage.Store
var PermissionSdk *permission.Client

func InitSDK(config *viper.Viper) error {
	if err := os.MkdirAll(filepath.Dir(config.GetString("database.default.db_name")), 0700); err != nil {
		return err
	}
	store, err := storage.NewLocal(config.GetString("document.storage_path"))
	if err != nil {
		return err
	}
	db, err := facade.GetDbFactory().Channel("default")
	if err != nil {
		store.Close()
		return err
	}
	dao.SetDefault(db)
	DocumentStorage = store
	PermissionSdk = permission.New(config.GetString("permission.server_url"))
	ZPKSdk = zpk.New(config.GetString("zpk.base_url"))
	return nil
}
