package oci

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/errdef"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
)

func NewRepositoryOci(url string, registryReference string, credential auth.Credential) (*remote.Repository, error) {
	url = strings.ReplaceAll(url, "https://", "")
	url = strings.ReplaceAll(url, "http://", "")
	url = strings.TrimRight(url, "/")
	registryReference = url + "/" + registryReference

	repo, err := remote.NewRepository(registryReference)
	if err != nil {
		if errors.Is(errors.Unwrap(err), errdef.ErrInvalidReference) {
			return nil, fmt.Errorf("%q: %v", registryReference, err)
		}
		return nil, err
	}
	repo.PlainHTTP = true

	repo.Client = &auth.Client{
		Client:     retry.DefaultClient,
		Cache:      auth.NewCache(),
		Credential: auth.StaticCredential(repo.Reference.Registry, credential),
	}

	return repo, nil
}

func GetOciDescriptorByData(data []byte, mediaType string) (*v1.Descriptor, error) {
	return &v1.Descriptor{
		MediaType: mediaType,
		Digest:    digest.FromBytes(data),
		Size:      int64(len(data)),
	}, nil
}

func GetOciDescriptorByPath(path string, mediaType string) (*v1.Descriptor, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var fi os.FileInfo
	fi, err = file.Stat()
	if err != nil {
		return nil, err
	}

	digester, err := digest.FromReader(file)
	if err != nil {
		return nil, err
	}

	return &v1.Descriptor{
		MediaType: mediaType,
		Digest:    digester,
		Size:      fi.Size(),
	}, nil
}
