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

type labelsTestEnv struct {
	handler *LabelsHandler
	users   *storage.UserStore
	secret  []byte
}

func newLabelsTestEnv(t *testing.T) *labelsTestEnv {
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
	labels, err := storage.NewLabelStore(filepath.Join(dir, "labels.json"))
	if err != nil {
		t.Fatalf("NewLabelStore() error = %v", err)
	}
	return &labelsTestEnv{
		handler: &LabelsHandler{Labels: labels, Tasks: tasks, Boards: boards},
		users:   users,
		secret:  []byte("test-secret"),
	}
}

func (env *labelsTestEnv) seedUser(t *testing.T) (models.User, string) {
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

func (env *labelsTestEnv) seedBoard(t *testing.T, userID string) models.Board {
	t.Helper()
	board := models.Board{ID: idgen.New(), UserID: userID, Name: "test board", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := env.handler.Boards.Put(board.ID, board); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	return board
}

func (env *labelsTestEnv) authedRequest(method, path, body, token string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

func TestLabelsList_RequiresBoardID(t *testing.T) {
	env := newLabelsTestEnv(t)
	_, token := env.seedUser(t)
	handler := middleware.RequireAuth(env.secret, env.users)(env.handler.List)

	req := env.authedRequest(http.MethodGet, "/api/labels", "", token)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "invalid_board") {
		t.Fatalf("status = %d, body = %s, want 400 invalid_board", rec.Code, rec.Body.String())
	}
}

func TestLabelsList_RejectsBoardNotOwnedByCaller(t *testing.T) {
	env := newLabelsTestEnv(t)
	owner, _ := env.seedUser(t)
	_, otherToken := env.seedUser(t)
	board := env.seedBoard(t, owner.ID)
	handler := middleware.RequireAuth(env.secret, env.users)(env.handler.List)

	req := env.authedRequest(http.MethodGet, "/api/labels?board_id="+board.ID, "", otherToken)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "invalid_board") {
		t.Fatalf("status = %d, body = %s, want 400 invalid_board", rec.Code, rec.Body.String())
	}
}

func TestLabelsList_ReturnsOnlyThatBoardsLabels(t *testing.T) {
	env := newLabelsTestEnv(t)
	user, token := env.seedUser(t)
	boardA := env.seedBoard(t, user.ID)
	boardB := env.seedBoard(t, user.ID)

	labelA := models.Label{ID: idgen.New(), UserID: user.ID, BoardID: boardA.ID, Name: "A", Color: models.LabelColors[0], CreatedAt: time.Now()}
	labelB := models.Label{ID: idgen.New(), UserID: user.ID, BoardID: boardB.ID, Name: "B", Color: models.LabelColors[0], CreatedAt: time.Now()}
	env.handler.Labels.Put(labelA.ID, labelA)
	env.handler.Labels.Put(labelB.ID, labelB)

	handler := middleware.RequireAuth(env.secret, env.users)(env.handler.List)
	req := env.authedRequest(http.MethodGet, "/api/labels?board_id="+boardA.ID, "", token)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"A"`) || strings.Contains(rec.Body.String(), `"B"`) {
		t.Errorf("body = %s, want only board A's label", rec.Body.String())
	}
}

func TestLabelsCreate_Success(t *testing.T) {
	env := newLabelsTestEnv(t)
	user, token := env.seedUser(t)
	board := env.seedBoard(t, user.ID)
	handler := middleware.RequireAuth(env.secret, env.users)(env.handler.Create)

	body := `{"board_id":"` + board.ID + `","name":"Urgent","color":"` + models.LabelColors[0] + `"}`
	req := env.authedRequest(http.MethodPost, "/api/labels", body, token)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s, want 201", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Urgent") {
		t.Errorf("body = %s, want it to contain the label name", rec.Body.String())
	}
}

func TestLabelsCreate_RejectsEmptyName(t *testing.T) {
	env := newLabelsTestEnv(t)
	user, token := env.seedUser(t)
	board := env.seedBoard(t, user.ID)
	handler := middleware.RequireAuth(env.secret, env.users)(env.handler.Create)

	body := `{"board_id":"` + board.ID + `","name":"","color":"` + models.LabelColors[0] + `"}`
	req := env.authedRequest(http.MethodPost, "/api/labels", body, token)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "invalid_name") {
		t.Fatalf("status = %d, body = %s, want 400 invalid_name", rec.Code, rec.Body.String())
	}
}

func TestLabelsCreate_RejectsInvalidColor(t *testing.T) {
	env := newLabelsTestEnv(t)
	user, token := env.seedUser(t)
	board := env.seedBoard(t, user.ID)
	handler := middleware.RequireAuth(env.secret, env.users)(env.handler.Create)

	body := `{"board_id":"` + board.ID + `","name":"Urgent","color":"#000000"}`
	req := env.authedRequest(http.MethodPost, "/api/labels", body, token)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "invalid_color") {
		t.Fatalf("status = %d, body = %s, want 400 invalid_color", rec.Code, rec.Body.String())
	}
}

func TestLabelsCreate_RejectsBoardNotOwned(t *testing.T) {
	env := newLabelsTestEnv(t)
	owner, _ := env.seedUser(t)
	_, otherToken := env.seedUser(t)
	board := env.seedBoard(t, owner.ID)
	handler := middleware.RequireAuth(env.secret, env.users)(env.handler.Create)

	body := `{"board_id":"` + board.ID + `","name":"Urgent","color":"` + models.LabelColors[0] + `"}`
	req := env.authedRequest(http.MethodPost, "/api/labels", body, otherToken)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "invalid_board") {
		t.Fatalf("status = %d, body = %s, want 400 invalid_board", rec.Code, rec.Body.String())
	}
}

func TestLabelsUpdate_Success(t *testing.T) {
	env := newLabelsTestEnv(t)
	user, token := env.seedUser(t)
	board := env.seedBoard(t, user.ID)
	label := models.Label{ID: idgen.New(), UserID: user.ID, BoardID: board.ID, Name: "Old", Color: models.LabelColors[0], CreatedAt: time.Now()}
	env.handler.Labels.Put(label.ID, label)

	handler := middleware.RequireAuth(env.secret, env.users)(env.handler.Update)
	body := `{"name":"New","color":"` + models.LabelColors[1] + `"}`
	req := env.authedRequest(http.MethodPut, "/api/labels/"+label.ID, body, token)
	req.SetPathValue("id", label.ID)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s, want 200", rec.Code, rec.Body.String())
	}
	updated, _ := env.handler.Labels.Get(label.ID)
	if updated.Name != "New" || updated.Color != models.LabelColors[1] {
		t.Errorf("label = %+v, want name=New color=%s", updated, models.LabelColors[1])
	}
}

func TestLabelsUpdate_NotFoundOrNotOwned(t *testing.T) {
	env := newLabelsTestEnv(t)
	owner, _ := env.seedUser(t)
	_, otherToken := env.seedUser(t)
	board := env.seedBoard(t, owner.ID)
	label := models.Label{ID: idgen.New(), UserID: owner.ID, BoardID: board.ID, Name: "Old", Color: models.LabelColors[0], CreatedAt: time.Now()}
	env.handler.Labels.Put(label.ID, label)

	handler := middleware.RequireAuth(env.secret, env.users)(env.handler.Update)
	body := `{"name":"New","color":"` + models.LabelColors[1] + `"}`

	cases := []struct {
		name string
		id   string
	}{
		{name: "not owned", id: label.ID},
		{name: "unknown id", id: "does-not-exist"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			token := otherToken
			req := env.authedRequest(http.MethodPut, "/api/labels/"+tc.id, body, token)
			req.SetPathValue("id", tc.id)
			rec := httptest.NewRecorder()
			handler(rec, req)
			if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), "not_found") {
				t.Fatalf("status = %d, body = %s, want 404 not_found", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestLabelsUpdate_RejectsInvalidColor(t *testing.T) {
	env := newLabelsTestEnv(t)
	user, token := env.seedUser(t)
	board := env.seedBoard(t, user.ID)
	label := models.Label{ID: idgen.New(), UserID: user.ID, BoardID: board.ID, Name: "Old", Color: models.LabelColors[0], CreatedAt: time.Now()}
	env.handler.Labels.Put(label.ID, label)

	handler := middleware.RequireAuth(env.secret, env.users)(env.handler.Update)
	body := `{"name":"New","color":"#000000"}`
	req := env.authedRequest(http.MethodPut, "/api/labels/"+label.ID, body, token)
	req.SetPathValue("id", label.ID)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "invalid_color") {
		t.Fatalf("status = %d, body = %s, want 400 invalid_color", rec.Code, rec.Body.String())
	}
}

func TestLabelsDelete_Success_StripsFromReferencingTasks(t *testing.T) {
	env := newLabelsTestEnv(t)
	user, token := env.seedUser(t)
	board := env.seedBoard(t, user.ID)

	labelToDelete := models.Label{ID: idgen.New(), UserID: user.ID, BoardID: board.ID, Name: "Delete me", Color: models.LabelColors[0], CreatedAt: time.Now()}
	otherLabel := models.Label{ID: idgen.New(), UserID: user.ID, BoardID: board.ID, Name: "Keep me", Color: models.LabelColors[1], CreatedAt: time.Now()}
	env.handler.Labels.Put(labelToDelete.ID, labelToDelete)
	env.handler.Labels.Put(otherLabel.ID, otherLabel)

	taskWithBoth := models.Task{ID: idgen.New(), UserID: user.ID, BoardID: board.ID, Title: "t1", LabelIDs: []string{labelToDelete.ID, otherLabel.ID}, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	taskWithOnlyDeleted := models.Task{ID: idgen.New(), UserID: user.ID, BoardID: board.ID, Title: "t2", LabelIDs: []string{labelToDelete.ID}, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	env.handler.Tasks.Put(taskWithBoth.ID, taskWithBoth)
	env.handler.Tasks.Put(taskWithOnlyDeleted.ID, taskWithOnlyDeleted)

	handler := middleware.RequireAuth(env.secret, env.users)(env.handler.Delete)
	req := env.authedRequest(http.MethodDelete, "/api/labels/"+labelToDelete.ID, "", token)
	req.SetPathValue("id", labelToDelete.ID)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s, want 204", rec.Code, rec.Body.String())
	}

	if _, ok := env.handler.Labels.Get(labelToDelete.ID); ok {
		t.Error("expected the label to be deleted")
	}

	updated1, _ := env.handler.Tasks.Get(taskWithBoth.ID)
	if len(updated1.LabelIDs) != 1 || updated1.LabelIDs[0] != otherLabel.ID {
		t.Errorf("taskWithBoth.LabelIDs = %v, want only %s", updated1.LabelIDs, otherLabel.ID)
	}

	updated2, _ := env.handler.Tasks.Get(taskWithOnlyDeleted.ID)
	if len(updated2.LabelIDs) != 0 {
		t.Errorf("taskWithOnlyDeleted.LabelIDs = %v, want empty", updated2.LabelIDs)
	}
}

func TestLabelsDelete_NotFoundOrNotOwned(t *testing.T) {
	env := newLabelsTestEnv(t)
	owner, _ := env.seedUser(t)
	_, otherToken := env.seedUser(t)
	board := env.seedBoard(t, owner.ID)
	label := models.Label{ID: idgen.New(), UserID: owner.ID, BoardID: board.ID, Name: "Old", Color: models.LabelColors[0], CreatedAt: time.Now()}
	env.handler.Labels.Put(label.ID, label)

	handler := middleware.RequireAuth(env.secret, env.users)(env.handler.Delete)
	req := env.authedRequest(http.MethodDelete, "/api/labels/"+label.ID, "", otherToken)
	req.SetPathValue("id", label.ID)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), "not_found") {
		t.Fatalf("status = %d, body = %s, want 404 not_found", rec.Code, rec.Body.String())
	}
}

func TestLabelsDelete_LeavesOtherBoardsTasksUntouched(t *testing.T) {
	env := newLabelsTestEnv(t)
	user, token := env.seedUser(t)
	boardA := env.seedBoard(t, user.ID)
	boardB := env.seedBoard(t, user.ID)

	label := models.Label{ID: idgen.New(), UserID: user.ID, BoardID: boardA.ID, Name: "A-only", Color: models.LabelColors[0], CreatedAt: time.Now()}
	env.handler.Labels.Put(label.ID, label)

	taskOnBoardB := models.Task{ID: idgen.New(), UserID: user.ID, BoardID: boardB.ID, Title: "t-b", LabelIDs: []string{}, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	env.handler.Tasks.Put(taskOnBoardB.ID, taskOnBoardB)

	handler := middleware.RequireAuth(env.secret, env.users)(env.handler.Delete)
	req := env.authedRequest(http.MethodDelete, "/api/labels/"+label.ID, "", token)
	req.SetPathValue("id", label.ID)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s, want 204", rec.Code, rec.Body.String())
	}
	unchanged, _ := env.handler.Tasks.Get(taskOnBoardB.ID)
	if len(unchanged.LabelIDs) != 0 {
		t.Errorf("task on unrelated board changed: %v", unchanged.LabelIDs)
	}
}
