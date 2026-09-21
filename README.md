# Todo App

Full-stack todo app: a stdlib-only Go 1.22 API with custom auth (HMAC tokens,
hand-rolled PBKDF2-SHA512 password hashing, account lockout) and JSON file
storage, plus a React 18 + Vite 5 frontend with a drag-and-drop Kanban board
and an admin panel.

Everything runs via **Docker Desktop** - no native Go, Node, npm, or make
installation is required or used.

## Quick start

```
docker compose up
```

- API: http://localhost:8080
- Web app: http://localhost:5173
- Admin panel: http://localhost:5173/admin

On first boot the API creates an `admin@todo.io` account with a random
password and **prints it once** to the API container logs:

```
docker compose logs api
```

Look for a block like:

```
Bootstrapped admin account (shown only once):
  email:    admin@todo.io
  password: <random>
```

Save it - it is never shown again (though you can always reset it via
"Forgot password?" on the admin login, since the reset token is logged
server-side - see below).

## Public access via ngrok

To reach the app from another device, or share it with someone, expose it via [ngrok](https://ngrok.com) (requires `ngrok` installed and authenticated: `ngrok config add-authtoken <token>`):

```
.\start.ps1
```

This brings the Docker Compose stack up if it isn't already running, tunnels the `web` container's port (5173), and prints a public HTTPS URL. Only that one port needs tunneling — Vite proxies `/api/*` to the `api` container internally, so the same URL serves the whole app, including `<url>/admin`. Ctrl+C stops the tunnel only; the Docker stack keeps running (`docker compose down` to stop that separately). See [`decisions/0009-ngrok-for-public-exposure.md`](decisions/0009-ngrok-for-public-exposure.md).

## Makefile / raw docker compose commands

A `Makefile` is provided, but `make` isn't required - every target is a thin
wrapper you can run directly:

| Make target      | Equivalent command                              |
|------------------|---------------------------------------------------|
| `install-tools`  | `docker compose build`                             |
| `dev-api`        | `docker compose up api`                             |
| `dev-web`        | `docker compose up web`                             |
| `dev`            | `docker compose up`                                 |
| `test`           | `docker compose run --rm api go test ./...`         |
| `build`          | `docker compose build`                              |
| `down`           | `docker compose down`                               |

## Project layout

```
api/          Go 1.22 module "todo-app" (stdlib only), JSON file storage in api/data/
web/          React 18 + Vite 5 frontend, inline styles only, brand color #6366f1
decisions/    ADRs - why things are built the way they are
diagrams/     Mermaid + one interactive HTML diagram of the system architecture
features/     What each feature does - API routes, key files, UI flow
```

See inline package docs in `api/internal/*` for line-level architecture details.

## Documentation

- [`decisions/`](decisions/README.md) - architecture decision records: why Docker-only dev, why hand-rolled PBKDF2, why custom HMAC tokens, why cascading deletes, etc.
- [`diagrams/`](diagrams/README.md) - system architecture, auth request sequence, and data model diagrams (Mermaid `.md` + one interactive `system-architecture.html`).
- [`features/`](features/README.md) - what each feature area does: authentication, boards, tasks/kanban, attachments, admin panel.

## API

| Method | Path | Auth |
|---|---|---|
| POST | `/api/auth/register` | none |
| POST | `/api/auth/login` | none |
| POST | `/api/auth/logout` | bearer |
| GET | `/api/auth/me` | bearer |
| POST | `/api/auth/forgot-password` | none |
| POST | `/api/auth/reset-password` | none |
| GET/POST | `/api/boards` | bearer |
| PUT/DELETE | `/api/boards/{id}` | bearer (owner) |
| GET/POST | `/api/tasks` | bearer |
| PUT/DELETE | `/api/tasks/{id}` | bearer (owner) |
| POST/GET | `/api/tasks/{id}/attachments` | bearer (owner) |
| GET/DELETE | `/api/tasks/{id}/attachments/{aid}` | bearer (owner) |
| GET | `/api/admin/users` | bearer + admin |
| POST | `/api/admin/users/{id}/unlock` | bearer + admin |

## Running tests

```
docker compose run --rm api go test ./... -v
```
