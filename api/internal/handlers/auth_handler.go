package handlers

import (
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"todo-app/internal/auth"
	"todo-app/internal/idgen"
	"todo-app/internal/middleware"
	"todo-app/internal/models"
	"todo-app/internal/storage"
)

const (
	maxFailedLogins   = 3
	lockDuration      = 30 * time.Minute
	resetTokenTTL     = time.Hour
	minPasswordLength = 8

	twoFactorCodeTTL     = 10 * time.Minute
	twoFactorMaxAttempts = 5
)

type AuthHandler struct {
	Users  *storage.UserStore
	Secret []byte

	// Google OAuth config. GoogleClientID/GoogleClientSecret are blank when
	// Sign in with Google isn't configured, in which case the google/*
	// routes respond 503 rather than failing at startup.
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURI  string
	FrontendBaseURL    string
}

var (
	dummyOnce sync.Once
	dummySalt []byte
	dummyHash []byte
)

// dummyLoginParams returns a fixed salt/hash pair used when the submitted
// email doesn't match any user, so PBKDF2 still runs with the same cost as
// a real login and response timing can't reveal account existence.
func dummyLoginParams() (salt, hash []byte) {
	dummyOnce.Do(func() {
		dummySalt = []byte("fixed-dummy-salt-16b")
		dummyHash, _, _ = auth.HashPassword("dummy-password-for-timing-only")
	})
	return dummySalt, dummyHash
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" || !strings.Contains(email, "@") {
		writeError(w, http.StatusBadRequest, "invalid_email", "a valid email is required")
		return
	}
	if len(req.Password) < minPasswordLength {
		writeError(w, http.StatusBadRequest, "invalid_password", "password must be at least 8 characters")
		return
	}
	if _, found := h.Users.FindByEmail(email); found {
		writeError(w, http.StatusConflict, "email_taken", "an account with that email already exists")
		return
	}

	hash, salt, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to hash password")
		return
	}

	user := models.User{
		ID:           idgen.New(),
		Email:        email,
		PasswordHash: hash,
		Salt:         salt,
		CreatedAt:    time.Now(),
	}
	if err := h.Users.Put(user.ID, user); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to create account")
		return
	}

	token, err := h.mint(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to mint token")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"token": token, "user": user.Public()})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))

	user, found := h.Users.FindByEmail(email)
	hasPassword := found && len(user.PasswordHash) > 0

	var salt, hash []byte
	if hasPassword {
		salt, hash = user.Salt, user.PasswordHash
	} else {
		salt, hash = dummyLoginParams()
	}

	// Always run PBKDF2 - found or not, password set or not - so response
	// timing doesn't reveal whether the email is registered or is a
	// Google-only account with no password set.
	match := auth.VerifyPassword(req.Password, hash, salt)

	if !hasPassword {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
		return
	}

	if user.IsLocked() {
		writeError(w, http.StatusForbidden, "account_locked", "account is locked, try again later")
		return
	}

	if !match {
		user.FailedLoginCount++
		if user.FailedLoginCount >= maxFailedLogins {
			user.LockedUntil = time.Now().Add(lockDuration)
		}
		if err := h.Users.Put(user.ID, user); err != nil {
			log.Printf("failed to persist failed login count for %s: %v", user.ID, err)
		}
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
		return
	}

	user.FailedLoginCount = 0
	user.LockedUntil = time.Time{}
	if err := h.Users.Put(user.ID, user); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to update account")
		return
	}

	if user.TwoFactorEnabled {
		updated, err := h.issueTwoFactorChallenge(user)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to start two-factor verification")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"two_factor_required": true, "user_id": updated.ID})
		return
	}

	token, err := h.mint(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to mint token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"token": token, "user": user.Public()})
}

// Logout is a client-side token discard - tokens are stateless HMAC tokens
// with no server-side session store, so there is nothing to revoke here.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
		return
	}
	writeJSON(w, http.StatusOK, user.Public())
}

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req forgotPasswordRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	if user, found := h.Users.FindByEmail(email); found {
		user.ResetToken = idgen.New()
		user.ResetTokenExpires = time.Now().Add(resetTokenTTL)
		if err := h.Users.Put(user.ID, user); err == nil {
			// No email infrastructure exists in this project, so the reset
			// token is logged server-side instead of emailed.
			log.Printf("password reset requested for %s: reset_token=%s (expires %s)",
				user.Email, user.ResetToken, user.ResetTokenExpires.Format(time.RFC3339))
		}
	}

	// Always respond 200 regardless of whether the email exists, so this
	// endpoint can't be used to enumerate registered accounts.
	writeJSON(w, http.StatusOK, map[string]string{"message": "if that email exists, a reset link has been sent"})
}

type resetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req resetPasswordRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}
	if len(req.NewPassword) < minPasswordLength {
		writeError(w, http.StatusBadRequest, "invalid_password", "password must be at least 8 characters")
		return
	}

	user, found := h.Users.FindByResetToken(req.Token)
	if !found || req.Token == "" || time.Now().After(user.ResetTokenExpires) {
		writeError(w, http.StatusBadRequest, "invalid_token", "reset token is invalid or expired")
		return
	}

	hash, salt, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to hash password")
		return
	}
	user.PasswordHash = hash
	user.Salt = salt
	user.ResetToken = ""
	user.ResetTokenExpires = time.Time{}
	user.FailedLoginCount = 0
	user.LockedUntil = time.Time{}

	if err := h.Users.Put(user.ID, user); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to reset password")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "password has been reset"})
}

