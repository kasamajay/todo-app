# Authentication

Register/login/logout/me/forgot-password/reset-password, account lockout, and timing-safe login.

## What it does
- **Register** (`POST /api/auth/register`) — email + password (min 8 chars), rejects duplicate emails (409). On success, hashes the password (PBKDF2-HMAC-SHA512, see [decisions/0002](../decisions/0002-hand-rolled-pbkdf2.md)), creates the user, and returns `{token, user}` (auto-login).
- **Login** (`POST /api/auth/login`) — verifies email/password, mints and returns a token on success.
  - **Timing-safe**: PBKDF2 always runs, whether or not the email exists — against the real user's salt/hash if found, or a fixed package-level dummy salt/hash if not — so response latency can't be used to enumerate registered emails.
  - **Account lockout**: after 3 failed attempts, `LockedUntil` is set to 30 minutes in the future on the user record; further login attempts return 403 `account_locked` until it expires or an admin unlocks the account (see [admin-panel.md](admin-panel.md)). A successful login resets the failed-attempt counter.
- **Logout** (`POST /api/auth/logout`) — returns 204. Tokens are stateless (see [decisions/0003](../decisions/0003-custom-hmac-tokens-not-jwt.md)), so this is a no-op server-side; the frontend discards the token from `localStorage`.
- **Me** (`GET /api/auth/me`) — returns the current user (from the bearer token) as a `PublicUser` (no password hash/salt).
- **Forgot password** (`POST /api/auth/forgot-password`) — if the email matches an account, generates a reset token (1h expiry) and logs it server-side (see [decisions/0007](../decisions/0007-no-email-infra-reset-token-logged.md)); always responds 200 regardless of whether the email exists.
- **Reset password** (`POST /api/auth/reset-password`) — `{token, new_password}`, validates the token and expiry, rehashes the password, and clears any lockout state.
- **Sign in with Google** (`GET /api/auth/google/login` → `GET /api/auth/google/callback`) — OAuth 2.0 Authorization Code flow (see [decisions/0010](../decisions/0010-google-oauth-authorization-code-flow.md)); on the regular login screen only, not `/admin`. Matches an existing account by `GoogleID` first, then auto-links to an existing password account by verified email, else creates a new password-less account. A password `Login` attempt against a Google-only account (no password set) returns the same generic `invalid_credentials` as a nonexistent email. Disabled (routes respond `503`) unless `GOOGLE_CLIENT_ID`/`GOOGLE_CLIENT_SECRET` are configured.

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

## Key files
- `api/internal/handlers/auth_handler.go` — register/login/logout/me/forgot/reset handlers.
- `api/internal/handlers/auth_google_handler.go` — `GoogleLogin`/`GoogleCallback`.
- `api/internal/auth/pbkdf2.go`, `password.go` — password hashing.
- `api/internal/auth/token.go`, `secret.go` — token mint/verify, signing secret.
- `api/internal/auth/google.go` — stdlib HTTP calls to Google's token-exchange and userinfo endpoints.
- `api/internal/middleware/middleware.go` — `RequireAuth` extracts and verifies the bearer token.
- `api/internal/models/models.go` — `User.FailedLoginCount`, `LockedUntil`, `ResetToken`, `ResetTokenExpires`, `GoogleID`.
- `web/src/components/Login.jsx` — single component covering all four password modes (login/register/forgot/reset) plus the Google button (`allowGoogle` prop).
- `web/src/admin/AdminLogin.jsx` — renders `Login` with `allowGoogle={false}`.
- `web/src/App.jsx` — picks up `#google_token=`/`#google_error=` from the URL hash on mount after the Google redirect.
- `web/src/api.js` — `getToken`/`setToken`/`clearToken`, and `onUnauthorized` hook that bounces the app back to the login view on any 401.

## Related
[diagrams/auth-sequence.md](../diagrams/auth-sequence.md) — sequence diagram of the login request flow.
