package attach

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/w7panel/w7panel-zpk/common/function"
	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/w7panel/w7panel-zpk/common/service"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
)

var zipCacheLocks sync.Map
var zipCacheCleanupMu sync.Mutex
var zipCacheLastCleanup time.Time
var zipCacheCleanupRunning bool

const (
	zipCacheTTL             = 24 * time.Hour
	zipCacheCleanupInterval = time.Hour
	zipCacheTmpTTL          = time.Hour
)

type zipCacheLock struct {
	mu sync.Mutex
}

type PermanentAttachmentDownloadToken struct {
	Path      string `json:"zip_path"`
	Identifie string `json:"identifie"`
	Version   string `json:"version"`
}

// CreatePermanentAttachmentDownloadToken creates a permanent token for downloading a formula attachment.
func CreatePermanentAttachmentDownloadToken(application logic2.Application, attachmentPath string) (string, error) {
	if strings.TrimSpace(attachmentPath) == "" {
		return "", errors.New("attachment path is empty")
	}
	payload, err := json.Marshal(PermanentAttachmentDownloadToken{
		Path: attachmentPath, Identifie: application.Identifie, Version: application.Version,
	})
	if err != nil {
		return "", err
	}
	encrypted, err := function.AesEncrypt(string(payload), permanentAttachmentDownloadTokenEncryptionKey())
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

func ParsePermanentAttachmentDownloadToken(token string) (*PermanentAttachmentDownloadToken, error) {
	ciphertext, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return nil, errors.New("invalid permanent attachment token encoding")
	}
	payload, err := function.AesDecrypt(base64.StdEncoding.EncodeToString(ciphertext), permanentAttachmentDownloadTokenEncryptionKey())
	if err != nil {
		return nil, errors.New("invalid permanent attachment token payload")
	}
	result := &PermanentAttachmentDownloadToken{}
	if err = json.Unmarshal([]byte(payload), result); err != nil {
		return nil, errors.New("invalid permanent attachment token payload")
	}
	if strings.TrimSpace(result.Path) == "" || result.Identifie == "" || result.Version == "" {
		return nil, errors.New("incomplete permanent attachment token payload")
	}
	return result, nil
}

func permanentAttachmentDownloadTokenEncryptionKey() string {
	return function.GetMd5(facade.GetConfig().GetString("setting.secret"))
}

type Attach struct {
}

func (l Attach) GetZipFileContent(cacheRoot string, zipPath string, filePath string) ([]byte, error) {
	cacheDir, err := l.ensureZipCache(cacheRoot, zipPath)
	if err != nil {
		return nil, err
	}
	l.triggerZipCacheCleanup(cacheRoot)

	cleanPath, err := function.CleanZipFilePath(filePath)
	if err != nil {
		return nil, err
	}

	targetPath := filepath.Join(cacheDir, filepath.FromSlash(cleanPath))
	if !function.IsPathInDir(targetPath, cacheDir) {
		return nil, fmt.Errorf("非法文件路径: %s", filePath)
	}

	return os.ReadFile(targetPath)
}

func (l Attach) ensureZipCache(cacheRoot string, zipPath string) (string, error) {
	absZipPath, err := filepath.Abs(zipPath)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(absZipPath)
	if err != nil {
		return "", err
	}

	cacheKey := l.zipCacheKey(absZipPath, info)
	cacheDir := filepath.Join(cacheRoot, cacheKey)
	readyFile := cacheDir + ".ready"
	if function.FileExists(readyFile) {
		_ = os.Chtimes(readyFile, time.Now(), time.Now())
		return cacheDir, nil
	}

	lockValue, _ := zipCacheLocks.LoadOrStore(absZipPath, &zipCacheLock{})
	lock := lockValue.(*zipCacheLock)
	lock.mu.Lock()
	defer lock.mu.Unlock()

	if function.FileExists(readyFile) {
		_ = os.Chtimes(readyFile, time.Now(), time.Now())
		return cacheDir, nil
	}

	tmpDir := cacheDir + ".tmp"
	if err := os.RemoveAll(tmpDir); err != nil {
		return "", err
	}
	if err := os.RemoveAll(cacheDir); err != nil {
		return "", err
	}
	if err := os.MkdirAll(tmpDir, os.ModePerm); err != nil {
		return "", err
	}
	if err := function.ExtractZip(absZipPath, tmpDir); err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", err
	}
	if err := os.Rename(tmpDir, cacheDir); err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", err
	}
	if err := os.WriteFile(readyFile, []byte(absZipPath), service.FileMode); err != nil {
		return "", err
	}

	return cacheDir, nil
}

func (l Attach) zipCacheKey(zipPath string, info os.FileInfo) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%d:%d", zipPath, info.Size(), info.ModTime().UnixNano())))
	return fmt.Sprintf("%x", sum[:])
}

func (l Attach) triggerZipCacheCleanup(cacheRoot string) {
	now := time.Now()

	zipCacheCleanupMu.Lock()
	if !zipCacheLastCleanup.IsZero() && now.Sub(zipCacheLastCleanup) < zipCacheCleanupInterval {
		zipCacheCleanupMu.Unlock()
		return
	}
	if zipCacheCleanupRunning {
		zipCacheCleanupMu.Unlock()
		return
	}
	zipCacheLastCleanup = now
	zipCacheCleanupRunning = true
	zipCacheCleanupMu.Unlock()

	go func() {
		defer func() {
			zipCacheCleanupMu.Lock()
			zipCacheCleanupRunning = false
			zipCacheCleanupMu.Unlock()
		}()

		l.cleanupZipCache(cacheRoot, now)
	}()
}

func (l Attach) cleanupZipCache(cacheRoot string, now time.Time) {
	entries, err := os.ReadDir(cacheRoot)
	if err != nil {
		return
	}

	for _, entry := range entries {
		entryPath := filepath.Join(cacheRoot, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		if entry.IsDir() {
			if filepath.Ext(entry.Name()) == ".tmp" && now.Sub(info.ModTime()) > zipCacheTmpTTL {
				_ = os.RemoveAll(entryPath)
			}
			continue
		}

		if filepath.Ext(entry.Name()) != ".ready" || now.Sub(info.ModTime()) <= zipCacheTTL {
			continue
		}

		cacheDir := entryPath[:len(entryPath)-len(".ready")]
		_ = os.RemoveAll(cacheDir)
		_ = os.Remove(entryPath)
	}
}
