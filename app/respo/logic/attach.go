package logic

import (
	"github.com/w7panel/w7panel-zpk/common/service/attach"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
)

func GetLocalClient() attach.StorageClient {
	client := attach.NewStorageClient(&attach.StorageInput{
		Type:      attach.STORAGE_TYPE_LOCAL,
		Path:      facade.GetConfig().GetString("setting.depot.storage.local.path"),
		Endpoint:  facade.GetConfig().GetString("setting.depot.storage.local.endpoint"),
		SecretId:  facade.GetConfig().GetString("setting.depot.storage.s3.secret_id"),
		SecretKey: facade.GetConfig().GetString("setting.depot.storage.s3.secret_key"),
	})
	return client
}
