# Skills

Project-specific skills for working on todo-app, each a `SKILL.md` following the same format used elsewhere in this workspace (see `../../../rhel-golden-image-builder/skills-lock.json` for the fetched-skill convention this deliberately does **not** use here).

| Skill | Covers |
|---|---|
| [docker-dev-workflow](docker-dev-workflow/SKILL.md) | Running/building/testing everything through `docker compose` (no native Go/Node/make), and the real bugs hit setting up `air` + Vite hot reload on Windows |
| [go-stdlib-only-backend](go-stdlib-only-backend/SKILL.md) | The stdlib-only discipline in `api/` — hand-rolled PBKDF2, custom HMAC tokens, Go 1.22 routing, the atomic JSON storage pattern |
| [frontend-no-framework-react](frontend-no-framework-react/SKILL.md) | `web/` conventions — inline styles only, no React Router, native HTML5 drag-and-drop |

## Note on provenance

These three were **hand-authored directly for this project** rather than pulled from an external skills marketplace — the same deliberate choice `portfolio-website` made for its own skills (see `../../decisions/README.md` in this project for todo-app's own ADRs, and `../../../portfolio-website/decisions/0009-hand-authored-skills.md` for the sibling project's reasoning). Content is drawn from real gotchas hit and decisions made while building this app — see [`../../decisions/`](../../decisions/README.md) for the full "why" behind each one.
