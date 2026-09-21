# Tasks & Kanban board

Per-user, per-board task CRUD, surfaced as a 3-column Kanban board with native drag-and-drop.

## What it does
- Every task belongs to exactly one board (`board_id`) and one user (`user_id`); `POST /api/tasks` validates that the referenced board exists and is owned by the caller before creating the task.
- `status` is one of `todo` / `in_progress` / `done` (`models.TaskStatus`, validated server-side). New tasks default to `todo`.
- `GET /api/tasks?board_id=...` lists tasks scoped to both the caller and (optionally) a specific board; without `board_id` it returns all of the caller's tasks across every board.
- `PUT /api/tasks/{id}` handles both full field edits (from the task edit modal) and single-field status changes (from dragging a card between columns) — the request body only needs to include the fields being changed, since each field is a `*string` pointer that's only applied when non-nil.
- Deleting a task cascades to its attachments (metadata + binary blobs) before deleting the task itself.
- **Drag-and-drop** is native HTML5 DnD (`draggable`, `dataTransfer`), no library: `TaskCard.jsx` sets the task ID on `dragstart`; each column `<div>` in `Kanban.jsx` handles `onDragOver` (calls `preventDefault()` to allow dropping, plus a highlight state toggle) and `onDrop` (reads the task ID back out and calls `PUT /api/tasks/{id}` with the new status). The UI updates optimistically and rolls back if the API call fails.

## API
| Method | Path | Auth | Notes |
|---|---|---|---|
| GET | `/api/tasks` | bearer | optional `?board_id=` filter |
| POST | `/api/tasks` | bearer | validates board ownership |
| PUT | `/api/tasks/{id}` | bearer (owner) | partial update; used for both edits and DnD status moves |
| DELETE | `/api/tasks/{id}` | bearer (owner) | 204, cascades to attachments |

## Key files
- `api/internal/handlers/tasks_handler.go`
- `api/internal/storage/tasks_store.go` — `ListByUser`, `ListByBoard`, `ListByBoardAny`
- `api/internal/models/models.go` — `Task`, `TaskStatus`
- `web/src/components/Kanban.jsx` — column layout, drag/drop orchestration, optimistic updates
- `web/src/components/TaskCard.jsx` — the draggable card
- `web/src/components/TaskForm.jsx` — create/edit modal (also embeds the attachments panel — see [attachments.md](attachments.md))

## Related
[diagrams/data-model.md](../diagrams/data-model.md) shows how `Task` relates to `Board` and `Attachment`.
