package handlers

import (
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"todo-app/internal/auth"
	"todo-app/internal/idgen"
	"todo-app/internal/models"
)

const (
	googleStateCookie = "google_oauth_state"
	googleStatePath   = "/api/auth/google"
	googleStateTTL    = 5 * time.Minute
)

func (h *AuthHandler) googleConfigured() bool {
	return h.GoogleClientID != "" && h.GoogleClientSecret != ""
}

// GoogleLogin starts Google's OAuth 2.0 Authorization Code flow: it sets a
// short-lived CSRF state cookie and redirects the browser to Google's
// consent screen. See decisions/0010-google-oauth-authorization-code-flow.md.
func (h *AuthHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	if !h.googleConfigured() {
		writeError(w, http.StatusServiceUnavailable, "google_oauth_not_configured", "Google sign-in is not configured on this server")
		return
	}

	state, err := auth.RandomState()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to start Google sign-in")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     googleStateCookie,
		Value:    state,
		Path:     googleStatePath,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(googleStateTTL.Seconds()),
	})

	http.Redirect(w, r, auth.GoogleAuthURL(h.GoogleClientID, h.GoogleRedirectURI, state), http.StatusFound)
}

// GoogleCallback completes the Authorization Code flow: it exchanges the
// code Google redirected back with for an access token, fetches the user's
// Google profile, resolves it to a local account (matching by GoogleID,
// then auto-linking by verified email, then creating a new account), mints
// the same kind of app token password login uses, and redirects back to the
// frontend with the token in the URL fragment (never a query param, so it
// never reaches server access logs).
func (h *AuthHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	fail := func(code string) {
		http.Redirect(w, r, h.FrontendBaseURL+"/#google_error="+url.QueryEscape(code), http.StatusFound)
	}

	if !h.googleConfigured() {
		fail("google_oauth_not_configured")
		return
	}
	if r.URL.Query().Get("error") != "" {
		fail("google_denied")
		return
	}

	cookie, cookieErr := r.Cookie(googleStateCookie)
	// One-time use: clear the state cookie regardless of outcome.
	http.SetCookie(w, &http.Cookie{Name: googleStateCookie, Value: "", Path: googleStatePath, MaxAge: -1})
	if cookieErr != nil || cookie.Value == "" {
		fail("google_state_missing")
		return
	}
	state := r.URL.Query().Get("state")
	if state == "" || state != cookie.Value {
		fail("google_state_mismatch")
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		fail("google_missing_code")
		return
	}

	accessToken, err := auth.ExchangeGoogleCode(h.GoogleClientID, h.GoogleClientSecret, h.GoogleRedirectURI, code)
	if err != nil {
		log.Printf("google token exchange failed: %v", err)
		fail("google_exchange_failed")
		return
	}

	profile, err := auth.FetchGoogleUserInfo(accessToken)
	if err != nil {
		log.Printf("google userinfo fetch failed: %v", err)
		fail("google_userinfo_failed")
		return
	}
	if profile.Email == "" || !profile.EmailVerified {
		fail("google_email_unverified")
		return
	}

	email := strings.ToLower(strings.TrimSpace(profile.Email))

	user, found := h.Users.FindByGoogleID(profile.Sub)
	if !found {
		if existing, ok := h.Users.FindByEmail(email); ok {
			// Auto-link: Google has already verified this email, so treat
			// it as the same account rather than creating a duplicate.
			existing.GoogleID = profile.Sub
			if err := h.Users.Put(existing.ID, existing); err != nil {
				fail("google_internal_error")
				return
			}
			user = existing
		} else {
			user = models.User{
				ID:        idgen.New(),
				Email:     email,
				GoogleID:  profile.Sub,
				CreatedAt: time.Now(),
			}
			if err := h.Users.Put(user.ID, user); err != nil {
				fail("google_internal_error")
				return
			}
		}
	}

	if user.IsLocked() {
		fail("google_account_locked")
		return
	}

	token, err := h.mint(user)
	if err != nil {
		fail("google_internal_error")
		return
	}

	http.Redirect(w, r, h.FrontendBaseURL+"/#google_token="+url.QueryEscape(token), http.StatusFound)
}
