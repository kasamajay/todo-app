package auth

import (
	"net/url"
	"strings"
	"testing"
)

func TestRandomState_NonEmptyAndDistinct(t *testing.T) {
	a, err := RandomState()
	if err != nil {
		t.Fatalf("RandomState() error = %v", err)
	}
	if a == "" {
		t.Fatal("RandomState() returned empty string")
	}

	b, err := RandomState()
	if err != nil {
		t.Fatalf("RandomState() error = %v", err)
	}
	if a == b {
		t.Fatalf("RandomState() returned the same value twice: %q", a)
	}
}

func TestGoogleAuthURL(t *testing.T) {
	got := GoogleAuthURL("client-123", "http://localhost:5173/api/auth/google/callback", "state-abc")

	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("GoogleAuthURL() produced an unparseable URL: %v", err)
	}
	if u.Scheme+"://"+u.Host+u.Path != googleAuthURL {
		t.Errorf("GoogleAuthURL() base = %q, want %q", u.Scheme+"://"+u.Host+u.Path, googleAuthURL)
	}

	q := u.Query()
	if q.Get("client_id") != "client-123" {
		t.Errorf("client_id = %q, want %q", q.Get("client_id"), "client-123")
	}
	if q.Get("redirect_uri") != "http://localhost:5173/api/auth/google/callback" {
		t.Errorf("redirect_uri = %q", q.Get("redirect_uri"))
	}
	if q.Get("response_type") != "code" {
		t.Errorf("response_type = %q, want %q", q.Get("response_type"), "code")
	}
	if q.Get("state") != "state-abc" {
		t.Errorf("state = %q, want %q", q.Get("state"), "state-abc")
	}
	if !strings.Contains(q.Get("scope"), "openid") || !strings.Contains(q.Get("scope"), "email") {
		t.Errorf("scope = %q, want it to contain openid and email", q.Get("scope"))
	}
}
