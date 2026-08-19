package logic

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	"github.com/w7panel/w7panel-zpk/common/function"
	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
)

const backendZipTokenVersion = "v1"

const backendZipTokenMACContext = "w7panel-zpk:backend-zip:v1:"

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
	encodedCiphertext := base64.RawURLEncoding.EncodeToString(ciphertext)
	signature := backendZipTokenSignature(encodedCiphertext)
	return strings.Join([]string{backendZipTokenVersion, encodedCiphertext, signature}, "."), nil
}

func ParseBackendZipDownloadToken(token string) (*BackendZipDownloadToken, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] != backendZipTokenVersion {
		return nil, errors.New("invalid backend zip token version")
	}
	providedSignature, err := hex.DecodeString(parts[2])
	if err != nil {
		return nil, errors.New("invalid backend zip token signature")
	}
	expectedSignature, _ := hex.DecodeString(backendZipTokenSignature(parts[1]))
	if !hmac.Equal(providedSignature, expectedSignature) {
		return nil, errors.New("invalid backend zip token signature")
	}
	ciphertext, err := base64.RawURLEncoding.DecodeString(parts[1])
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

func backendZipTokenSignature(ciphertext string) string {
	secret := sha256.Sum256([]byte(backendZipTokenMACContext + facade.GetConfig().GetString("setting.secret")))
	signer := hmac.New(sha256.New, secret[:])
	_, _ = signer.Write([]byte(ciphertext))
	return hex.EncodeToString(signer.Sum(nil))
}
