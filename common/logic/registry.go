package logic

import (
	"github.com/w7panel/w7panel-zpk/common/service/oci"
	"github.com/w7panel/w7panel-zpk/common/service/registry"
	"github.com/w7panel/w7panel-zpk/common/service/registry/client"
	commontypes "github.com/w7panel/w7panel-zpk/common/service/registry/types"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry/remote/auth"
)

func GetDefaultRemoteOci(registryReference string) (oras.GraphTarget, error) {
	return oci.NewRepositoryOci(facade.GetConfig().GetString("registry_cli.default.url"), registryReference, auth.Credential{
		Username: facade.GetConfig().GetString("registry_cli.default.username"),
		Password: facade.GetConfig().GetString("registry_cli.default.password"),
	})
}

func GetDefaultRegistryClient() client.Client {
	return registry.NewRegistryClient(facade.GetConfig().GetString("registry_cli.default.url"), &commontypes.Credential{
		AccessKey:    facade.GetConfig().GetString("registry_cli.default.username"),
		AccessSecret: facade.GetConfig().GetString("registry_cli.default.password"),
	})
}
