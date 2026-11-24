package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
	"github.com/w7panel/w7panel-zpk/modules/document/common/dao"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/app"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/permission"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/storage"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/zpk"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
)

var ZPKSdk *zpk.Client
var DocumentStorage storage.Store
var PermissionSdk *permission.Client
var AppSdk *app.Client

type appCredential struct {
	Appid  string
	Secret string
}

func InitSDK(config *viper.Viper) error {
	credential, err := fetchAppCredential(context.Background())
	if err != nil {
		return err
	}
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
	signer := app.New(credential.Appid, credential.Secret)
	AppSdk = signer
	PermissionSdk = permission.New(config.GetString("permission.server_url"), signer)
	ZPKSdk = zpk.New(config.GetString("zpk.base_url"), signer)
	return nil
}

func fetchAppCredential(ctx context.Context) (appCredential, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://api.w7.cc/api/app/info", nil)
	if err != nil {
		return appCredential{}, err
	}
	response, err := (&http.Client{Timeout: 10 * time.Second}).Do(request)
	if err != nil {
		return appCredential{}, fmt.Errorf("获取应用签名凭据失败: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return appCredential{}, errors.New("获取应用签名凭据失败")
	}
	var result struct {
		Code  int    `json:"code"`
		Error string `json:"error"`
		Data  struct {
			AppID  string `json:"appid"`
			Secret string `json:"appsecret"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&result); err != nil {
		return appCredential{}, errors.New("应用签名凭据响应无效")
	}
	if (result.Code != 0 && result.Code != 200) || result.Error != "" ||
		strings.TrimSpace(result.Data.AppID) == "" || strings.TrimSpace(result.Data.Secret) == "" {
		return appCredential{}, errors.New("应用签名凭据响应无效")
	}
	return appCredential{Appid: strings.TrimSpace(result.Data.AppID), Secret: strings.TrimSpace(result.Data.Secret)}, nil
}
