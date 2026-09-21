package models

import "time"

type User struct {
	ID                string    `json:"id"`
	Email             string    `json:"email"`
	PasswordHash      []byte    `json:"password_hash"`
	Salt              []byte    `json:"salt"`
	IsAdmin           bool      `json:"is_admin"`
	FailedLoginCount  int       `json:"failed_login_count"`
	LockedUntil       time.Time `json:"locked_until,omitempty"`
	ResetToken        string    `json:"reset_token,omitempty"`
	ResetTokenExpires time.Time `json:"reset_token_expires,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

// Public is the user representation safe to send to clients (no secrets).
type PublicUser struct {
	ID               string    `json:"id"`
	Email            string    `json:"email"`
	IsAdmin          bool      `json:"is_admin"`
	FailedLoginCount int       `json:"failed_login_count"`
	LockedUntil      time.Time `json:"locked_until,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

func (u User) Public() PublicUser {
	return PublicUser{
		ID:               u.ID,
		Email:            u.Email,
		IsAdmin:          u.IsAdmin,
		FailedLoginCount: u.FailedLoginCount,
		LockedUntil:      u.LockedUntil,
		CreatedAt:        u.CreatedAt,
	}
}

func (u User) IsLocked() bool {
	return time.Now().Before(u.LockedUntil)
}

type Board struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	Summary   string    `json:"summary"`
	StartDate string    `json:"start_date"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TaskStatus string

const (
	StatusTodo       TaskStatus = "todo"
	StatusInProgress TaskStatus = "in_progress"
	StatusDone       TaskStatus = "done"
)

func (s TaskStatus) Valid() bool {
	switch s {
	case StatusTodo, StatusInProgress, StatusDone:
		return true
	}
	return false
}

type Task struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	BoardID     string     `json:"board_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      TaskStatus `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type Attachment struct {
	ID          string    `json:"id"`
	TaskID      string    `json:"task_id"`
	UserID      string    `json:"user_id"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
	CreatedAt   time.Time `json:"created_at"`
}
