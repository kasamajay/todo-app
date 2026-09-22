package handlers

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"todo-app/internal/auth"
	"todo-app/internal/idgen"
	"todo-app/internal/middleware"
	"todo-app/internal/models"
	"todo-app/internal/storage"
)

func TestParseDueDate(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		wantNil bool
		wantErr bool
	}{
		{name: "empty string means no due date", raw: "", wantNil: true},
		{name: "whitespace only means no due date", raw: "   ", wantNil: true},
		{name: "valid RFC3339", raw: "2026-09-25T00:00:00Z"},
		{name: "date-only is not valid RFC3339", raw: "2026-09-25", wantErr: true},
		{name: "garbage string", raw: "not-a-date", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseDueDate(tc.raw)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("parseDueDate(%q): expected error, got nil", tc.raw)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseDueDate(%q): unexpected error: %v", tc.raw, err)
			}
			if tc.wantNil && got != nil {
				t.Fatalf("parseDueDate(%q): expected nil, got %v", tc.raw, *got)
			}
			if !tc.wantNil && got == nil {
				t.Fatalf("parseDueDate(%q): expected non-nil result", tc.raw)
			}
		})
	}
}

type tasksTestEnv struct {
	handler *TasksHandler
	users   *storage.UserStore
	secret  []byte
}

func newTasksTestEnv(t *testing.T) *tasksTestEnv {
	t.Helper()
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
	return &tasksTestEnv{
		handler: &TasksHandler{Tasks: tasks, Boards: boards, Attachments: attachments, Labels: labels},
		users:   users,
		secret:  []byte("test-secret"),
	}
}

func (env *tasksTestEnv) seedUser(t *testing.T) (models.User, string) {
	t.Helper()
	user := models.User{ID: idgen.New(), Email: idgen.New() + "@example.com", CreatedAt: time.Now()}
	if err := env.users.Put(user.ID, user); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	token, err := auth.Mint(env.secret, auth.Claims{UserID: user.ID, ExpiresAt: time.Now().Add(time.Hour)})
	if err != nil {
		t.Fatalf("Mint() error = %v", err)
	}
	return user, token
}

func (env *tasksTestEnv) seedBoard(t *testing.T, userID string) models.Board {
	t.Helper()
	board := models.Board{ID: idgen.New(), UserID: userID, Name: "test board", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := env.handler.Boards.Put(board.ID, board); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	return board
}

func (env *tasksTestEnv) seedLabel(t *testing.T, userID, boardID, name string) models.Label {
	t.Helper()
	label := models.Label{ID: idgen.New(), UserID: userID, BoardID: boardID, Name: name, Color: models.LabelColors[0], CreatedAt: time.Now()}
	if err := env.handler.Labels.Put(label.ID, label); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	return label
}

func (env *tasksTestEnv) authedRequest(method, path, body, token string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

func TestTasksCreate_AcceptsValidLabelIDs(t *testing.T) {
	env := newTasksTestEnv(t)
	user, token := env.seedUser(t)
	board := env.seedBoard(t, user.ID)
	labelA := env.seedLabel(t, user.ID, board.ID, "A")
	labelB := env.seedLabel(t, user.ID, board.ID, "B")

	handler := middleware.RequireAuth(env.secret, env.users)(env.handler.Create)
	body := `{"board_id":"` + board.ID + `","title":"t1","label_ids":["` + labelA.ID + `","` + labelB.ID + `"]}`
	req := env.authedRequest(http.MethodPost, "/api/tasks", body, token)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s, want 201", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), labelA.ID) || !strings.Contains(rec.Body.String(), labelB.ID) {
		t.Errorf("body = %s, want it to echo both label ids", rec.Body.String())
	}
}

func TestTasksCreate_RejectsLabelFromAnotherBoard(t *testing.T) {
	env := newTasksTestEnv(t)
	user, token := env.seedUser(t)
	board := env.seedBoard(t, user.ID)
	otherBoard := env.seedBoard(t, user.ID)
	label := env.seedLabel(t, user.ID, otherBoard.ID, "elsewhere")

	handler := middleware.RequireAuth(env.secret, env.users)(env.handler.Create)
	body := `{"board_id":"` + board.ID + `","title":"t1","label_ids":["` + label.ID + `"]}`
	req := env.authedRequest(http.MethodPost, "/api/tasks", body, token)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "invalid_label") {
		t.Fatalf("status = %d, body = %s, want 400 invalid_label", rec.Code, rec.Body.String())
	}
}

func TestTasksCreate_RejectsLabelFromAnotherUser(t *testing.T) {
	env := newTasksTestEnv(t)
	user, token := env.seedUser(t)
	otherUser, _ := env.seedUser(t)
	board := env.seedBoard(t, user.ID)
	// Same board ID reused by seeding a label directly owned by otherUser
	// but pointed at user's board - simulates a forged/stale reference.
	foreignLabel := models.Label{ID: idgen.New(), UserID: otherUser.ID, BoardID: board.ID, Name: "foreign", Color: models.LabelColors[0], CreatedAt: time.Now()}
	env.handler.Labels.Put(foreignLabel.ID, foreignLabel)

	handler := middleware.RequireAuth(env.secret, env.users)(env.handler.Create)
	body := `{"board_id":"` + board.ID + `","title":"t1","label_ids":["` + foreignLabel.ID + `"]}`
	req := env.authedRequest(http.MethodPost, "/api/tasks", body, token)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "invalid_label") {
		t.Fatalf("status = %d, body = %s, want 400 invalid_label", rec.Code, rec.Body.String())
	}
}

