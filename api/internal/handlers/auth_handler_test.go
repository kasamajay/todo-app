package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"todo-app/internal/idgen"
	"todo-app/internal/models"
)

func TestLogin_RejectsGoogleOnlyAccountPasswordAttempt(t *testing.T) {
	h := newTestAuthHandler(t, false)

	user := models.User{
		ID:        idgen.New(),
		Email:     "google-only@example.com",
		GoogleID:  "google-sub-123",
		CreatedAt: time.Now(),
	}
	if err := h.Users.Put(user.ID, user); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	body := `{"email":"google-only@example.com","password":"whatever123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "invalid_credentials") {
		t.Errorf("body = %s, want it to contain invalid_credentials", rec.Body.String())
	}
}
