package logic

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/w7panel/w7panel-zpk/common/function"
	"github.com/w7panel/w7panel-zpk/common/service"
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
