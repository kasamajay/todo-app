package storage

import (
	"strings"

	"todo-app/internal/models"
)

type UserStore struct {
	*Store[models.User]
}

func NewUserStore(path string) (*UserStore, error) {
	s, err := NewStore[models.User](path)
	if err != nil {
		return nil, err
	}
	return &UserStore{s}, nil
}

func (s *UserStore) FindByEmail(email string) (models.User, bool) {
	email = strings.ToLower(strings.TrimSpace(email))
	for _, u := range s.List() {
		if strings.ToLower(u.Email) == email {
			return u, true
		}
	}
	return models.User{}, false
}

func (s *UserStore) FindByResetToken(token string) (models.User, bool) {
	if token == "" {
		return models.User{}, false
	}
	for _, u := range s.List() {
		if u.ResetToken == token {
			return u, true
		}
	}
	return models.User{}, false
}
