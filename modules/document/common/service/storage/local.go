// Package storage defines an object-key API; the initial driver uses only the local filesystem.
package storage

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path"
	"strings"
)

type Local struct{ root *os.Root }

func NewLocal(dir string) (*Local, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	r, e := os.OpenRoot(dir)
	return &Local{r}, e
}

func (l *Local) Close() error { return l.root.Close() }
func (l *Local) Put(ctx context.Context, b []byte) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	hash := hex.EncodeToString(sum[:])
	dir := "blobs/" + hash[:2]
	key := dir + "/" + hash
	if err := l.root.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	// Publish complete immutable objects atomically. Concurrent identical writers are harmless.
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	tmp := ".document-" + hex.EncodeToString(random)
	f, e := l.root.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return "", e
	}
	defer l.root.Remove(tmp)
	if _, e = io.Copy(f, bytes.NewReader(b)); e != nil {
		f.Close()
		return "", e
	}
	if e = f.Sync(); e != nil {
		f.Close()
		return "", e
	}
	if e = f.Close(); e != nil {
		return "", e
	}
	if e = l.root.Rename(tmp, key); e != nil {
		return "", e
	}
	return key, nil
}

func (l *Local) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	if !strings.HasPrefix(key, "blobs/") || path.Clean(key) != key || strings.Contains(key, "\\") {
		return nil, errors.New("无效存储键")
	}
	return l.root.Open(key)
}
