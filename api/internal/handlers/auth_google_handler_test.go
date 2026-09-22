package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"

	"todo-app/internal/storage"
)

func newTestAuthHandler(t *testing.T, configured bool) *AuthHandler {
	t.Helper()
	users, err := storage.NewUserStore(filepath.Join(t.TempDir(), "users.json"))
	if err != nil {
		t.Fatalf("NewUserStore() error = %v", err)
	}
	h := &AuthHandler{
		Users:             users,
		Secret:            []byte("test-secret"),
		GoogleRedirectURI: "http://localhost:5173/api/auth/google/callback",
		FrontendBaseURL:   "http://localhost:5173",
	}
	if configured {
		h.GoogleClientID = "test-client-id"
		h.GoogleClientSecret = "test-client-secret"
	}
	return h
}

func TestGoogleLogin_NotConfigured(t *testing.T) {
	h := newTestAuthHandler(t, false)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/google/login", nil)
	rec := httptest.NewRecorder()
	h.GoogleLogin(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestGoogleLogin_RedirectsToGoogleWithStateCookie(t *testing.T) {
	h := newTestAuthHandler(t, true)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/google/login", nil)
	rec := httptest.NewRecorder()
	h.GoogleLogin(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}

	loc, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatalf("Location header not a valid URL: %v", err)
	}
	if loc.Host != "accounts.google.com" {
		t.Errorf("redirect host = %q, want accounts.google.com", loc.Host)
	}
	q := loc.Query()
	if q.Get("client_id") != "test-client-id" {
		t.Errorf("client_id = %q", q.Get("client_id"))
	}
	if q.Get("redirect_uri") != h.GoogleRedirectURI {
		t.Errorf("redirect_uri = %q, want %q", q.Get("redirect_uri"), h.GoogleRedirectURI)
	}
	if q.Get("response_type") != "code" {
		t.Errorf("response_type = %q, want code", q.Get("response_type"))
	}
	if q.Get("state") == "" {
		t.Error("state query param is empty")
	}

	var stateCookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == googleStateCookie {
			stateCookie = c
			break
		}
	}
	if stateCookie == nil {
		t.Fatal("state cookie was not set")
	}
	if stateCookie.Value != q.Get("state") {
		t.Errorf("cookie state %q != redirect state %q", stateCookie.Value, q.Get("state"))
	}
	if !stateCookie.HttpOnly {
		t.Error("state cookie should be HttpOnly")
	}
}

func TestGoogleCallback_MissingStateCookie(t *testing.T) {
	h := newTestAuthHandler(t, true)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/google/callback?code=abc&state=xyz", nil)
	rec := httptest.NewRecorder()
	h.GoogleCallback(rec, req)

	assertRedirectsWithGoogleError(t, rec, "google_state_missing")
}

func TestGoogleCallback_StateMismatch(t *testing.T) {
	h := newTestAuthHandler(t, true)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/google/callback?code=abc&state=wrong-value", nil)
	req.AddCookie(&http.Cookie{Name: googleStateCookie, Value: "correct-value"})
	rec := httptest.NewRecorder()
	h.GoogleCallback(rec, req)

	assertRedirectsWithGoogleError(t, rec, "google_state_mismatch")
}

func TestGoogleCallback_UserDenied(t *testing.T) {
	h := newTestAuthHandler(t, true)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/google/callback?error=access_denied", nil)
	rec := httptest.NewRecorder()
	h.GoogleCallback(rec, req)

	assertRedirectsWithGoogleError(t, rec, "google_denied")
}

func assertRedirectsWithGoogleError(t *testing.T, rec *httptest.ResponseRecorder, wantCode string) {
	t.Helper()
	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}
	loc, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatalf("Location header not a valid URL: %v", err)
	}
	frag, err := url.ParseQuery(loc.Fragment)
	if err != nil {
		t.Fatalf("could not parse redirect fragment %q: %v", loc.Fragment, err)
	}
	if got := frag.Get("google_error"); got != wantCode {
		t.Errorf("google_error = %q, want %q", got, wantCode)
	}
}
