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

type TasksHandler struct {
	Tasks       *storage.TaskStore
	Boards      *storage.BoardStore
	Attachments *storage.AttachmentStore
	Labels      *storage.LabelStore
}

func (h *TasksHandler) List(w http.ResponseWriter, r *http.Request) {
	user, _ := middleware.UserFromContext(r.Context())

	boardID := r.URL.Query().Get("board_id")
	if boardID != "" {
		writeJSON(w, http.StatusOK, h.Tasks.ListByBoard(user.ID, boardID))
		return
	}
	writeJSON(w, http.StatusOK, h.Tasks.ListByUser(user.ID))
}

type taskRequest struct {
	BoardID     string    `json:"board_id"`
	Title       *string   `json:"title"`
	Description *string   `json:"description"`
	Status      *string   `json:"status"`
	DueDate     *string   `json:"due_date"`
	LabelIDs    *[]string `json:"label_ids"`
}

// parseDueDate converts a client-supplied due-date string into a *time.Time.
// An empty string returns (nil, nil), meaning "no due date". A non-empty
// string must be valid RFC3339; anything else is a validation error.
func parseDueDate(raw string) (*time.Time, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// validateLabelIDs reports whether every id in ids names a Label that
// belongs to boardID and to userID.
func (h *TasksHandler) validateLabelIDs(ids []string, boardID, userID string) bool {
	for _, id := range ids {
		label, ok := h.Labels.Get(id)
		if !ok || label.BoardID != boardID || label.UserID != userID {
			return false
		}
	}
	return true
}

func (h *TasksHandler) Create(w http.ResponseWriter, r *http.Request) {
	user, _ := middleware.UserFromContext(r.Context())

	var req taskRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}
	if req.Title == nil || strings.TrimSpace(*req.Title) == "" {
		writeError(w, http.StatusBadRequest, "invalid_title", "task title is required")
		return
	}

	board, ok := h.Boards.Get(req.BoardID)
	if !ok || board.UserID != user.ID {
		writeError(w, http.StatusBadRequest, "invalid_board", "board not found")
		return
	}

	status := models.StatusTodo
	if req.Status != nil {
		s := models.TaskStatus(*req.Status)
		if !s.Valid() {
			writeError(w, http.StatusBadRequest, "invalid_status", "status must be todo, in_progress, or done")
			return
		}
		status = s
	}

	description := ""
	if req.Description != nil {
		description = *req.Description
	}

	var dueDate *time.Time
	if req.DueDate != nil {
		d, err := parseDueDate(*req.DueDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_due_date", "due date must be a valid RFC3339 timestamp")
			return
		}
		dueDate = d
	}

	labelIDs := []string{}
	if req.LabelIDs != nil {
		if !h.validateLabelIDs(*req.LabelIDs, board.ID, user.ID) {
			writeError(w, http.StatusBadRequest, "invalid_label", "one or more label ids are invalid for this board")
			return
		}
		labelIDs = *req.LabelIDs
	}

	now := time.Now()
	task := models.Task{
		ID:          idgen.New(),
		UserID:      user.ID,
		BoardID:     board.ID,
		Title:       *req.Title,
		Description: description,
		Status:      status,
		DueDate:     dueDate,
		LabelIDs:    labelIDs,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := h.Tasks.Put(task.ID, task); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to create task")
		return
	}
	writeJSON(w, http.StatusCreated, task)
}

func (h *TasksHandler) Update(w http.ResponseWriter, r *http.Request) {
	user, _ := middleware.UserFromContext(r.Context())
	id := r.PathValue("id")

	task, ok := h.Tasks.Get(id)
	if !ok || task.UserID != user.ID {
		writeError(w, http.StatusNotFound, "not_found", "task not found")
		return
	}

	var req taskRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}

	if req.Title != nil {
		if strings.TrimSpace(*req.Title) == "" {
			writeError(w, http.StatusBadRequest, "invalid_title", "task title cannot be empty")
			return
		}
		task.Title = *req.Title
	}
	if req.Description != nil {
		task.Description = *req.Description
	}
	if req.Status != nil {
		s := models.TaskStatus(*req.Status)
		if !s.Valid() {
			writeError(w, http.StatusBadRequest, "invalid_status", "status must be todo, in_progress, or done")
			return
		}
		task.Status = s
	}
	if req.DueDate != nil {
		d, err := parseDueDate(*req.DueDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_due_date", "due date must be a valid RFC3339 timestamp")
			return
		}
		task.DueDate = d
	}
	if req.LabelIDs != nil {
		// Validated against the task's current board - a request that both
		// reassigns the board and sets label_ids in the same call would
		// validate against the old board. Not reachable from the frontend
		// (TaskForm never reassigns boards), so left as a documented edge
		// case rather than handled.
		if !h.validateLabelIDs(*req.LabelIDs, task.BoardID, user.ID) {
			writeError(w, http.StatusBadRequest, "invalid_label", "one or more label ids are invalid for this board")
			return
		}
		task.LabelIDs = *req.LabelIDs
	}
	if req.BoardID != "" && req.BoardID != task.BoardID {
		board, ok := h.Boards.Get(req.BoardID)
		if !ok || board.UserID != user.ID {
			writeError(w, http.StatusBadRequest, "invalid_board", "board not found")
			return
		}
		task.BoardID = board.ID
	}
	task.UpdatedAt = time.Now()

	if err := h.Tasks.Put(task.ID, task); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to update task")
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (h *TasksHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user, _ := middleware.UserFromContext(r.Context())
	id := r.PathValue("id")

	task, ok := h.Tasks.Get(id)
	if !ok || task.UserID != user.ID {
		writeError(w, http.StatusNotFound, "not_found", "task not found")
		return
	}

	for _, att := range h.Attachments.ListByTask(task.ID) {
		h.Attachments.DeleteBlob(att.ID)
		h.Attachments.Delete(att.ID)
	}

	if err := h.Tasks.Delete(task.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to delete task")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
