package auth

import (
	"encoding/hex"
	"testing"
)

// Test vectors independently generated via Python's stdlib
// hashlib.pbkdf2_hmac("sha512", password, salt, iterations, dklen) to avoid
// depending on any external Go crypto library for verification.
func TestPBKDF2Key_KnownVectors(t *testing.T) {
	cases := []struct {
		name       string
		password   string
		salt       string
		iterations int
		keyLen     int
		wantHex    string
	}{
		{
			name:       "iterations=1",
			password:   "password",
			salt:       "salt",
			iterations: 1,
			keyLen:     64,
			wantHex:    "867f70cf1ade02cff3752599a3a53dc4af34c7a669815ae5d513554e1c8cf252c02d470a285a0501bad999bfe943c08f050235d7d68b1da55e63f73b60a57fce",
		},
		{
			name:       "iterations=2",
			password:   "password",
			salt:       "salt",
			iterations: 2,
			keyLen:     64,
			wantHex:    "e1d9c16aa681708a45f5c7c4e215ceb66e011a2e9f0040713f18aefdb866d53cf76cab2868a39b9f7840edce4fef5a82be67335c77a6068e04112754f27ccf4e",
		},
		{
			name:       "iterations=4096",
			password:   "password",
			salt:       "salt",
			iterations: 4096,
			keyLen:     64,
			wantHex:    "d197b1b33db0143e018b12f3d1d1479e6cdebdcc97c5c0f87f6902e072f457b5143f30602641b3d55cd335988cb36b84376060ecd532e039b742a239434af2d5",
		},
		{
			name:       "long password and salt",
			password:   "passwordPASSWORDpassword",
			salt:       "saltSALTsaltSALTsaltSALTsaltSALTsalt",
			iterations: 4096,
			keyLen:     64,
			wantHex:    "8c0511f4c6e597c6ac6315d8f0362e225f3c501495ba23b868c005174dc4ee71115b59f9e60cd9532fa33e0f75aefe30225c583a186cd82bd4daea9724a3d3b8",
		},
		{
			name:       "embedded null bytes",
			password:   "pass\x00word",
			salt:       "sa\x00lt",
			iterations: 4096,
			keyLen:     64,
			wantHex:    "9d9e9c4cd21fe4be24d5b8244c759665f39d98fc12a9ca759bb021db3cfadf345844aebe70dd8b2f6966f25f3613e1187bbd24ed2ca43ed13b246e4675be7ab9",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := PBKDF2Key([]byte(tc.password), []byte(tc.salt), tc.iterations, tc.keyLen)
			gotHex := hex.EncodeToString(got)
			if gotHex != tc.wantHex {
				t.Errorf("PBKDF2Key() = %s, want %s", gotHex, tc.wantHex)
			}
		})
	}
}

func TestPBKDF2Key_DeterministicAndDistinct(t *testing.T) {
	a := PBKDF2Key([]byte("pw"), []byte("salt1"), 1000, 64)
	b := PBKDF2Key([]byte("pw"), []byte("salt1"), 1000, 64)
	if hex.EncodeToString(a) != hex.EncodeToString(b) {
		t.Error("PBKDF2Key should be deterministic for identical inputs")
	}

	c := PBKDF2Key([]byte("pw"), []byte("salt2"), 1000, 64)
	if hex.EncodeToString(a) == hex.EncodeToString(c) {
		t.Error("different salts should produce different keys")
	}
}

func TestPBKDF2Key_ShortKeyLen(t *testing.T) {
	// keyLen smaller than the SHA-512 block size exercises the truncation path.
	got := PBKDF2Key([]byte("password"), []byte("salt"), 1, 16)
	if len(got) != 16 {
		t.Fatalf("expected 16 bytes, got %d", len(got))
	}
	full := PBKDF2Key([]byte("password"), []byte("salt"), 1, 64)
	if hex.EncodeToString(got) != hex.EncodeToString(full[:16]) {
		t.Error("short keyLen should be a prefix of the full-length derivation")
	}
}
