# 0008. File-watch polling for hot reload under Docker Desktop on Windows

## Context
Both dev containers are meant to hot-reload on source changes: `air` rebuilds and restarts the Go API, Vite serves updated frontend modules. With the default configuration, neither picked up edits made on the Windows host and bind-mounted into the containers — `air` never logged a rebuild, and while Vite happened to still serve current file content on each request, it wasn't emitting change/HMR activity either. This is a known class of issue: Docker Desktop's Windows bind mount doesn't reliably propagate native filesystem change events (inotify) into the Linux container, so watchers that rely on those events see nothing.

A second, unrelated issue surfaced during the same debugging pass: `docker-compose.yml`'s `web` service command was `npm run dev -- --host 0.0.0.0`, which exited immediately (code 0, no output, no vite startup banner) every time it ran detached/non-interactively in this container context — confirmed by testing the identical invocation via `npm run`, via `npx vite`, and finally via `node node_modules/vite/bin/vite.js` directly, where only the last one stayed running and served requests.

## Decision
- `api/.air.toml`: added `poll = true` and `poll_interval = 500` under `[build]`, so air polls the filesystem for changes instead of relying on inotify events.
- `web/vite.config.js`: added `server.watch = { usePolling: true, interval: 500 }` for the same reason.
- `web/Dockerfile.dev` and `docker-compose.yml`'s `web` command both invoke `node node_modules/vite/bin/vite.js --host 0.0.0.0 --clearScreen false` directly instead of `npm run dev -- ...`, since the npm/npx wrapper layers were the ones exiting early, not vite itself.

Verified live: editing `api/internal/models/models.go` while the stack was running triggered `air`'s "has changed / building... / running..." log sequence within ~2s; editing `web/src/theme.js` was immediately reflected when re-fetching the module from the running Vite server.

## Alternatives considered
- **WSL2-backed bind mounts / moving the repo into the WSL2 filesystem:** would likely fix inotify propagation natively, but changes the project's location/workflow beyond what was asked; polling is a self-contained fix inside the repo's own config files.
- **Leave hot reload broken, require manual `docker compose restart`:** works but defeats the point of `air`/Vite dev servers and the spec's explicit "hot-reload" requirement for `dev-api`.

## Consequences
- Polling uses more CPU than native event-based watching, though at 500ms intervals on a project this size it's negligible.
- Both watcher configs (and the Dockerfile/compose command overrides) are Windows-bind-mount-specific workarounds; they're harmless but unnecessary on a native Linux Docker host.