func TestTasksCreate_RejectsUnknownLabelID(t *testing.T) {
	env := newTasksTestEnv(t)
	user, token := env.seedUser(t)
	board := env.seedBoard(t, user.ID)

	handler := middleware.RequireAuth(env.secret, env.users)(env.handler.Create)
	body := `{"board_id":"` + board.ID + `","title":"t1","label_ids":["does-not-exist"]}`
	req := env.authedRequest(http.MethodPost, "/api/tasks", body, token)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "invalid_label") {
		t.Fatalf("status = %d, body = %s, want 400 invalid_label", rec.Code, rec.Body.String())
	}
}

func TestTasksCreate_OmittedLabelIDsDefaultsEmpty(t *testing.T) {
	env := newTasksTestEnv(t)
	user, token := env.seedUser(t)
	board := env.seedBoard(t, user.ID)

	handler := middleware.RequireAuth(env.secret, env.users)(env.handler.Create)
	body := `{"board_id":"` + board.ID + `","title":"t1"}`
	req := env.authedRequest(http.MethodPost, "/api/tasks", body, token)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s, want 201", rec.Code, rec.Body.String())
	}
}

func TestTasksUpdate_ReplacesLabelIDs(t *testing.T) {
	env := newTasksTestEnv(t)
	user, token := env.seedUser(t)
	board := env.seedBoard(t, user.ID)
	labelA := env.seedLabel(t, user.ID, board.ID, "A")
	labelB := env.seedLabel(t, user.ID, board.ID, "B")

	task := models.Task{ID: idgen.New(), UserID: user.ID, BoardID: board.ID, Title: "t1", LabelIDs: []string{labelA.ID}, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	env.handler.Tasks.Put(task.ID, task)

	handler := middleware.RequireAuth(env.secret, env.users)(env.handler.Update)
	body := `{"label_ids":["` + labelB.ID + `"]}`
	req := env.authedRequest(http.MethodPut, "/api/tasks/"+task.ID, body, token)
	req.SetPathValue("id", task.ID)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s, want 200", rec.Code, rec.Body.String())
	}
	updated, _ := env.handler.Tasks.Get(task.ID)
	if len(updated.LabelIDs) != 1 || updated.LabelIDs[0] != labelB.ID {
		t.Errorf("LabelIDs = %v, want only %s", updated.LabelIDs, labelB.ID)
	}
}

func TestTasksUpdate_EmptyArrayClearsLabels(t *testing.T) {
	env := newTasksTestEnv(t)
	user, token := env.seedUser(t)
	board := env.seedBoard(t, user.ID)
	labelA := env.seedLabel(t, user.ID, board.ID, "A")

	task := models.Task{ID: idgen.New(), UserID: user.ID, BoardID: board.ID, Title: "t1", LabelIDs: []string{labelA.ID}, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	env.handler.Tasks.Put(task.ID, task)

	handler := middleware.RequireAuth(env.secret, env.users)(env.handler.Update)
	body := `{"label_ids":[]}`
	req := env.authedRequest(http.MethodPut, "/api/tasks/"+task.ID, body, token)
	req.SetPathValue("id", task.ID)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s, want 200", rec.Code, rec.Body.String())
	}
	updated, _ := env.handler.Tasks.Get(task.ID)
	if len(updated.LabelIDs) != 0 {
		t.Errorf("LabelIDs = %v, want empty", updated.LabelIDs)
	}
}

func TestTasksUpdate_OmittedLabelIDsLeavesUnchanged(t *testing.T) {
	env := newTasksTestEnv(t)
	user, token := env.seedUser(t)
	board := env.seedBoard(t, user.ID)
	labelA := env.seedLabel(t, user.ID, board.ID, "A")

	task := models.Task{ID: idgen.New(), UserID: user.ID, BoardID: board.ID, Title: "t1", LabelIDs: []string{labelA.ID}, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	env.handler.Tasks.Put(task.ID, task)

	handler := middleware.RequireAuth(env.secret, env.users)(env.handler.Update)
	body := `{"title":"t1-renamed"}`
	req := env.authedRequest(http.MethodPut, "/api/tasks/"+task.ID, body, token)
	req.SetPathValue("id", task.ID)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s, want 200", rec.Code, rec.Body.String())
	}
	updated, _ := env.handler.Tasks.Get(task.ID)
	if len(updated.LabelIDs) != 1 || updated.LabelIDs[0] != labelA.ID {
		t.Errorf("LabelIDs = %v, want unchanged [%s]", updated.LabelIDs, labelA.ID)
	}
}

func TestTasksUpdate_RejectsInvalidLabelID_NoPartialApply(t *testing.T) {
	env := newTasksTestEnv(t)
	user, token := env.seedUser(t)
	board := env.seedBoard(t, user.ID)
	labelA := env.seedLabel(t, user.ID, board.ID, "A")

	task := models.Task{ID: idgen.New(), UserID: user.ID, BoardID: board.ID, Title: "t1", LabelIDs: []string{labelA.ID}, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	env.handler.Tasks.Put(task.ID, task)

	handler := middleware.RequireAuth(env.secret, env.users)(env.handler.Update)
	body := `{"title":"should not apply","label_ids":["does-not-exist"]}`
	req := env.authedRequest(http.MethodPut, "/api/tasks/"+task.ID, body, token)
	req.SetPathValue("id", task.ID)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "invalid_label") {
		t.Fatalf("status = %d, body = %s, want 400 invalid_label", rec.Code, rec.Body.String())
	}
	unchanged, _ := env.handler.Tasks.Get(task.ID)
	if unchanged.Title != "t1" {
		t.Errorf("Title = %q, want unchanged %q (no partial apply)", unchanged.Title, "t1")
	}
}
