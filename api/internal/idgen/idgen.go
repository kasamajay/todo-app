// Package idgen generates random, URL-safe unique identifiers for entities.
package idgen

import (
	"crypto/rand"
	"encoding/hex"
)

// New returns a random 32-character hex identifier.
func New() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand.Read failing is effectively unrecoverable on any
		// supported platform; panicking surfaces it immediately.
		panic(err)
	}
	return hex.EncodeToString(b)
}
