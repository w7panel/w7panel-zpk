package registry

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/docker/distribution/registry/client/auth/challenge"
	registryhttp "github.com/w7panel/w7panel-zpk/common/service/registry/http"
	"oras.land/oras-go/v2/registry/remote"

	"github.com/w7panel/w7panel-zpk/common/service/registry/client"
	"github.com/w7panel/w7panel-zpk/common/service/registry/types"
)

func NewRegistryClient(url string, credential *types.Credential) client.Client {
	return client.NewClient(url, credential.AccessKey, credential.AccessSecret, true)
}

// CheckPull lets the Registry validate repository pull access and returns its
// authentication challenge when a token is missing, invalid or lacks pull access.
func CheckPull(ctx context.Context, serverURL, repository, token string) (bool, string, error) {
	server, err := url.Parse(serverURL)
	if err != nil || server.Host == "" || (server.Scheme != "http" && server.Scheme != "https") || server.User != nil || server.RawQuery != "" || server.Fragment != "" {
		return false, "", errors.New("Registry 地址无效")
	}
	repo, err := remote.NewRepository(server.Host + "/" + repository)
	if err != nil || repo.Reference.Repository != repository || repo.Reference.Reference != "" {
		return false, "", errors.New("Registry 仓库路径无效")
	}
	server.Path = strings.TrimRight(server.Path, "/") + "/v2/" + repository + "/tags/list"
	server.RawPath = ""
	server.RawQuery = "n=1"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, server.String(), nil)
	if err != nil {
		return false, "", err
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	client := &http.Client{
		Timeout:       10 * time.Second,
		Transport:     registryhttp.GetHTTPTransport(),
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
	}
	response, err := client.Do(request)
	if err != nil {
		return false, "", fmt.Errorf("获取 Registry 认证信息失败: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		return true, "", nil
	}
	if response.StatusCode != http.StatusUnauthorized {
		return false, "", fmt.Errorf("Registry 未返回认证挑战，状态码 %d", response.StatusCode)
	}
	for _, raw := range response.Header.Values("WWW-Authenticate") {
		probe := &http.Response{StatusCode: http.StatusUnauthorized, Header: http.Header{"Www-Authenticate": []string{raw}}}
		parsed := challenge.ResponseChallenges(probe)
		if len(parsed) != 1 || parsed[0].Scheme != "bearer" {
			continue
		}
		params := parsed[0].Parameters
		realm, err := url.Parse(params["realm"])
		if err != nil || realm.Host == "" || (realm.Scheme != "http" && realm.Scheme != "https") || realm.User != nil || realm.Fragment != "" {
			continue
		}
		if params["service"] == "" || params["scope"] != "repository:"+repository+":pull" {
			continue
		}
		return false, raw, nil
	}
	return false, "", errors.New("Registry 未返回有效的 Bearer pull 认证挑战")
}
