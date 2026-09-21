package handlers

import (
	"net/http"
	"strings"
	"time"

	"todo-app/internal/idgen"
	"todo-app/internal/middleware"
	"todo-app/internal/models"
	"todo-app/internal/storage"
)

type BoardsHandler struct {
	Boards      *storage.BoardStore
	Tasks       *storage.TaskStore
	Attachments *storage.AttachmentStore
}

func (h *BoardsHandler) List(w http.ResponseWriter, r *http.Request) {
	user, _ := middleware.UserFromContext(r.Context())
	writeJSON(w, http.StatusOK, h.Boards.ListByUser(user.ID))
}

type boardRequest struct {
	Name      string `json:"name"`
	Summary   string `json:"summary"`
	StartDate string `json:"start_date"`
}

func (h *BoardsHandler) Create(w http.ResponseWriter, r *http.Request) {
	user, _ := middleware.UserFromContext(r.Context())

	var req boardRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "invalid_name", "board name is required")
		return
	}

	now := time.Now()
	board := models.Board{
		ID:        idgen.New(),
		UserID:    user.ID,
		Name:      req.Name,
		Summary:   req.Summary,
		StartDate: req.StartDate,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := h.Boards.Put(board.ID, board); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to create board")
		return
	}
	writeJSON(w, http.StatusCreated, board)
}

func (h *BoardsHandler) Update(w http.ResponseWriter, r *http.Request) {
	user, _ := middleware.UserFromContext(r.Context())
	id := r.PathValue("id")

	board, ok := h.Boards.Get(id)
	if !ok || board.UserID != user.ID {
		writeError(w, http.StatusNotFound, "not_found", "board not found")
		return
	}

	var req boardRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "invalid_name", "board name is required")
		return
	}

	board.Name = req.Name
	board.Summary = req.Summary
	board.StartDate = req.StartDate
	board.UpdatedAt = time.Now()

	if err := h.Boards.Put(board.ID, board); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to update board")
		return
	}
	writeJSON(w, http.StatusOK, board)
}

func (h *BoardsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user, _ := middleware.UserFromContext(r.Context())
	id := r.PathValue("id")

	board, ok := h.Boards.Get(id)
	if !ok || board.UserID != user.ID {
		writeError(w, http.StatusNotFound, "not_found", "board not found")
		return
	}

	// Cascade: delete every task on this board, and every attachment on
	// each of those tasks (metadata + binary blob).
	for _, task := range h.Tasks.ListByBoardAny(board.ID) {
		for _, att := range h.Attachments.ListByTask(task.ID) {
			h.Attachments.DeleteBlob(att.ID)
			h.Attachments.Delete(att.ID)
		}
		h.Tasks.Delete(task.ID)
	}

	if err := h.Boards.Delete(board.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to delete board")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
