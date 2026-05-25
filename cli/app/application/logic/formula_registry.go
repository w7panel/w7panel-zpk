package logic

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/w7panel/w7panel-zpk/common/function"
	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/w7panel/w7panel-zpk/common/service/oci"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
)

type FormulaRegistryInfo struct {
	Artifact      string
	OciRegistry   string
	OciRepository string
	OciTag        string
}

func (target FormulaRegistryInfo) Reference() string {
	if target.OciRegistry == "" || target.OciRepository == "" {
		return target.Artifact
	}
	reference := target.OciRegistry + "/" + target.OciRepository
	if target.OciTag != "" {
		reference += ":" + target.OciTag
	}
	return reference
}

func ParseFormulaRegistry(name string) (FormulaRegistryInfo, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return FormulaRegistryInfo{}, fmt.Errorf("artifact name is required")
	}

	if function.LooksLikeRegistryReference(name) {
		ref, err := registry.ParseReference(name)
		if err != nil {
			return FormulaRegistryInfo{}, err
		}
		artifact := strings.TrimSpace(path.Base(ref.Repository))
		if artifact == "." || artifact == "/" {
			return FormulaRegistryInfo{}, fmt.Errorf("invalid artifact repository %q", ref.Repository)
		}
		if strings.Contains(ref.Reference, ":") {
			return FormulaRegistryInfo{}, fmt.Errorf("OCI image reference must use a tag, not digest")
		}
		return FormulaRegistryInfo{
			Artifact:      strings.ReplaceAll(artifact, "_", "-"),
			OciRegistry:   ref.Registry,
			OciRepository: strings.ReplaceAll(ref.Repository, artifact, strings.ReplaceAll(artifact, "-", "_")),
			OciTag:        ref.Reference,
		}, nil
	}

	return FormulaRegistryInfo{Artifact: strings.ReplaceAll(name, "_", "-")}, nil
}

func NewRemoteRepository(session Session, credential auth.Credential) (*remote.Repository, error) {
	if session.OciRegistry != "" && session.OciRepository != "" {
		return oci.NewRepositoryOciReference(session.OciRegistry+"/"+session.OciRepository, credential)
	}
	return oci.NewRepositoryOci(session.Host, commonlogic.GetFormulaOciName(session.Artifact), credential)
}

func ListRemoteTags(session Session) ([]string, error) {
	password, err := DecryptPassword(session.Password)
	if err != nil {
		return nil, err
	}
	remoteRepository, err := NewRemoteRepository(session, auth.Credential{
		Username: session.Username,
		Password: password,
	})
	if err != nil {
		return nil, err
	}

	tags := make([]string, 0)
	if err := remoteRepository.Tags(context.Background(), "", func(page []string) error {
		tags = append(tags, page...)
		return nil
	}); err != nil {
		return nil, err
	}
	function.SortTagsDesc(tags)
	return tags, nil
}
