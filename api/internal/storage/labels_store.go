package storage

import "todo-app/internal/models"

type LabelStore struct {
	*Store[models.Label]
}

func NewLabelStore(path string) (*LabelStore, error) {
	s, err := NewStore[models.Label](path)
	if err != nil {
		return nil, err
	}
	return &LabelStore{s}, nil
}

func (s *LabelStore) ListByBoard(userID, boardID string) []models.Label {
	return s.Find(func(l models.Label) bool { return l.UserID == userID && l.BoardID == boardID })
}

func (s *LabelStore) ListByBoardAny(boardID string) []models.Label {
	return s.Find(func(l models.Label) bool { return l.BoardID == boardID })
}
