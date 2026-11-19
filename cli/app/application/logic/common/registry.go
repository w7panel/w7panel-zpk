package common

import (
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/w7panel/w7panel-zpk/common/function"
	"github.com/w7panel/w7panel-zpk/common/service/oci"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
)

// Target identifies a tagged artifact without applying a particular artifact
// type's naming or namespace conventions.
type Target struct {
	Registry   string
	Repository string
	Identifie  string
	Version    string
	PlainHTTP  bool
}

func (target Target) Reference() string { return target.Registry + "/" + target.Repository }

// ResolveTarget accepts a short repository name or a full registry reference.
// Each caller supplies its default namespace and applies any type-specific
// identifier conversion after parsing.
func ResolveTarget(reference, defaultHost, namespace string, plainHTTP bool) (Target, error) {
	reference = strings.TrimSpace(reference)
	target := Target{PlainHTTP: plainHTTP}
	if strings.Contains(reference, "://") {
		parsed, err := url.Parse(reference)
		if err != nil {
			return target, err
		}
		if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
			return target, fmt.Errorf("仓库地址必须是不含身份或查询参数的 HTTP(S) 地址")
		}
		reference = parsed.Host + parsed.Path
		target.PlainHTTP = parsed.Scheme == "http"
	}
	if !function.LooksLikeRegistryReference(reference) {
		if defaultHost == "" {
			return target, fmt.Errorf("请通过 --host 指定默认仓库，或先执行 zpk login")
		}
		prefix := defaultHost + "/"
		if namespace != "" {
			prefix += namespace + "/"
		}
		reference = prefix + reference
	}
	parsed, err := registry.ParseReference(reference)
	if err != nil {
		return target, err
	}
	if strings.Contains(parsed.Reference, ":") {
		return target, fmt.Errorf("请使用版本 tag，暂不接受 digest 引用")
	}
	target.Registry, target.Repository, target.Version = parsed.Registry, parsed.Repository, parsed.Reference
	target.Identifie = path.Base(parsed.Repository)
	return target, nil
}

func NewRepository(target Target, session Session) (*remote.Repository, error) {
	credential, err := session.RegistryCredential(target.Registry)
	if err != nil {
		return nil, err
	}
	repository, err := oci.NewRepositoryOciReference(target.Reference(), credential)
	if err != nil {
		return nil, err
	}
	repository.PlainHTTP = target.PlainHTTP
	return repository, nil
}

// RegistryCredential never forwards the default registry's credentials to a
// different host. Legacy single-registry sessions remain readable.
func (session Session) RegistryCredential(host string) (auth.Credential, error) {
	saved, ok := session.Credentials[host]
	if !ok && host == session.Host {
		saved = RegistryCredential{Username: session.Username, Password: session.Password}
	}
	if saved.Username == "" || saved.Password == "" {
		return auth.Credential{}, nil
	}
	password, err := DecryptPassword(saved.Password)
	if err != nil {
		return auth.Credential{}, err
	}
	return auth.Credential{Username: saved.Username, Password: password}, nil
}
