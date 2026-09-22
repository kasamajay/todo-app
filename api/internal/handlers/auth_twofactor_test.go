package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"todo-app/internal/auth"
	"todo-app/internal/idgen"
	"todo-app/internal/middleware"
	"todo-app/internal/models"
)

func seedUserWithPassword(t *testing.T, h *AuthHandler, email, password string, twoFactorEnabled bool) models.User {
	t.Helper()
	hash, salt, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	user := models.User{
		ID:               idgen.New(),
		Email:            email,
		PasswordHash:     hash,
		Salt:             salt,
		TwoFactorEnabled: twoFactorEnabled,
		CreatedAt:        time.Now(),
	}
	if err := h.Users.Put(user.ID, user); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	return user
}

func loginBody(email, password string) string {
	return `{"email":"` + email + `","password":"` + password + `"}`
}

func TestLogin_TwoFactorEnabled_ReturnsChallengeNotToken(t *testing.T) {
	h := newTestAuthHandler(t, false)
	user := seedUserWithPassword(t, h, "2fa@example.com", "correct-password", true)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(loginBody(user.Email, "correct-password")))
	rec := httptest.NewRecorder()
	h.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body = %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"two_factor_required":true`) {
		t.Errorf("body = %s, want two_factor_required:true", body)
	}
	if !strings.Contains(body, user.ID) {
		t.Errorf("body = %s, want it to contain user_id %s", body, user.ID)
	}
	if strings.Contains(body, `"token"`) {
		t.Errorf("body = %s, want no token when 2FA is required", body)
	}

	updated, found := h.Users.Get(user.ID)
	if !found {
		t.Fatal("user disappeared from store")
	}
	if updated.TwoFactorCode == "" {
		t.Error("expected a pending two-factor code to be set")
	}
	if updated.TwoFactorCodeExpires.Before(time.Now()) {
		t.Error("expected TwoFactorCodeExpires to be in the future")
	}
}

func TestLogin_TwoFactorDisabled_ReturnsTokenDirectly(t *testing.T) {
	h := newTestAuthHandler(t, false)
	user := seedUserWithPassword(t, h, "nofa@example.com", "correct-password", false)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(loginBody(user.Email, "correct-password")))
	rec := httptest.NewRecorder()
	h.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"token"`) {
		t.Errorf("body = %s, want a token when 2FA is disabled", rec.Body.String())
	}
}

func TestVerifyTwoFactor_CorrectCode(t *testing.T) {
	h := newTestAuthHandler(t, false)
	user := seedUserWithPassword(t, h, "verify-ok@example.com", "correct-password", true)
	updated, err := h.issueTwoFactorChallenge(user)
	if err != nil {
		t.Fatalf("issueTwoFactorChallenge() error = %v", err)
	}

	body := `{"user_id":"` + updated.ID + `","code":"` + updated.TwoFactorCode + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/2fa/verify", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.VerifyTwoFactor(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"token"`) {
		t.Errorf("body = %s, want a token on correct code", rec.Body.String())
	}

	final, _ := h.Users.Get(updated.ID)
	if final.TwoFactorCode != "" || final.TwoFactorAttempts != 0 {
		t.Errorf("expected pending 2FA state cleared, got code=%q attempts=%d", final.TwoFactorCode, final.TwoFactorAttempts)
	}
}

func TestVerifyTwoFactor_WrongCode_IncrementsAttempts(t *testing.T) {
	h := newTestAuthHandler(t, false)
	user := seedUserWithPassword(t, h, "verify-wrong@example.com", "correct-password", true)
	updated, _ := h.issueTwoFactorChallenge(user)

	body := `{"user_id":"` + updated.ID + `","code":"000000"}`
	if updated.TwoFactorCode == "000000" {
		body = `{"user_id":"` + updated.ID + `","code":"111111"}`
	}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/2fa/verify", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.VerifyTwoFactor(rec, req)

	if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), "two_factor_code_incorrect") {
		t.Fatalf("status = %d, body = %s, want 401 two_factor_code_incorrect", rec.Code, rec.Body.String())
	}

	final, _ := h.Users.Get(updated.ID)
	if final.TwoFactorAttempts != 1 {
		t.Errorf("TwoFactorAttempts = %d, want 1", final.TwoFactorAttempts)
	}
	if final.TwoFactorCode == "" {
		t.Error("code should still be pending after a single wrong attempt")
	}
}

