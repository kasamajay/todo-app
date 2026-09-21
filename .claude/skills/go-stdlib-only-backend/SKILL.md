---
name: go-stdlib-only-backend
description: >
  The stdlib-only discipline for api/ (Go 1.22, module "todo-app", zero
  entries in go.mod) - hand-rolled PBKDF2 password hashing, custom
  HMAC-SHA256 tokens instead of JWT, Go 1.22's native http.ServeMux routing,
  and the atomic JSON storage pattern. Use before adding any backend
  dependency, touching auth/token code, or adding a new API route.
---

# Go stdlib-only backend (todo-app api/)

## Table of Contents

- [Overview](#overview)
- [When to Use](#when-to-use)
- [This project's structure](#this-projects-structure)
- [Conventions to follow](#conventions-to-follow)
- [Best Practices](#best-practices)

## Overview

`api/go.mod` has **zero dependencies** — every line of `api/` is written against the Go standard library only, by explicit project requirement. This is not an accident or an oversight to "fix" by adding a package; it's the whole point of several pieces of this codebase. See `../../../decisions/0002-hand-rolled-pbkdf2.md`, `0003-custom-hmac-tokens-not-jwt.md`, and `0005-go122-stdlib-routing.md`.

## When to Use

- You're about to `go get` anything, or add an `import` that isn't in the standard library.
- Touching password hashing/verification, token minting/verification, or login/lockout logic.
- Adding a new API route or resource.
- Adding a new persisted entity.

## This project's structure

```
api/cmd/api/main.go            entrypoint: stores, secret, admin bootstrap, route table, listen
api/internal/models/           User, Board, Task, Attachment structs
api/internal/storage/          generic Store[T] (atomic JSON writes) + per-entity stores
api/internal/auth/             pbkdf2.go, password.go, token.go, secret.go
api/internal/middleware/       RequireAuth, RequireAdmin, Logging, Recover
api/internal/handlers/         one file per resource, sharing respond.go's writeJSON/writeError
api/internal/bootstrap/        first-run admin@todo.io creation
```

## Conventions to follow

- **No external packages, ever.** If a task seems to need one (crypto primitive, router, UUID generator, etc.), check whether the standard library already covers it, or whether it should be hand-implemented like `internal/auth/pbkdf2.go` (RFC 8018 PBKDF2-HMAC-SHA512 built from `crypto/hmac` + `crypto/sha512` only — Go's stdlib has no PBKDF2 package). If you genuinely believe an exception is warranted, raise it with the user first; don't silently `go get` something.
- **Tokens are custom HMAC-SHA256, not JWT**: `base64url(json claims).hex(HMAC-SHA256(claims, secret))`, minted/verified in `internal/auth/token.go`. They're stateless — there's no session store, so `POST /api/auth/logout` is a 204 no-op; don't add a "revoke this token" feature without first re-reading `decisions/0003` on why that was left out.
- **Routing uses Go 1.22's native `http.ServeMux`** method+wildcard patterns directly in `cmd/api/main.go` (e.g. `mux.HandleFunc("PUT /api/boards/{id}", ...)`, `r.PathValue("id")`) — don't reach for a router package; the stdlib already does this as of 1.22.
- **New persisted entities** get a `models` struct, a `storage.Store[T]`-backed wrapper (see `boards_store.go` for the minimal shape, `attachments_store.go` for one with a binary blob alongside), and a JSON file under `data/`. Every mutation goes through `Store.Put`/`Delete`, which handles the atomic temp-file-then-rename write — never write to a `data/*.json` file directly.
- **Ownership vs. admin-gating status codes differ on purpose**: a board/task that exists but isn't owned by the caller returns **404** (hides existence from non-owners); an admin-only route hit by a non-admin returns **403** (the point there is enforcing the gate, not hiding it). Match whichever pattern fits when adding a new protected route.
- **Password verification is timing-safe by construction**: `AuthHandler.Login` always calls `auth.PBKDF2Key` — against the real user's salt/hash if found, or a fixed dummy salt/hash if not — before any branch that could return early. If you touch this function, keep that ordering; moving the "email not found" check before the PBKDF2 call re-introduces a timing side-channel.

## Best Practices

### ✅ DO
- Run `docker compose run --rm api go test ./...` after any change under `internal/auth/` — `pbkdf2_test.go` and `token_test.go` are the safety net for the hand-rolled crypto.
- When adding a PBKDF2 test vector, generate it from an independent implementation (e.g. Python's `hashlib.pbkdf2_hmac`) rather than hand-computing or trusting a remembered value — that's how the existing vectors in `pbkdf2_test.go` were produced and verified.
- Keep `respond.go`'s `writeJSON`/`writeError` as the only way handlers write HTTP responses, for a consistent `{"error": {"code", "message"}}` envelope.

### ❌ DON'T
- Don't add `golang.org/x/crypto` (or any module) for password hashing, even though it's the "normal" way to get PBKDF2/bcrypt/scrypt in Go — that's exactly what this project deliberately avoids.
- Don't introduce a JWT library — the spec and `decisions/0003` are explicit about custom HMAC tokens.
- Don't return a distinct "no such account" error from `Login` — both "email not found" and "wrong password" must produce the same generic `401 invalid_credentials`.
