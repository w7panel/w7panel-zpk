package logic

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"

	"github.com/w7panel/w7panel-zpk/common/function"
	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
)

type BackendZipDownloadToken struct {
	ZipPath   string `json:"zip_path"`
	Identifie string `json:"identifie"`
	Version   string `json:"version"`
}

func createBackendZipDownloadToken(application logic2.Application, zipPath string) (string, error) {
	if strings.TrimSpace(zipPath) == "" {
		return "", errors.New("backend zip path is empty")
	}
	payload, err := json.Marshal(BackendZipDownloadToken{
		ZipPath: zipPath, Identifie: application.Identifie, Version: application.Version,
	})
	if err != nil {
		return "", err
	}
	encrypted, err := function.AesEncrypt(string(payload), backendZipTokenEncryptionKey())
	if err != nil {
		return "", err
	}
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}
	// Keep the public token compact: the encrypted payload (including its
	// random IV) is enough for the download endpoint and is URL-safe encoded
	// without the standard Base64 padding.
	return base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

func ParseBackendZipDownloadToken(token string) (*BackendZipDownloadToken, error) {
	ciphertext, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return nil, errors.New("invalid backend zip token encoding")
	}
	payload, err := function.AesDecrypt(base64.StdEncoding.EncodeToString(ciphertext), backendZipTokenEncryptionKey())
	if err != nil {
		return nil, errors.New("invalid backend zip token payload")
	}
	result := &BackendZipDownloadToken{}
	if err = json.Unmarshal([]byte(payload), result); err != nil {
		return nil, errors.New("invalid backend zip token payload")
	}
	if strings.TrimSpace(result.ZipPath) == "" || result.Identifie == "" || result.Version == "" {
		return nil, errors.New("incomplete backend zip token payload")
	}
	return result, nil
}

func backendZipTokenEncryptionKey() string {
	return function.GetMd5(facade.GetConfig().GetString("setting.secret"))
}
