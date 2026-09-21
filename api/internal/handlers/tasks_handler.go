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
	BoardID     string  `json:"board_id"`
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
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

	now := time.Now()
	task := models.Task{
		ID:          idgen.New(),
		UserID:      user.ID,
		BoardID:     board.ID,
		Title:       *req.Title,
		Description: description,
		Status:      status,
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
