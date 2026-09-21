package storage

import "todo-app/internal/models"

type TaskStore struct {
	*Store[models.Task]
}

func NewTaskStore(path string) (*TaskStore, error) {
	s, err := NewStore[models.Task](path)
	if err != nil {
		return nil, err
	}
	return &TaskStore{s}, nil
}

func (s *TaskStore) ListByUser(userID string) []models.Task {
	return s.Find(func(t models.Task) bool { return t.UserID == userID })
}

func (s *TaskStore) ListByBoard(userID, boardID string) []models.Task {
	return s.Find(func(t models.Task) bool { return t.UserID == userID && t.BoardID == boardID })
}

func (s *TaskStore) ListByBoardAny(boardID string) []models.Task {
	return s.Find(func(t models.Task) bool { return t.BoardID == boardID })
}
