package handlers

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"todo-app/internal/auth"
	"todo-app/internal/idgen"
	"todo-app/internal/middleware"
	"todo-app/internal/models"
	"todo-app/internal/storage"
)

func TestBoardsDelete_CascadesToLabels(t *testing.T) {
	dir := t.TempDir()
	users, err := storage.NewUserStore(filepath.Join(dir, "users.json"))
	if err != nil {
		t.Fatalf("NewUserStore() error = %v", err)
	}
	boards, err := storage.NewBoardStore(filepath.Join(dir, "boards.json"))
	if err != nil {
		t.Fatalf("NewBoardStore() error = %v", err)
	}
	tasks, err := storage.NewTaskStore(filepath.Join(dir, "tasks.json"))
	if err != nil {
		t.Fatalf("NewTaskStore() error = %v", err)
	}
	attachments, err := storage.NewAttachmentStore(filepath.Join(dir, "attachments.json"), dir)
	if err != nil {
		t.Fatalf("NewAttachmentStore() error = %v", err)
	}
	labels, err := storage.NewLabelStore(filepath.Join(dir, "labels.json"))
	if err != nil {
		t.Fatalf("NewLabelStore() error = %v", err)
	}
	handler := &BoardsHandler{Boards: boards, Tasks: tasks, Attachments: attachments, Labels: labels}
	secret := []byte("test-secret")

	user := models.User{ID: idgen.New(), Email: "user@example.com", CreatedAt: time.Now()}
	users.Put(user.ID, user)
	token, err := auth.Mint(secret, auth.Claims{UserID: user.ID, ExpiresAt: time.Now().Add(time.Hour)})
	if err != nil {
		t.Fatalf("Mint() error = %v", err)
	}

	board := models.Board{ID: idgen.New(), UserID: user.ID, Name: "board", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	boards.Put(board.ID, board)

	label := models.Label{ID: idgen.New(), UserID: user.ID, BoardID: board.ID, Name: "label", Color: models.LabelColors[0], CreatedAt: time.Now()}
	labels.Put(label.ID, label)

	task := models.Task{ID: idgen.New(), UserID: user.ID, BoardID: board.ID, Title: "task", LabelIDs: []string{label.ID}, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	tasks.Put(task.ID, task)

	protected := middleware.RequireAuth(secret, users)(handler.Delete)
	req := httptest.NewRequest(http.MethodDelete, "/api/boards/"+board.ID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", board.ID)
	rec := httptest.NewRecorder()
	protected(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s, want 204", rec.Code, rec.Body.String())
	}

	if _, ok := boards.Get(board.ID); ok {
		t.Error("expected board to be deleted")
	}
	if _, ok := tasks.Get(task.ID); ok {
		t.Error("expected task to be deleted")
	}
	if _, ok := labels.Get(label.ID); ok {
		t.Error("expected label to be deleted")
	}
	if len(labels.ListByBoardAny(board.ID)) != 0 {
		t.Error("expected no labels remaining for the deleted board")
	}
}
