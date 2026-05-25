package logic

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"time"
)

const (
	AttachTypeFrontend = "frontend"
	AttachTypeBackend  = "backend"
	AttachTypeHelm     = "helm"
)

type Attachment struct {
	Path      string    `json:"path"`
	Type      string    `json:"type"`
	AddedAt   time.Time `json:"added_at"`
	MediaHint string    `json:"media_hint,omitempty"`
	Artifact  string    `json:"artifact"`
}

type Session struct {
	Host          string       `json:"host"`
	Username      string       `json:"username"`
	Password      string       `json:"encrypted_password"`
	Artifact      string       `json:"artifact"`
	OciRegistry   string       `json:"oci_registry,omitempty"`
	OciRepository string       `json:"oci_repository,omitempty"`
	OciTag        string       `json:"oci_tag,omitempty"`
	Attachments   []Attachment `json:"attachments"`
	UpdatedAt     time.Time    `json:"updated_at"`
}

func LoadSession() (*Session, error) {
	path, err := sessionFilePath()
	if err != nil {
		return nil, err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Session{}, nil
		}
		return nil, err
	}
	if len(content) == 0 {
		return &Session{}, nil
	}

	var session Session
	if err := json.Unmarshal(content, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

func SaveSession(session *Session) error {
	path, err := sessionFilePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	session.UpdatedAt = time.Now()
	content, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o600)
}
func EncryptPassword(password string) (string, error) {
	key := encryptionKey()
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aesgcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := aesgcm.Seal(nonce, nonce, []byte(password), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func DecryptPassword(ciphertext string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	key := encryptionKey()
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < aesgcm.NonceSize() {
		return "", fmt.Errorf("encrypted password is invalid")
	}

	nonce, data := raw[:aesgcm.NonceSize()], raw[aesgcm.NonceSize():]
	plaintext, err := aesgcm.Open(nil, nonce, data, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func NormalizeAttachType(value string) (string, error) {
	if slices.Contains([]string{AttachTypeFrontend, AttachTypeBackend, AttachTypeHelm}, value) {
		return value, nil
	}

	return "", fmt.Errorf("unsupported attachment type %q, allowed: frontend, backend, helm", value)
}

func encryptionKey() [32]byte {
	return sha256.Sum256([]byte("w7-zpk-cli-session-v1"))
}

func sessionFilePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "w7-zpk", "session.json"), nil
}
