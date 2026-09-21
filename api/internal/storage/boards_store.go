package storage

import "todo-app/internal/models"

type BoardStore struct {
	*Store[models.Board]
}

func NewBoardStore(path string) (*BoardStore, error) {
	s, err := NewStore[models.Board](path)
	if err != nil {
		return nil, err
	}
	return &BoardStore{s}, nil
}

func (s *BoardStore) ListByUser(userID string) []models.Board {
	return s.Find(func(b models.Board) bool { return b.UserID == userID })
}
