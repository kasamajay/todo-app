package auth

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// GenerateSixDigitCode returns a zero-padded 6-digit numeric code for a
// one-time two-factor login challenge, using crypto/rand (not math/rand)
// since this code gates account access.
func GenerateSixDigitCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
