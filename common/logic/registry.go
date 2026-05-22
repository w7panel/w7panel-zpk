package logic

import (
	"strings"

	"github.com/w7panel/w7panel-zpk/common/service/oci"
	"github.com/w7panel/w7panel-zpk/common/service/registry"
	"github.com/w7panel/w7panel-zpk/common/service/registry/client"
	commontypes "github.com/w7panel/w7panel-zpk/common/service/registry/types"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
)

var DefaultNamespace = "default"

func ParseRepositoryNameAndNamespace(repositoryName string) (string, string) {
	nameArr := strings.SplitN(repositoryName, "/", 2)
	targetRepositoryName := repositoryName
	namespace := DefaultNamespace
	if len(nameArr) >= 2 {
		namespace = nameArr[0]
		targetRepositoryName = nameArr[1]
	}
	return targetRepositoryName, namespace
}

func BuildRepositoryName(repositoryName string, namespace string) string {
	if namespace != DefaultNamespace {
		repositoryName = namespace + "/" + repositoryName
	}
	return repositoryName
}

func GetDefaultRemoteOci(registryReference string) (*remote.Repository, error) {
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
