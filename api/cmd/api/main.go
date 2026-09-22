// Command api is the todo-app backend HTTP server.
package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"todo-app/internal/auth"
	"todo-app/internal/bootstrap"
	"todo-app/internal/handlers"
	"todo-app/internal/idgen"
	"todo-app/internal/middleware"
	"todo-app/internal/storage"
)

func main() {
	addr := getEnv("ADDR", ":8080")
	dataDir := getEnv("DATA_DIR", "./data")
	googleClientID := getEnv("GOOGLE_CLIENT_ID", "")
	googleClientSecret := getEnv("GOOGLE_CLIENT_SECRET", "")
	googleRedirectURI := getEnv("GOOGLE_REDIRECT_URI", "http://localhost:5173/api/auth/google/callback")
	frontendBaseURL := getEnv("FRONTEND_BASE_URL", "http://localhost:5173")
	if googleClientID == "" || googleClientSecret == "" {
		log.Printf("Google OAuth not configured (GOOGLE_CLIENT_ID/GOOGLE_CLIENT_SECRET unset) - Sign in with Google is disabled")
	}

	if err := os.MkdirAll(filepath.Join(dataDir, "attachments"), 0o755); err != nil {
		log.Fatalf("failed to create data directory: %v", err)
	}

	users, err := storage.NewUserStore(filepath.Join(dataDir, "users.json"))
	if err != nil {
		log.Fatalf("failed to load users store: %v", err)
	}
	boards, err := storage.NewBoardStore(filepath.Join(dataDir, "boards.json"))
	if err != nil {
		log.Fatalf("failed to load boards store: %v", err)
	}
	tasks, err := storage.NewTaskStore(filepath.Join(dataDir, "tasks.json"))
	if err != nil {
		log.Fatalf("failed to load tasks store: %v", err)
	}
	attachments, err := storage.NewAttachmentStore(filepath.Join(dataDir, "attachments.json"), dataDir)
	if err != nil {
		log.Fatalf("failed to load attachments store: %v", err)
	}

	secret, err := auth.LoadOrCreateSecret(dataDir)
	if err != nil {
		log.Fatalf("failed to load or create signing secret: %v", err)
	}

	if err := bootstrap.BootstrapAdmin(users, idgen.New); err != nil {
		log.Fatalf("failed to bootstrap admin account: %v", err)
	}

	authH := &handlers.AuthHandler{
		Users:              users,
		Secret:             secret,
		GoogleClientID:     googleClientID,
		GoogleClientSecret: googleClientSecret,
		GoogleRedirectURI:  googleRedirectURI,
		FrontendBaseURL:    frontendBaseURL,
	}
	boardsH := &handlers.BoardsHandler{Boards: boards, Tasks: tasks, Attachments: attachments}
	tasksH := &handlers.TasksHandler{Tasks: tasks, Boards: boards, Attachments: attachments}
	attachmentsH := &handlers.AttachmentsHandler{Attachments: attachments, Tasks: tasks}
	adminH := &handlers.AdminHandler{Users: users}

	requireAuth := middleware.RequireAuth(secret, users)
	protected := func(h http.HandlerFunc) http.HandlerFunc { return requireAuth(h) }
	adminOnly := func(h http.HandlerFunc) http.HandlerFunc { return requireAuth(middleware.RequireAdmin(h)) }

	mux := http.NewServeMux()

	// Public auth routes.
	mux.HandleFunc("POST /api/auth/register", authH.Register)
	mux.HandleFunc("POST /api/auth/login", authH.Login)
	mux.HandleFunc("POST /api/auth/forgot-password", authH.ForgotPassword)
	mux.HandleFunc("POST /api/auth/reset-password", authH.ResetPassword)
	mux.HandleFunc("GET /api/auth/google/login", authH.GoogleLogin)
	mux.HandleFunc("GET /api/auth/google/callback", authH.GoogleCallback)

	// Authenticated routes.
	mux.HandleFunc("POST /api/auth/logout", protected(authH.Logout))
	mux.HandleFunc("GET /api/auth/me", protected(authH.Me))

	mux.HandleFunc("GET /api/boards", protected(boardsH.List))
	mux.HandleFunc("POST /api/boards", protected(boardsH.Create))
	mux.HandleFunc("PUT /api/boards/{id}", protected(boardsH.Update))
	mux.HandleFunc("DELETE /api/boards/{id}", protected(boardsH.Delete))

	mux.HandleFunc("GET /api/tasks", protected(tasksH.List))
	mux.HandleFunc("POST /api/tasks", protected(tasksH.Create))
	mux.HandleFunc("PUT /api/tasks/{id}", protected(tasksH.Update))
	mux.HandleFunc("DELETE /api/tasks/{id}", protected(tasksH.Delete))

	mux.HandleFunc("POST /api/tasks/{id}/attachments", protected(attachmentsH.Upload))
	mux.HandleFunc("GET /api/tasks/{id}/attachments", protected(attachmentsH.List))
	mux.HandleFunc("GET /api/tasks/{id}/attachments/{aid}", protected(attachmentsH.Download))
	mux.HandleFunc("DELETE /api/tasks/{id}/attachments/{aid}", protected(attachmentsH.Delete))

	// Admin-only routes.
	mux.HandleFunc("GET /api/admin/users", adminOnly(adminH.ListUsers))
	mux.HandleFunc("POST /api/admin/users/{id}/unlock", adminOnly(adminH.Unlock))

	var h http.Handler = mux
	h = middleware.Logging(h)
	h = middleware.Recover(h)

	log.Printf("todo-app api listening on %s (data dir: %s)", addr, dataDir)
	if err := http.ListenAndServe(addr, h); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
