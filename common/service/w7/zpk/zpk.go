package zpk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/w7panel/w7panel-zpk/common/service/w7/base"
)

type ZpkService struct {
	base.Base
}

type BaseInfoResp struct {
	LatestVersion string `json:"latest_version"`
}

type FormulaInfoResp struct {
	Manifest       string            `json:"manifest,omitempty"`
	ChildManifests map[string]string `json:"child_manifests,omitempty"`
	ZipURL         string            `json:"zip_url,omitempty"`
	ZipURLs        map[string]string `json:"zip_urls,omitempty"`
	WebZipURL      map[string]string `json:"webzip_url,omitempty"`
	HelmURL        string            `json:"helm_url"`
	HelmURLs       map[string]string `json:"helm_urls,omitempty"`
	FormulaType    string            `json:"formula_type"`
}

// RemoteFormulaInfoURL accepts either a complete formula-info endpoint or a
// ZPK service base URL and returns the canonical endpoint for a formula
// identifier. A path that already contains an identifier is authoritative;
// the collection endpoint itself still receives the requested identifier.
func RemoteFormulaInfoURL(from, formulaIdentifie string) (string, error) {
	from = strings.TrimSpace(from)
	if from == "" {
		return "", fmt.Errorf("来源 URL 不能为空")
	}
	if formulaIdentifie == "" || formulaIdentifie == "." || formulaIdentifie == ".." || strings.ContainsAny(formulaIdentifie, `/\\`) {
		return "", fmt.Errorf("制品标识无效: %q", formulaIdentifie)
	}

	parsed, err := url.Parse(from)
	if err != nil {
		return "", fmt.Errorf("解析来源 URL 失败: %w", err)
	}
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", fmt.Errorf("来源 URL 必须包含协议和主机")
	}

	const infoPath = "/zpk/respo/info"
	path := strings.TrimRight(parsed.Path, "/")
	// `/zpk/respo/info/<identifie>` is already a complete endpoint.  The
	// collection endpoint `/zpk/respo/info` still needs the identifier appended
	// just like a service root does.
	if strings.HasPrefix(path, infoPath+"/") && strings.TrimPrefix(path, infoPath+"/") != "" {
		return parsed.String(), nil
	}
	if path == "" {
		path = infoPath
	} else if path == "/zpk" {
		path += "/respo/info"
	} else {
		path += infoPath
	}
	parsed.Path = path + "/" + formulaIdentifie
	parsed.RawPath = ""
	return parsed.String(), nil
}

// WithCompleteManifest requests the extended info representation without
// disturbing query parameters (for example order_sn) already present on the
// endpoint.
func WithCompleteManifest(infoURL string) (string, error) {
	infoURL = strings.TrimSpace(infoURL)
	if infoURL == "" {
		return "", fmt.Errorf("来源 URL 不能为空")
	}
	parsed, err := url.Parse(infoURL)
	if err != nil {
		return "", fmt.Errorf("解析来源 URL 失败: %w", err)
	}
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", fmt.Errorf("来源 URL 必须包含协议和主机")
	}
	query := parsed.Query()
	query.Set("full_manifest", "1")
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func (s ZpkService) GetRemoteFormulaInfo(ctx context.Context, infoURL string) (*FormulaInfoResp, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, infoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		var apiError base.ApiError
		if err := json.Unmarshal(body, &apiError); err == nil && apiError.ErrorMsg != "" {
			return nil, apiError
		}
		return nil, fmt.Errorf("获取制品信息返回 HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var result struct {
		Code  int             `json:"code"`
		Error string          `json:"error"`
		Data  FormulaInfoResp `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	if result.Code != 0 && result.Code != http.StatusOK {
		return nil, base.ApiError{Code: result.Code, ErrorMsg: result.Error}
	}
	// A complete info response can describe applications that are not Helm
	// applications (for example docker or tradition applications).  Do not
	// reject those responses merely because they have no Helm archive; callers
	// that specifically need a chart validate the corresponding URL themselves.
	return &result.Data, nil
}

func (s ZpkService) GetRemoteFormulaBaseInfo(formulaName string) (*BaseInfoResp, error) {
	infoUrl := s.BaseUrl + "/zpk/respo/open-api/formula/base-info"

	params := map[string]string{
		"identifie": formulaName,
	}
	paramsJson, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	req, err := http.NewRequest("POST", infoUrl, bytes.NewReader(paramsJson))
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-requested-with", "XMLHttpRequest")
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		var apiError base.ApiError
		err = json.Unmarshal(respBody, &apiError)
		if err != nil {
			return nil, err
		}
		return nil, apiError
	}

	type wrappedBaseInfoResp struct {
		Data BaseInfoResp `json:"data"`
	}
	var wrappedResp wrappedBaseInfoResp
	if err := json.Unmarshal(respBody, &wrappedResp); err == nil && wrappedResp.Data.LatestVersion != "" {
		return &wrappedResp.Data, nil
	}

	ret := BaseInfoResp{}
	err = json.Unmarshal(respBody, &ret)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}
