package service

import (
	"os"
	"path/filepath"

	"github.com/gin-contrib/sessions"
	"github.com/spf13/viper"
	"github.com/w7panel/w7panel-zpk/modules/document/common/dao"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/storage"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/zpk"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	ranginesession "github.com/we7coreteam/w7-rangine-go/v2/src/http/session"
)

var ZPKSdk *zpk.Client
var DocumentStorage *storage.Local
var SessionStore sessions.Store

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
	SessionStore = ranginesession.GetGormStore(config, []byte(config.GetString("session.secret")))
	ZPKSdk = zpk.New(config.GetString("zpk.base_url"))
	return nil
}
