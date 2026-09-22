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

type LabelsHandler struct {
	Labels *storage.LabelStore
	Tasks  *storage.TaskStore
	Boards *storage.BoardStore
}

func (h *LabelsHandler) List(w http.ResponseWriter, r *http.Request) {
	user, _ := middleware.UserFromContext(r.Context())

	boardID := r.URL.Query().Get("board_id")
	if boardID == "" {
		writeError(w, http.StatusBadRequest, "invalid_board", "board_id is required")
		return
	}
	board, ok := h.Boards.Get(boardID)
	if !ok || board.UserID != user.ID {
		writeError(w, http.StatusBadRequest, "invalid_board", "board not found")
		return
	}

	writeJSON(w, http.StatusOK, h.Labels.ListByBoard(user.ID, boardID))
}

type labelRequest struct {
	BoardID string `json:"board_id"`
	Name    string `json:"name"`
	Color   string `json:"color"`
}

func (h *LabelsHandler) Create(w http.ResponseWriter, r *http.Request) {
	user, _ := middleware.UserFromContext(r.Context())

	var req labelRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}

	board, ok := h.Boards.Get(req.BoardID)
	if !ok || board.UserID != user.ID {
		writeError(w, http.StatusBadRequest, "invalid_board", "board not found")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "invalid_name", "label name is required")
		return
	}
	if !models.IsValidLabelColor(req.Color) {
		writeError(w, http.StatusBadRequest, "invalid_color", "color must be one of the allowed label colors")
		return
	}

	label := models.Label{
		ID:        idgen.New(),
		UserID:    user.ID,
		BoardID:   board.ID,
		Name:      req.Name,
		Color:     req.Color,
		CreatedAt: time.Now(),
	}
	if err := h.Labels.Put(label.ID, label); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to create label")
		return
	}
	writeJSON(w, http.StatusCreated, label)
}

func (h *LabelsHandler) Update(w http.ResponseWriter, r *http.Request) {
	user, _ := middleware.UserFromContext(r.Context())
	id := r.PathValue("id")

	label, ok := h.Labels.Get(id)
	if !ok || label.UserID != user.ID {
		writeError(w, http.StatusNotFound, "not_found", "label not found")
		return
	}

	var req labelRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "invalid_name", "label name is required")
		return
	}
	if !models.IsValidLabelColor(req.Color) {
		writeError(w, http.StatusBadRequest, "invalid_color", "color must be one of the allowed label colors")
		return
	}

	label.Name = req.Name
	label.Color = req.Color

	if err := h.Labels.Put(label.ID, label); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to update label")
		return
	}
	writeJSON(w, http.StatusOK, label)
}

func (h *LabelsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user, _ := middleware.UserFromContext(r.Context())
	id := r.PathValue("id")

	label, ok := h.Labels.Get(id)
	if !ok || label.UserID != user.ID {
		writeError(w, http.StatusNotFound, "not_found", "label not found")
		return
	}

	// Cascade: strip this label from every task that references it, rather
	// than deleting those tasks (decisions/0006 says cascade fully with no
	// dangling references - here that means removing the reference, since
	// the referencing resource itself shouldn't disappear).
	for _, task := range h.Tasks.ListByBoardAny(label.BoardID) {
		idx := -1
		for i, lid := range task.LabelIDs {
			if lid == label.ID {
				idx = i
				break
			}
		}
		if idx == -1 {
			continue
		}
		task.LabelIDs = append(task.LabelIDs[:idx], task.LabelIDs[idx+1:]...)
		task.UpdatedAt = time.Now()
		h.Tasks.Put(task.ID, task)
	}

	if err := h.Labels.Delete(label.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to delete label")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
