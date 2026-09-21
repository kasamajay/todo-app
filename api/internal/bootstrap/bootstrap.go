// Package bootstrap seeds first-run state, currently just the default admin
// account.
package bootstrap

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"time"

	"todo-app/internal/auth"
	"todo-app/internal/models"
	"todo-app/internal/storage"
)

const AdminEmail = "admin@todo.io"

// BootstrapAdmin creates the admin@todo.io account with a randomly
// generated password if no users exist yet, printing the plaintext password
// to the log exactly once (it is never stored or shown again).
func BootstrapAdmin(users *storage.UserStore, newID func() string) error {
	if len(users.List()) > 0 {
		return nil
	}

	plaintext, err := generatePassword()
	if err != nil {
		return err
	}

	hash, salt, err := auth.HashPassword(plaintext)
	if err != nil {
		return err
	}

	admin := models.User{
		ID:           newID(),
		Email:        AdminEmail,
		PasswordHash: hash,
		Salt:         salt,
		IsAdmin:      true,
		CreatedAt:    time.Now(),
	}
	if err := users.Put(admin.ID, admin); err != nil {
		return err
	}

	log.Printf("=====================================================")
	log.Printf("Bootstrapped admin account (shown only once):")
	log.Printf("  email:    %s", AdminEmail)
	log.Printf("  password: %s", plaintext)
	log.Printf("=====================================================")
	return nil
}

func generatePassword() (string, error) {
	raw := make([]byte, 18)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
