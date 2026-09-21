package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// TokenTTL is how long a minted token remains valid.
const TokenTTL = 24 * time.Hour

var (
	ErrMalformedToken = errors.New("malformed token")
	ErrInvalidToken   = errors.New("invalid token signature")
	ErrExpiredToken   = errors.New("token expired")
)

// Claims is the payload embedded in every token. Tokens are self-contained
// and stateless (custom HMAC-SHA256 signed, not JWT) - there is no
// server-side session store, so "logout" is a client-side token discard.
type Claims struct {
	UserID    string    `json:"uid"`
	IsAdmin   bool      `json:"adm"`
	ExpiresAt time.Time `json:"exp"`
}

// Mint produces a token of the form base64url(json claims) + "." + hex(HMAC-SHA256(...)).
func Mint(secret []byte, c Claims) (string, error) {
	payload, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	sig := sign(secret, encoded)
	return encoded + "." + sig, nil
}

// Verify checks the token's signature and expiry, returning its Claims.
func Verify(secret []byte, token string) (Claims, error) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return Claims{}, ErrMalformedToken
	}
	encoded, providedSig := parts[0], parts[1]

	expectedSig := sign(secret, encoded)
	if !hmac.Equal([]byte(expectedSig), []byte(providedSig)) {
		return Claims{}, ErrInvalidToken
	}

	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return Claims{}, ErrMalformedToken
	}

	var c Claims
	if err := json.Unmarshal(payload, &c); err != nil {
		return Claims{}, ErrMalformedToken
	}

	if time.Now().After(c.ExpiresAt) {
		return Claims{}, ErrExpiredToken
	}
	return c, nil
}

func sign(secret []byte, encodedPayload string) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(encodedPayload))
	return hex.EncodeToString(mac.Sum(nil))
}
