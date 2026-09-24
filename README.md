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

## Production mode

Dev mode (above) runs Vite's dev server, which compiles JSX on the fly, plus
`air` hot-reloading the API. Production mode instead compiles and bundles the
frontend at image build time and serves it as static files from **nginx** -
no Node process runs at all:

```
docker compose -f docker-compose.prod.yml up -d --build
```

- Web app: http://localhost:8081
- Admin panel: http://localhost:8081/admin
- Stop: `docker compose -f docker-compose.prod.yml down`

```
browser -> nginx :8081 --+-- /assets/*, index.html  (prebuilt Vite bundle)
                         +-- /api/*  -> api :8080   (compiled Go binary, not exposed on the host)
```

- `web/Dockerfile` runs `vite build` in a throwaway `node:20` stage and copies
  only `dist/` into `nginx:alpine`; `web/nginx.conf` does the `/api` proxy,
  the `/admin` -> `index.html` fallback, gzip, and caching (hashed
  `/assets/*` cached forever, `index.html` always revalidated).
- `api/Dockerfile` builds a static binary onto `alpine` - no Go toolchain or `air`.
- It runs under its own Compose project (`todo-app-prod`), so the dev stack is
  untouched, but it **shares `api/data`** with dev - same users and tasks. Don't
  run both stacks at once (two processes writing the same JSON files).
- Google sign-in in prod: add `http://localhost:8081/api/auth/google/callback`
  as an extra Authorized redirect URI in Google Cloud Console. Override the
  prod URLs with `PROD_GOOGLE_REDIRECT_URI` / `PROD_FRONTEND_BASE_URL` in `.env`.
- Frontend changes need a rebuild (`--build`) - there is no hot reload.

See [`decisions/0013`](decisions/0013-production-mode-nginx-static-bundle.md), and
[`diagrams/dev-vs-prod-serving.md`](diagrams/dev-vs-prod-serving.md) for a
walkthrough of how each mode serves the frontend.

## Public access via ngrok

To reach the app from another device, or share it with someone, expose it via [ngrok](https://ngrok.com) (requires `ngrok` installed and authenticated: `ngrok config add-authtoken <token>`):

```
.\start.ps1
```

To tunnel **production mode** instead (nginx on 8081, rebuilt with `--build` each run):

```
.\start.ps1 -Prod
```

The script refuses to start one stack while the other is running, because they share `api/data`, and prints the `down` command to run first. For Google sign-in over a static ngrok domain in prod, set `PROD_GOOGLE_REDIRECT_URI=https://<domain>/api/auth/google/callback` and `PROD_FRONTEND_BASE_URL=https://<domain>` in `.env`. The dev `GOOGLE_REDIRECT_URI`/`FRONTEND_BASE_URL` don't apply to prod, but the callback URL registered in Google Console is the same.

In dev mode, this brings the Docker Compose stack up if it isn't already running, tunnels the `web` container's port (5173), and prints a public HTTPS URL. Only that one port needs tunneling — Vite proxies `/api/*` to the `api` container internally, so the same URL serves the whole app, including `<url>/admin`. Ctrl+C stops the tunnel only; the Docker stack keeps running (`docker compose down` to stop that separately). See [`decisions/0009-ngrok-for-public-exposure.md`](decisions/0009-ngrok-for-public-exposure.md).

## Sign in with Google (optional)

The login screen has a "Sign in with Google" button. It's disabled by default
(clicking it returns an error) until you configure your own OAuth client -
this app has no shared credentials, and Google requires each app to register
its own:

1. https://console.cloud.google.com/apis/credentials → select or create a project.
2. **OAuth consent screen** → User type "External", fill in the required
   fields, leave it in **Testing** mode (add your own Google account as a
   test user if prompted - no verification needed for personal use).
3. **Credentials → Create Credentials → OAuth client ID** → Application type
   **Web application**.
4. **Authorized redirect URIs** → add exactly
   `http://localhost:5173/api/auth/google/callback`.
5. Copy the generated **Client ID** and **Client secret**.
6. `cp .env.example .env`, then fill in `GOOGLE_CLIENT_ID=` and
   `GOOGLE_CLIENT_SECRET=` with those values.
7. `docker compose up` (or restart the `api` service if it's already
   running) to pick up the new env vars.

This only works at `http://localhost:5173` out of the box, not automatically
through an ngrok tunnel - see [`decisions/0010`](decisions/0010-google-oauth-authorization-code-flow.md)
and the next section for how to make it work over ngrok too.

### Google sign-in over ngrok

ngrok's free tier normally hands out a new random URL every run, which can't
be pre-registered with Google. Fix this once with a free static ngrok domain:

1. https://dashboard.ngrok.com/domains → **+ Create Domain** (free accounts
   get one static `<name>.ngrok-free.app` domain).
2. Add `NGROK_DOMAIN=<name>.ngrok-free.app` to `.env`.
3. In Google Cloud Console (Credentials → your OAuth client → Authorized
   redirect URIs), add `https://<name>.ngrok-free.app/api/auth/google/callback`
   (you can keep the `localhost:5173` one alongside it - Google allows
   multiple).
4. In `.env`, set:
   ```
   GOOGLE_REDIRECT_URI=https://<name>.ngrok-free.app/api/auth/google/callback
   FRONTEND_BASE_URL=https://<name>.ngrok-free.app
   ```
5. `docker compose up -d api` (recreate, not just `restart` - Compose only
   re-reads `.env` when a container is created, not on every restart) then
   `.\start.ps1`. The printed public URL is now your static domain every
   time. See [`decisions/0012`](decisions/0012-optional-ngrok-static-domain-for-google-oauth.md).

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
| `prod-build`     | `docker compose -f docker-compose.prod.yml build`   |
| `prod-up`        | `docker compose -f docker-compose.prod.yml up -d --build` |
| `prod-down`      | `docker compose -f docker-compose.prod.yml down`    |

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
- [`diagrams/`](diagrams/README.md) - system architecture, auth request sequence, and data model diagrams (Mermaid `.md` + one interactive `system-architecture.html`), plus [dev vs production serving](diagrams/dev-vs-prod-serving.md): on-demand JSX compilation and the HMR WebSocket vs the nginx static bundle and `/api` proxy.
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
| GET | `/api/auth/google/login` | none |
| GET | `/api/auth/google/callback` | none |
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
