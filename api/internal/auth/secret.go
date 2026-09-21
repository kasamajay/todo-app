package auth

import (
	"crypto/rand"
	"os"
	"path/filepath"

	"todo-app/internal/storage"
)

const secretLen = 32

// LoadOrCreateSecret reads the HMAC signing secret from
// <dataDir>/secret.key, generating and persisting a new random one on first
// run so tokens remain valid across restarts.
func LoadOrCreateSecret(dataDir string) ([]byte, error) {
	path := filepath.Join(dataDir, "secret.key")

	if existing, err := os.ReadFile(path); err == nil {
		if len(existing) == secretLen {
			return existing, nil
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	secret := make([]byte, secretLen)
	if _, err := rand.Read(secret); err != nil {
		return nil, err
	}
	if err := storage.WriteFileAtomic(path, secret); err != nil {
		return nil, err
	}
	return secret, nil
}
