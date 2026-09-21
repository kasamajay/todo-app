package handlers

import (
	"net/http"
	"time"

	"todo-app/internal/models"
	"todo-app/internal/storage"
)

type AdminHandler struct {
	Users *storage.UserStore
}

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users := h.Users.List()
	public := make([]models.PublicUser, 0, len(users))
	for _, u := range users {
		public = append(public, u.Public())
	}
	writeJSON(w, http.StatusOK, public)
}

func (h *AdminHandler) Unlock(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	user, ok := h.Users.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "user not found")
		return
	}

	user.FailedLoginCount = 0
	user.LockedUntil = time.Time{}

	if err := h.Users.Put(user.ID, user); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to unlock user")
		return
	}
	writeJSON(w, http.StatusOK, user.Public())
}
