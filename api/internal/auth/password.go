package auth

import (
	"crypto/rand"
	"crypto/subtle"
)

const (
	pbkdf2Iterations = 100_000
	pbkdf2KeyLen     = 64 // SHA-512 output size
	saltLen          = 16
)

// HashPassword generates a random salt and derives a PBKDF2-HMAC-SHA512 hash
// of plaintext, suitable for persisting on a User record.
func HashPassword(plaintext string) (hash, salt []byte, err error) {
	salt = make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, nil, err
	}
	hash = PBKDF2Key([]byte(plaintext), salt, pbkdf2Iterations, pbkdf2KeyLen)
	return hash, salt, nil
}

// VerifyPassword recomputes the PBKDF2 hash for plaintext with the given
// salt and compares it against hash in constant time.
func VerifyPassword(plaintext string, hash, salt []byte) bool {
	computed := PBKDF2Key([]byte(plaintext), salt, pbkdf2Iterations, pbkdf2KeyLen)
	return subtle.ConstantTimeCompare(computed, hash) == 1
}
