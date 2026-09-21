package handlers

import (
	"io"
	"net/http"
	"strconv"
	"time"

	"todo-app/internal/idgen"
	"todo-app/internal/middleware"
	"todo-app/internal/models"
	"todo-app/internal/storage"
)

const maxAttachmentBytes = 10 * 1024 * 1024 // 10MB

type AttachmentsHandler struct {
	Attachments *storage.AttachmentStore
	Tasks       *storage.TaskStore
}

// ownedTask returns the task with id owned by the authenticated user, or
// writes a 404 and returns ok=false.
func (h *AttachmentsHandler) ownedTask(w http.ResponseWriter, r *http.Request) (models.Task, bool) {
	user, _ := middleware.UserFromContext(r.Context())
	taskID := r.PathValue("id")

	task, ok := h.Tasks.Get(taskID)
	if !ok || task.UserID != user.ID {
		writeError(w, http.StatusNotFound, "not_found", "task not found")
		return models.Task{}, false
	}
	return task, true
}

func (h *AttachmentsHandler) Upload(w http.ResponseWriter, r *http.Request) {
	task, ok := h.ownedTask(w, r)
	if !ok {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxAttachmentBytes+1024) // small margin for multipart overhead
	if err := r.ParseMultipartForm(maxAttachmentBytes); err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "file_too_large", "attachment exceeds the 10MB limit")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_file", "a 'file' form field is required")
		return
	}
	defer file.Close()

	if header.Size > maxAttachmentBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "file_too_large", "attachment exceeds the 10MB limit")
		return
	}

	content, err := io.ReadAll(io.LimitReader(file, maxAttachmentBytes+1))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to read upload")
		return
	}
	if len(content) > maxAttachmentBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "file_too_large", "attachment exceeds the 10MB limit")
		return
	}

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(content)
	}

	user, _ := middleware.UserFromContext(r.Context())
	attachment := models.Attachment{
		ID:          idgen.New(),
		TaskID:      task.ID,
		UserID:      user.ID,
		Filename:    header.Filename,
		ContentType: contentType,
		SizeBytes:   int64(len(content)),
		CreatedAt:   time.Now(),
	}

	if err := h.Attachments.WriteBlob(attachment.ID, content); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to store attachment")
		return
	}
	if err := h.Attachments.Put(attachment.ID, attachment); err != nil {
		h.Attachments.DeleteBlob(attachment.ID)
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to store attachment")
		return
	}

	writeJSON(w, http.StatusCreated, attachment)
}

func (h *AttachmentsHandler) List(w http.ResponseWriter, r *http.Request) {
	task, ok := h.ownedTask(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, h.Attachments.ListByTask(task.ID))
}

func (h *AttachmentsHandler) Download(w http.ResponseWriter, r *http.Request) {
	task, ok := h.ownedTask(w, r)
	if !ok {
		return
	}

	aid := r.PathValue("aid")
	attachment, ok := h.Attachments.Get(aid)
	if !ok || attachment.TaskID != task.ID {
		writeError(w, http.StatusNotFound, "not_found", "attachment not found")
		return
	}

	content, err := h.Attachments.ReadBlob(attachment.ID)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "attachment content not found")
		return
	}

	w.Header().Set("Content-Type", attachment.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(int64(len(content)), 10))
	w.Header().Set("Content-Disposition", `inline; filename="`+attachment.Filename+`"`)
	w.WriteHeader(http.StatusOK)
	w.Write(content)
}

func (h *AttachmentsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	task, ok := h.ownedTask(w, r)
	if !ok {
		return
	}

	aid := r.PathValue("aid")
	attachment, ok := h.Attachments.Get(aid)
	if !ok || attachment.TaskID != task.ID {
		writeError(w, http.StatusNotFound, "not_found", "attachment not found")
		return
	}

	if err := h.Attachments.DeleteBlob(attachment.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to delete attachment content")
		return
	}
	if err := h.Attachments.Delete(attachment.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to delete attachment")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
