// Package middleware provides HTTP middleware for authentication,
// authorization, logging and panic recovery.
package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"todo-app/internal/auth"
	"todo-app/internal/models"
	"todo-app/internal/storage"
)

type ctxKey int

const userCtxKey ctxKey = 0

// UserFromContext returns the authenticated user stored in the request
// context by RequireAuth, if any.
func UserFromContext(ctx context.Context) (models.User, bool) {
	u, ok := ctx.Value(userCtxKey).(models.User)
	return u, ok
}

// RequireAuth returns a wrapper that extracts and verifies the bearer token,
// loads the corresponding user, and stores it in the request context. It
// responds 401 and does not call next if authentication fails.
func RequireAuth(secret []byte, users *storage.UserStore) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				writeUnauthorized(w)
				return
			}
			token := strings.TrimPrefix(authHeader, "Bearer ")

			claims, err := auth.Verify(secret, token)
			if err != nil {
				writeUnauthorized(w)
				return
			}

			user, ok := users.Get(claims.UserID)
			if !ok {
				writeUnauthorized(w)
				return
			}

			ctx := context.WithValue(r.Context(), userCtxKey, user)
			next(w, r.WithContext(ctx))
		}
	}
}

// RequireAdmin must be applied after RequireAuth (via composition) - it
// reads the user already stored in context and rejects non-admins with 403.
func RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := UserFromContext(r.Context())
		if !ok || !user.IsAdmin {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error":{"code":"forbidden","message":"admin access required"}}`))
			return
		}
		next(w, r)
	}
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"error":{"code":"unauthorized","message":"missing or invalid token"}}`))
}

// Logging logs method, path, status and duration for every request.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, sw.status, time.Since(start))
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// Recover turns panics in downstream handlers into 500 responses instead of
// crashing the process.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic handling %s %s: %v", r.Method, r.URL.Path, rec)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":{"code":"internal_error","message":"internal server error"}}`))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
