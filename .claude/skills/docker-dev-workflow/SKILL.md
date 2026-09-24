---
name: docker-dev-workflow
description: >
  Building, running, and testing todo-app entirely through Docker Compose -
  no native Go/Node/npm/make - plus exposing it publicly via ngrok
  (start.ps1), including the real bugs hit setting this up (air's Go version
  requirement, npm run dev exiting immediately in this container context,
  Windows bind-mount file-watching, and Vite blocking ngrok's Host header).
  Use when `docker compose up` fails, hot reload isn't picking up changes,
  ngrok exposure isn't working, or you're about to add/change a Dockerfile,
  docker-compose.yml, .air.toml, or vite.config.js.
---

# Docker dev workflow (todo-app)

## Table of Contents

- [Overview](#overview)
- [When to Use](#when-to-use)
- [Quick Start](#quick-start)
- [Known issues (hit for real in this project)](#known-issues-hit-for-real-in-this-project)
- [Best Practices](#best-practices)

## Overview

The local machine has Go 1.20.6 (too old for the Go 1.22 stdlib routing this API uses) and no `make`. Everything therefore runs via Docker Compose instead: a `golang:1.22` container (`api`, running `air` for hot reload) and a `node:20` container (`web`, running Vite's dev server), joined on Docker's default bridge network. See `../../../decisions/0001-docker-only-dev-workflow.md`.

## When to Use

- Starting/stopping the stack, running tests, or rebuilding images.
- `docker compose up` succeeds but a container exits immediately or never becomes reachable.
- Editing a file under `api/**/*.go` or `web/src/**` doesn't seem to take effect without a manual restart.
- Touching `api/Dockerfile.dev`, `web/Dockerfile.dev`, `docker-compose.yml`, `api/.air.toml`, or `web/vite.config.js`.

## Quick Start

```
docker compose up                    # full stack: api :8080, web :5173
docker compose build                 # rebuild both images
docker compose run --rm api go test ./...
docker compose run --rm api go build ./...
docker compose run --rm web npm install
docker compose logs api              # admin bootstrap password is printed here on first run
docker compose down
```

`make` isn't installed on this machine — the `Makefile` targets are wrappers around exactly the commands above; use the raw `docker compose` commands directly rather than debugging `make`.

### Exposing via ngrok

```
.\start.ps1          # brings the stack up if needed, tunnels web :5173, prints the public URL
.\start.ps1 -Prod    # same for production mode (nginx :8081, rebuilt with --build)
```

Only the `web` container needs tunneling — Vite proxies `/api/*` to `api` internally, so one URL covers the whole app including `/admin`. Ctrl+C stops only the tunnel, not the Docker stack (`docker compose down` separately for that). See `../../../decisions/0009-ngrok-for-public-exposure.md`.

## Known issues (hit for real in this project)

1. **`go install github.com/air-verse/air@latest` fails: `requires go >= 1.26.0 (running go 1.22.12)`.** The latest `air` release now targets a newer Go than this project's pinned `golang:1.22` image. Fixed by pinning an older release in `api/Dockerfile.dev`: `go install github.com/air-verse/air@v1.52.3`. If this starts failing again on a rebuild, the fix is the same shape — find an `air` version whose own `go.mod` still allows Go 1.22, don't bump the base image just to satisfy a dev tool.

2. **`docker-compose.yml`'s `web` command (`npm run dev -- --host 0.0.0.0`) exits immediately — code 0, no output, no Vite startup banner — every time, detached or not.** Confirmed by testing the identical invocation three ways: `npm run dev -- ...` (fails), `npx vite --host 0.0.0.0` (fails), and `node node_modules/vite/bin/vite.js --host 0.0.0.0` (works, stays up, serves requests). The npm/npx wrapper layers are what's failing in this container context, not Vite itself. **Fix already applied** — `web/Dockerfile.dev`'s `CMD` and `docker-compose.yml`'s `web.command` both invoke `node node_modules/vite/bin/vite.js --host 0.0.0.0 --clearScreen false` directly. If you ever "simplify" this back to `npm run dev`, it will silently break again — verify with `docker compose logs web` that you see the `VITE vX ready in Xms` banner, not just an npm banner line.

3. **Hot reload doesn't fire on file edits — neither `air` (Go) nor Vite (frontend) reacts, even though the container is running and the bind mount is correctly wired.** Docker Desktop's Windows bind mount doesn't reliably propagate native filesystem change events (inotify) into the Linux container, so any watcher relying on those events sees nothing. **Fix already applied** — both watchers use polling: `api/.air.toml` has `poll = true` / `poll_interval = 500` under `[build]`, and `web/vite.config.js` has `server.watch = { usePolling: true, interval: 500 }`. If you scaffold a new watcher/dev-server config in this project, it needs the same polling opt-in or it won't see host-side edits at all.

See `../../../decisions/0008-polling-for-windows-bind-mount-hotreload.md` for the full debugging trail behind #2 and #3.

4. **A request through the ngrok tunnel gets Vite's `Blocked request. This host is not allowed` error instead of the app.** Vite 5 rejects requests with an unrecognized `Host` header as DNS-rebinding protection, and a tunneled request's `Host` is `<random>.ngrok-free.app`, not `localhost:5173`. **Fix already applied** — `web/vite.config.js`'s `server.allowedHosts` includes ngrok's domain suffixes (`.ngrok-free.app`, etc., leading dot = any subdomain). If ngrok ever issues a domain under a suffix not already in that list, add it there.

## Best Practices

### ✅ DO
- After any change to `docker-compose.yml`, a `Dockerfile.dev`, `.air.toml`, or `vite.config.js`, verify hot reload still actually works: edit a source file and confirm `docker compose logs api` shows a rebuild (`has changed / building... / running...`) or the frontend module content updates on re-fetch — don't just confirm the containers are "Up".
- Reset test data cleanly when you need a fresh admin bootstrap: `docker compose down`, delete `api/data/{users,boards,tasks,attachments}.json` and `api/data/secret.key` (never delete `api/data/.gitkeep`), then `docker compose up -d`.
- Use `MSYS_NO_PATHCONV=1` (or an equivalent) when passing Unix-style paths to `docker` commands from Git Bash on Windows — it otherwise mangles bind-mount source paths.

### ❌ DON'T
- Don't add `golang.org/x/crypto`, a router package, or any other dependency to `api/go.mod` to work around a tooling gap — this project is stdlib-only by design (see the `go-stdlib-only-backend` skill).
- Don't revert the `web` container's command back to `npm run dev -- ...` — it's a known dead end here, not a style preference.
- Don't assume a missing hot-reload log line means the code change didn't save — check the polling config first; this has already been the actual cause twice.