// issueTwoFactorChallenge generates a 6-digit code for user, persists it
// with a short expiry, and logs it server-side instead of emailing it (no
// email infrastructure exists in this project - see decisions/0007 and
// decisions/0011). Shared by Login and GoogleCallback, since both reach the
// same account and must gate on 2FA identically.
func (h *AuthHandler) issueTwoFactorChallenge(user models.User) (models.User, error) {
	code, err := auth.GenerateSixDigitCode()
	if err != nil {
		return user, err
	}
	user.TwoFactorCode = code
	user.TwoFactorCodeExpires = time.Now().Add(twoFactorCodeTTL)
	user.TwoFactorAttempts = 0
	if err := h.Users.Put(user.ID, user); err != nil {
		return user, err
	}
	log.Printf("two-factor code requested for %s: code=%s (expires %s)",
		user.Email, user.TwoFactorCode, user.TwoFactorCodeExpires.Format(time.RFC3339))
	return user, nil
}

type verifyTwoFactorRequest struct {
	UserID string `json:"user_id"`
	Code   string `json:"code"`
}

// VerifyTwoFactor completes a login that issueTwoFactorChallenge put on
// hold: it's a public route (the caller isn't fully authenticated yet),
// gated by the code itself plus a small independent attempt budget rather
// than a bearer token.
func (h *AuthHandler) VerifyTwoFactor(w http.ResponseWriter, r *http.Request) {
	var req verifyTwoFactorRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}
	req.UserID = strings.TrimSpace(req.UserID)
	req.Code = strings.TrimSpace(req.Code)
	if req.UserID == "" || req.Code == "" {
		writeError(w, http.StatusBadRequest, "invalid_body", "user_id and code are required")
		return
	}

	user, found := h.Users.Get(req.UserID)
	if !found || user.TwoFactorCode == "" {
		writeError(w, http.StatusUnauthorized, "two_factor_not_pending", "no verification code is pending for this account")
		return
	}

	if time.Now().After(user.TwoFactorCodeExpires) {
		user.TwoFactorCode = ""
		user.TwoFactorCodeExpires = time.Time{}
		user.TwoFactorAttempts = 0
		if err := h.Users.Put(user.ID, user); err != nil {
			log.Printf("failed to clear expired two-factor code for %s: %v", user.ID, err)
		}
		writeError(w, http.StatusUnauthorized, "two_factor_code_expired", "verification code has expired, please log in again")
		return
	}

	if req.Code != user.TwoFactorCode {
		user.TwoFactorAttempts++
		if user.TwoFactorAttempts >= twoFactorMaxAttempts {
			user.TwoFactorCode = ""
			user.TwoFactorCodeExpires = time.Time{}
			user.TwoFactorAttempts = 0
			if err := h.Users.Put(user.ID, user); err != nil {
				log.Printf("failed to invalidate two-factor code for %s: %v", user.ID, err)
			}
			writeError(w, http.StatusUnauthorized, "two_factor_too_many_attempts", "too many incorrect attempts, please log in again")
			return
		}
		if err := h.Users.Put(user.ID, user); err != nil {
			log.Printf("failed to persist two-factor attempt count for %s: %v", user.ID, err)
		}
		writeError(w, http.StatusUnauthorized, "two_factor_code_incorrect", "incorrect verification code")
		return
	}

	user.TwoFactorCode = ""
	user.TwoFactorCodeExpires = time.Time{}
	user.TwoFactorAttempts = 0
	if err := h.Users.Put(user.ID, user); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to update account")
		return
	}

	token, err := h.mint(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to mint token")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": token, "user": user.Public()})
}

type updateTwoFactorRequest struct {
	Enabled bool `json:"enabled"`
}

// UpdateTwoFactor lets the authenticated caller enable/disable 2FA on their
// own account - opt-in only, no admin override (see decisions/0011).
func (h *AuthHandler) UpdateTwoFactor(w http.ResponseWriter, r *http.Request) {
	ctxUser, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
		return
	}
	var req updateTwoFactorRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}

	user, found := h.Users.Get(ctxUser.ID)
	if !found {
		writeError(w, http.StatusUnauthorized, "unauthorized", "account not found")
		return
	}
	user.TwoFactorEnabled = req.Enabled
	if !req.Enabled {
		user.TwoFactorCode = ""
		user.TwoFactorCodeExpires = time.Time{}
		user.TwoFactorAttempts = 0
	}
	if err := h.Users.Put(user.ID, user); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to update account")
		return
	}
	writeJSON(w, http.StatusOK, user.Public())
}

func (h *AuthHandler) mint(user models.User) (string, error) {
	return auth.Mint(h.Secret, auth.Claims{
		UserID:    user.ID,
		IsAdmin:   user.IsAdmin,
		ExpiresAt: time.Now().Add(auth.TokenTTL),
	})
}
