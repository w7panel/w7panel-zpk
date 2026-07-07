package function

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type oidcStatePayload struct {
	Nonce       string `json:"nonce"`
	RedirectURL string `json:"redirect_url,omitempty"`
	Exp         int64  `json:"exp"`
}

type OIDCState struct {
	Nonce       string
	RedirectURL string
}

func RandomOIDCToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func MakeOIDCState(nonce string, redirectURL string, secret string, ttl time.Duration) (string, error) {
	payload, err := json.Marshal(oidcStatePayload{
		Nonce:       nonce,
		RedirectURL: redirectURL,
		Exp:         time.Now().Add(ttl).Unix(),
	})
	if err != nil {
		return "", err
	}
	payloadPart := base64.RawURLEncoding.EncodeToString(payload)
	return payloadPart + "." + signOIDCState(payloadPart, secret), nil
}

func VerifyOIDCState(state string, secret string) (OIDCState, error) {
	parts := strings.Split(state, ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return OIDCState{}, errors.New("oidc state mismatch")
	}
	if !hmac.Equal([]byte(parts[1]), []byte(signOIDCState(parts[0], secret))) {
		return OIDCState{}, errors.New("oidc state mismatch")
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return OIDCState{}, errors.New("oidc state mismatch")
	}
	payload := oidcStatePayload{}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return OIDCState{}, errors.New("oidc state mismatch")
	}
	if payload.Nonce == "" || time.Now().Unix() >= payload.Exp {
		return OIDCState{}, errors.New("oidc state expired")
	}
	return OIDCState{
		Nonce:       payload.Nonce,
		RedirectURL: payload.RedirectURL,
	}, nil
}

func signOIDCState(payloadPart string, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payloadPart))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
