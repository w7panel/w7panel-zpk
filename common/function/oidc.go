package function

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type oidcStatePayload struct {
	Nonce string `json:"nonce"`
	Exp   int64  `json:"exp"`
}

func RandomOIDCToken() (string, error) {
	return base64.RawURLEncoding.EncodeToString([]byte(GetRandomString(32))), nil
}

func MakeOIDCState(nonce string, secret string, ttl time.Duration) (string, error) {
	payload, err := json.Marshal(oidcStatePayload{
		Nonce: nonce,
		Exp:   time.Now().Add(ttl).Unix(),
	})
	if err != nil {
		return "", err
	}
	payloadPart := base64.RawURLEncoding.EncodeToString(payload)
	return payloadPart + "." + signOIDCState(payloadPart, secret), nil
}

func VerifyOIDCState(state string, secret string) (string, error) {
	parts := strings.Split(state, ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", errors.New("oidc state mismatch")
	}
	if !hmac.Equal([]byte(parts[1]), []byte(signOIDCState(parts[0], secret))) {
		return "", errors.New("oidc state mismatch")
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", errors.New("oidc state mismatch")
	}
	payload := oidcStatePayload{}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return "", errors.New("oidc state mismatch")
	}
	if payload.Nonce == "" || time.Now().Unix() >= payload.Exp {
		return "", errors.New("oidc state expired")
	}
	return payload.Nonce, nil
}

func signOIDCState(payloadPart string, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payloadPart))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
