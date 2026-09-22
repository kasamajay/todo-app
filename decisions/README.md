# Decisions

Lightweight architecture decision records (ADRs) for this project. Each file captures one decision made while building the app: the context, what was decided, alternatives that were considered, and the consequences.

| # | Decision |
|---|----------|
| [0001](0001-docker-only-dev-workflow.md) | Docker Compose for everything, no native Go/Node/make install |
| [0002](0002-hand-rolled-pbkdf2.md) | Hand-rolled PBKDF2-HMAC-SHA512, no external crypto package |
| [0003](0003-custom-hmac-tokens-not-jwt.md) | Custom HMAC-SHA256 tokens, not JWT |
| [0004](0004-json-file-storage-atomic-writes.md) | JSON file storage with atomic writes, no database |
| [0005](0005-go122-stdlib-routing.md) | Go 1.22 stdlib `http.ServeMux` routing, no router package |
| [0006](0006-cascading-deletes.md) | Deletes cascade instead of blocking on non-empty resources |
| [0007](0007-no-email-infra-reset-token-logged.md) | No email infrastructure — reset tokens are logged, not emailed |
| [0008](0008-polling-for-windows-bind-mount-hotreload.md) | File-watch polling for hot reload under Docker Desktop on Windows |
| [0009](0009-ngrok-for-public-exposure.md) | ngrok tunnel on the web container's port only |
| [0010](0010-google-oauth-authorization-code-flow.md) | Google OAuth via the Authorization Code flow, not ID-token verification |
| [0011](0011-opt-in-email-code-two-factor-auth.md) | Opt-in two-factor authentication via a logged 6-digit email code |
| [0012](0012-optional-ngrok-static-domain-for-google-oauth.md) | Optional ngrok static domain for stable Google OAuth redirects |

To add a new one: copy the format below into a new numbered file and add a row to the table above.

```markdown
# NNNN. Title

## Context
What prompted this decision.

## Decision
What was chosen.

## Alternatives considered
Other options and why they were passed over.

## Consequences
Trade-offs / follow-on effects of this choice.
```