func TestVerifyTwoFactor_TooManyAttempts_InvalidatesCode(t *testing.T) {
	h := newTestAuthHandler(t, false)
	user := seedUserWithPassword(t, h, "verify-lockout@example.com", "correct-password", true)
	updated, _ := h.issueTwoFactorChallenge(user)
	wrongCode := "999999"
	if updated.TwoFactorCode == wrongCode {
		wrongCode = "888888"
	}

	var rec *httptest.ResponseRecorder
	for i := 0; i < twoFactorMaxAttempts; i++ {
		body := `{"user_id":"` + updated.ID + `","code":"` + wrongCode + `"}`
		req := httptest.NewRequest(http.MethodPost, "/api/auth/2fa/verify", strings.NewReader(body))
		rec = httptest.NewRecorder()
		h.VerifyTwoFactor(rec, req)
	}

	if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), "two_factor_too_many_attempts") {
		t.Fatalf("status = %d, body = %s, want 401 two_factor_too_many_attempts", rec.Code, rec.Body.String())
	}

	final, _ := h.Users.Get(updated.ID)
	if final.TwoFactorCode != "" || final.TwoFactorAttempts != 0 || !final.TwoFactorCodeExpires.IsZero() {
		t.Errorf("expected all pending 2FA state cleared, got %+v", final)
	}
}

func TestVerifyTwoFactor_ExpiredCode(t *testing.T) {
	h := newTestAuthHandler(t, false)
	user := seedUserWithPassword(t, h, "verify-expired@example.com", "correct-password", true)
	updated, _ := h.issueTwoFactorChallenge(user)
	updated.TwoFactorCodeExpires = time.Now().Add(-time.Minute)
	if err := h.Users.Put(updated.ID, updated); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	body := `{"user_id":"` + updated.ID + `","code":"` + updated.TwoFactorCode + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/2fa/verify", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.VerifyTwoFactor(rec, req)

	if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), "two_factor_code_expired") {
		t.Fatalf("status = %d, body = %s, want 401 two_factor_code_expired", rec.Code, rec.Body.String())
	}
}

func TestVerifyTwoFactor_NoPendingChallenge(t *testing.T) {
	h := newTestAuthHandler(t, false)
	user := seedUserWithPassword(t, h, "verify-none@example.com", "correct-password", true)
	// No issueTwoFactorChallenge call - TwoFactorCode stays empty.

	cases := []string{user.ID, "unknown-user-id"}
	for _, id := range cases {
		body := `{"user_id":"` + id + `","code":"123456"}`
		req := httptest.NewRequest(http.MethodPost, "/api/auth/2fa/verify", strings.NewReader(body))
		rec := httptest.NewRecorder()
		h.VerifyTwoFactor(rec, req)

		if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), "two_factor_not_pending") {
			t.Errorf("user_id=%q: status = %d, body = %s, want 401 two_factor_not_pending", id, rec.Code, rec.Body.String())
		}
	}
}

func TestUpdateTwoFactor_RequiresBearerAuth(t *testing.T) {
	h := newTestAuthHandler(t, false)
	protected := middleware.RequireAuth(h.Secret, h.Users)(h.UpdateTwoFactor)

	req := httptest.NewRequest(http.MethodPut, "/api/auth/2fa", strings.NewReader(`{"enabled":true}`))
	rec := httptest.NewRecorder()
	protected(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 without a bearer token", rec.Code)
	}
}

func TestUpdateTwoFactor_TogglesFlag(t *testing.T) {
	h := newTestAuthHandler(t, false)
	user := seedUserWithPassword(t, h, "settings@example.com", "correct-password", false)
	protected := middleware.RequireAuth(h.Secret, h.Users)(h.UpdateTwoFactor)

	token, err := auth.Mint(h.Secret, auth.Claims{UserID: user.ID, ExpiresAt: time.Now().Add(time.Hour)})
	if err != nil {
		t.Fatalf("Mint() error = %v", err)
	}

	enable := httptest.NewRequest(http.MethodPut, "/api/auth/2fa", strings.NewReader(`{"enabled":true}`))
	enable.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	protected(rec, enable)

	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"two_factor_enabled":true`) {
		t.Fatalf("status = %d, body = %s, want 200 two_factor_enabled:true", rec.Code, rec.Body.String())
	}

	updated, _ := h.Users.Get(user.ID)
	updated, _ = h.issueTwoFactorChallenge(updated)
	if updated.TwoFactorCode == "" {
		t.Fatal("expected a pending code before disabling")
	}

	disable := httptest.NewRequest(http.MethodPut, "/api/auth/2fa", strings.NewReader(`{"enabled":false}`))
	disable.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	protected(rec, disable)

	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"two_factor_enabled":false`) {
		t.Fatalf("status = %d, body = %s, want 200 two_factor_enabled:false", rec.Code, rec.Body.String())
	}

	final, _ := h.Users.Get(user.ID)
	if final.TwoFactorEnabled {
		t.Error("expected TwoFactorEnabled to be false")
	}
	if final.TwoFactorCode != "" {
		t.Error("expected disabling 2FA to clear any pending code")
	}
}
