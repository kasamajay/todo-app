package models

import "time"

type User struct {
	ID                string    `json:"id"`
	Email             string    `json:"email"`
	PasswordHash      []byte    `json:"password_hash"`
	Salt              []byte    `json:"salt"`
	GoogleID          string    `json:"google_id,omitempty"`
	IsAdmin           bool      `json:"is_admin"`
	FailedLoginCount  int       `json:"failed_login_count"`
	LockedUntil       time.Time `json:"locked_until,omitempty"`
	ResetToken        string    `json:"reset_token,omitempty"`
	ResetTokenExpires time.Time `json:"reset_token_expires,omitempty"`

	// Two-factor auth: opt-in per account (see decisions/0011). TwoFactorCode
	// et al. hold a pending login challenge and are cleared once verified,
	// expired, or invalidated after too many wrong attempts.
	TwoFactorEnabled     bool      `json:"two_factor_enabled"`
	TwoFactorCode        string    `json:"two_factor_code,omitempty"`
	TwoFactorCodeExpires time.Time `json:"two_factor_code_expires,omitempty"`
	TwoFactorAttempts    int       `json:"two_factor_attempts,omitempty"`

	CreatedAt time.Time `json:"created_at"`
}

// Public is the user representation safe to send to clients (no secrets).
type PublicUser struct {
	ID               string    `json:"id"`
	Email            string    `json:"email"`
	IsAdmin          bool      `json:"is_admin"`
	FailedLoginCount int       `json:"failed_login_count"`
	LockedUntil      time.Time `json:"locked_until,omitempty"`
	TwoFactorEnabled bool      `json:"two_factor_enabled"`
	CreatedAt        time.Time `json:"created_at"`
}

func (u User) Public() PublicUser {
	return PublicUser{
		ID:               u.ID,
		Email:            u.Email,
		IsAdmin:          u.IsAdmin,
		FailedLoginCount: u.FailedLoginCount,
		LockedUntil:      u.LockedUntil,
		TwoFactorEnabled: u.TwoFactorEnabled,
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

type Label struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	BoardID   string    `json:"board_id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"created_at"`
}

// LabelColors is the fixed palette labels may use. web/src/theme.js's
// labelColors mirrors this exact list (same hex values, same order) -
// kept in sync manually since there's no shared config between the Go API
// and the JS frontend.
var LabelColors = []string{
	"#6366f1", // brand indigo
	"#ef4444", // danger red
	"#f59e0b", // warning amber
	"#16a34a", // success green
	"#0ea5e9", // blue
	"#8b5cf6", // purple
	"#ec4899", // pink
	"#6b7280", // gray
}

func IsValidLabelColor(c string) bool {
	for _, v := range LabelColors {
		if v == c {
			return true
		}
	}
	return false
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
	DueDate     *time.Time `json:"due_date,omitempty"`
	LabelIDs    []string   `json:"label_ids,omitempty"`
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
