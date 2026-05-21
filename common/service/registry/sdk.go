package registry

import (
	"github.com/w7panel/w7panel-zpk/common/service/registry/client"
	"github.com/w7panel/w7panel-zpk/common/service/registry/types"
)

func NewRegistryClient(url string, credential *types.Credential) client.Client {
	return client.NewClient(url, credential.AccessKey, credential.AccessSecret, true)
}
