package auth

import (
	"strings"
	"testing"
	"time"
)

func TestMintAndVerify_RoundTrip(t *testing.T) {
	secret := []byte("test-secret-32-bytes-long-------")
	claims := Claims{UserID: "u1", IsAdmin: false, ExpiresAt: time.Now().Add(time.Hour)}

	token, err := Mint(secret, claims)
	if err != nil {
		t.Fatalf("Mint() error = %v", err)
	}
	if !strings.Contains(token, ".") {
		t.Fatalf("token should contain a '.' separator, got %q", token)
	}

	got, err := Verify(secret, token)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if got.UserID != claims.UserID || got.IsAdmin != claims.IsAdmin {
		t.Errorf("Verify() = %+v, want %+v", got, claims)
	}
}

func TestVerify_RejectsTamperedSignature(t *testing.T) {
	secret := []byte("secret")
	token, _ := Mint(secret, Claims{UserID: "u1", ExpiresAt: time.Now().Add(time.Hour)})

	parts := strings.SplitN(token, ".", 2)
	tampered := parts[0] + ".0000000000000000000000000000000000000000000000000000000000000000"

	if _, err := Verify(secret, tampered); err != ErrInvalidToken {
		t.Errorf("Verify() error = %v, want ErrInvalidToken", err)
	}
}

func TestVerify_RejectsWrongSecret(t *testing.T) {
	token, _ := Mint([]byte("secret-a"), Claims{UserID: "u1", ExpiresAt: time.Now().Add(time.Hour)})

	if _, err := Verify([]byte("secret-b"), token); err != ErrInvalidToken {
		t.Errorf("Verify() error = %v, want ErrInvalidToken", err)
	}
}

func TestVerify_RejectsExpiredToken(t *testing.T) {
	secret := []byte("secret")
	token, _ := Mint(secret, Claims{UserID: "u1", ExpiresAt: time.Now().Add(-time.Minute)})

	if _, err := Verify(secret, token); err != ErrExpiredToken {
		t.Errorf("Verify() error = %v, want ErrExpiredToken", err)
	}
}

func TestVerify_RejectsMalformedToken(t *testing.T) {
	secret := []byte("secret")
	cases := []string{"", "no-dot-here", ".", "abc.", ".def"}
	for _, tok := range cases {
		if _, err := Verify(secret, tok); err != ErrMalformedToken {
			t.Errorf("Verify(%q) error = %v, want ErrMalformedToken", tok, err)
		}
	}
}
