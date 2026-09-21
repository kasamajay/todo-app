# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A full-stack todo app: a stdlib-only Go 1.22 API (`api/`) with custom auth (HMAC-SHA256 tokens, hand-rolled PBKDF2-SHA512 password hashing, account lockout) and JSON file storage, plus a React 18 + Vite 5 frontend (`web/`) with a drag-and-drop Kanban board and an admin panel. Everything runs via Docker Compose — there is no native Go/Node/npm/make installation on this machine, and none is required.

## Commands

Everything is a `docker compose` invocation; the `Makefile` targets are thin wrappers around the same commands (`make` isn't installed here either, so use the raw commands directly):

```
docker compose up                                   # full stack: api on :8080, web on :5173
docker compose up api                                # api only
docker compose up web                                # web only
docker compose build                                 # (re)build both images
docker compose down                                   # stop and remove containers
docker compose logs api                               # includes the admin bootstrap password on first run
.\start.ps1                                           # brings the stack up + tunnels web :5173 via ngrok (public URL)
```

Backend build/test/vet (no native Go needed):
```
docker compose run --rm api go build ./...
docker compose run --rm api go vet ./...
docker compose run --rm api go test ./...
docker compose run --rm api go test ./internal/auth/... -run TestPBKDF2Key_KnownVectors -v   # single test
docker compose run --rm api go fmt ./...
```

Frontend build/install (no native Node needed):
```
docker compose run --rm web npm install
docker compose run --rm web npm run build
```

There is no linter configured for either side beyond `go vet`.

## Git workflow

Never commit directly to `main`. For any change: create a branch, make and verify the change there, push the branch, and open a PR (`gh pr create`) — don't push straight to `main` and don't merge the PR automatically. This applies to every change in this repo, however small.

## Architecture

### Docker layer
Two services wired by `docker-compose.yml`: `api` (`golang:1.22`, running `air` for hot reload per `api/.air.toml`) and `web` (`node:20`, running Vite's dev server directly via `node node_modules/vite/bin/vite.js`, **not** `npm run dev` — that wrapper was found to exit immediately in this container context). Both watchers use polling (`poll = true` in `.air.toml`, `usePolling: true` in `vite.config.js`) because Docker Desktop's Windows bind mount doesn't propagate native filesystem change events. `api/data` is bind-mounted to the host so JSON storage, the HMAC signing secret, and uploaded attachment blobs persist across restarts. The browser only ever talks to `localhost:5173`; Vite proxies `/api/*` to `http://api:8080` (Docker's service-name DNS — not `localhost`, since Vite runs *inside* the `web` container).

### Backend (`api/`, module `todo-app`, stdlib only — zero entries in `go.mod`)
- `cmd/api/main.go` — entrypoint: loads all four JSON stores, loads/creates the HMAC secret, bootstraps the `admin@todo.io` account on an empty user store, builds the route table on Go 1.22's `http.ServeMux` (method + `{id}`-style wildcard patterns), wraps it in logging/recover middleware, listens on `:8080`.
- `internal/storage` — generic `Store[T]` (in-memory `map[string]T` + mutex, one JSON file per entity) with atomic writes (temp file in the same dir → `Sync()` → `os.Rename()`). Per-entity wrappers (`UserStore`, `BoardStore`, `TaskStore`, `AttachmentStore`) add typed query helpers (`FindByEmail`, `ListByUser`, etc.). Attachment binaries live separately under `data/attachments/<id>`, written the same atomic way.
- `internal/auth` — `pbkdf2.go` is a from-scratch RFC 8018 PBKDF2-HMAC-SHA512 implementation (Go's stdlib has no PBKDF2; it only exists in the external `golang.org/x/crypto`), tested against vectors independently generated via Python's `hashlib.pbkdf2_hmac`. `token.go` mints/verifies custom `base64url(claims).hex(hmac)` tokens (not JWT) — stateless, no server-side session store, so logout is a client-side token discard. `secret.go` loads or generates the 32-byte signing secret into `data/secret.key` on first boot.
- `internal/middleware` — `RequireAuth` (extracts + verifies the bearer token, loads the user into request context) and `RequireAdmin` (reads that context user, requires `IsAdmin`); compose them at route-registration time in `main.go` rather than via a middleware chain.
- `internal/handlers` — one file per resource (`auth_handler.go`, `boards_handler.go`, `tasks_handler.go`, `attachments_handler.go`, `admin_handler.go`), all sharing `respond.go`'s `writeJSON`/`writeError` helpers. Ownership checks return **404** (not 403) for resources that exist but aren't owned by the caller, to avoid confirming existence to non-owners; admin-only routes return 403 for non-admins instead, since gatekeeping *is* the point there.
- `internal/bootstrap` — creates `admin@todo.io` with a random password on first run only, printed to the log exactly once.

Login (`AuthHandler.Login`) always runs PBKDF2 — against the real user's salt/hash if the email is found, or a fixed dummy salt/hash if not — before any early return, so response timing can't reveal account existence. Three failed logins sets `LockedUntil` 30 minutes out on the user record. Deleting a board or task cascades to its children (tasks → attachments; boards → tasks → attachments).

### Frontend (`web/src`)
No React Router — `main.jsx` picks `App.jsx` or `admin/AdminApp.jsx` based on `window.location.pathname` (the only place the URL path is read), and each of those is a `useState`-driven view machine (`App.jsx`: `login → boards → kanban`). `api.js` is the single fetch wrapper: attaches the bearer token from `localStorage`, and on any 401 clears the token and calls a registered `onUnauthorized` handler to bounce back to login. All styling is inline `style={{...}}` objects built from `theme.js` constants (brand color `#6366f1`) — no CSS files, no UI framework. `Kanban.jsx`/`TaskCard.jsx` implement drag-and-drop with native HTML5 DnD (`draggable`, `dataTransfer`), not a library. `AttachmentUploader.jsx` fetches attachment bytes as authenticated blobs (`URL.createObjectURL`) for image previews and downloads, since `<img src>`/`<a href>` can't carry an `Authorization` header.

## Documentation

- [`decisions/`](decisions/README.md) — ADRs for the non-obvious choices (Docker-only workflow, hand-rolled PBKDF2, custom HMAC tokens, JSON atomic-write storage, Go 1.22 stdlib routing, cascading deletes, no-email reset tokens, Windows bind-mount hot-reload polling fix).
- [`diagrams/`](diagrams/README.md) — Mermaid system architecture / auth sequence / data model diagrams, plus an interactive `system-architecture.html`.
- [`features/`](features/README.md) — per-feature reference (API routes, key files, UI flow) for auth, boards, tasks/kanban, attachments, and the admin panel.
