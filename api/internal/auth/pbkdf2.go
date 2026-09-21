// Package auth implements password hashing and API tokens using only the Go
// standard library. PBKDF2 itself is not part of the standard library (it
// only exists in the external golang.org/x/crypto module), so PBKDF2Key
// implements RFC 8018 directly on top of crypto/hmac and crypto/sha512.
package auth

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/binary"
	"hash"
)

// PBKDF2Key derives a keyLen-byte key from password and salt using PBKDF2
// with HMAC-SHA512 as the pseudorandom function (RFC 8018).
func PBKDF2Key(password, salt []byte, iterations, keyLen int) []byte {
	prf := hmac.New(sha512.New, password)
	hLen := prf.Size()

	numBlocks := (keyLen + hLen - 1) / hLen
	dk := make([]byte, 0, numBlocks*hLen)

	blockIndex := make([]byte, 4)
	for block := 1; block <= numBlocks; block++ {
		binary.BigEndian.PutUint32(blockIndex, uint32(block))

		u := hmacSum(prf, salt, blockIndex)
		t := make([]byte, hLen)
		copy(t, u)

		for i := 1; i < iterations; i++ {
			u = hmacSum(prf, u)
			for j := range t {
				t[j] ^= u[j]
			}
		}
		dk = append(dk, t...)
	}
	return dk[:keyLen]
}

func hmacSum(prf hash.Hash, parts ...[]byte) []byte {
	prf.Reset()
	for _, p := range parts {
		prf.Write(p)
	}
	return prf.Sum(nil)
}
