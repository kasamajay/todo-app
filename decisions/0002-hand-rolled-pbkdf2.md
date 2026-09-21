# 0002. Hand-rolled PBKDF2-HMAC-SHA512, no external crypto package

## Context
The spec called for "PBKDF2-SHA512 passwords (100k iterations, all stdlib)". Go's standard library has `crypto/hmac` and `crypto/sha512`, but no PBKDF2 implementation — PBKDF2 only exists in `golang.org/x/crypto/pbkdf2`, which is an external module (`go get` required), not part of the standard library, directly conflicting with the "stdlib only, no external packages" requirement.

## Decision
Implemented PBKDF2 from RFC 8018 by hand in `api/internal/auth/pbkdf2.go`, built only from `crypto/hmac`, `crypto/sha512`, and `encoding/binary`. `HashPassword`/`VerifyPassword` in `password.go` use it with a 16-byte random salt, 100,000 iterations, and a 64-byte (SHA-512 output size) derived key.

Correctness is proven independently: `pbkdf2_test.go` checks the output against PBKDF2-HMAC-SHA512 test vectors generated locally with Python's stdlib `hashlib.pbkdf2_hmac` — a separate, trusted implementation — rather than trusting memorized hex values.

## Alternatives considered
- **`golang.org/x/crypto/pbkdf2`:** the standard way to get PBKDF2 in Go, but it's an external module and would violate "stdlib only".
- **bcrypt/scrypt instead of PBKDF2:** also only available via `golang.org/x/crypto`, and the spec explicitly named PBKDF2-SHA512.

## Consequences
- `go.mod` has zero dependencies — `go mod download` is a no-op and there's no `go.sum` to manage.
- The PBKDF2 implementation is project-owned code that must be trusted and maintained like any other crypto primitive; its test suite is the safety net.
