# API Design Review

## Major

### No filtering or sorting on `GET /spaces/{slug}/tasks`

The task list endpoint accepts only `cursor` and `limit`. There is no support for filtering by
status, assignee, tag, due date window, recurrence type, or parent task ID. The brief explicitly
calls for filtering to a subtree via parent task, sorting by staleness/due date/status/assignee, a
"today view" unified across spaces with urgency ordering, and full-text search. The implementation
in `ListTasksBySpace` currently only sorts by `ID` (cursor-based), with no filter predicates.

**Fix:** Add query parameters: `parentId`, `status`, `assigneeId`, `tags` (multi-value), `sortBy`
(due_date | staleness | status | position), `sortDir`, and a `dueWindow` range.

**Files:** `api/src/tasks.tsp:96`, `server/internal/api/task_handlers.go:247`

---

### No cross-space "today view" endpoint

The product brief's "Today view" is v0.1 scope, but there is no `GET /users/me/tasks` or
`GET /tasks` endpoint in the spec. Without this, the web SPA and CLI cannot implement the unified
dashboard without making N separate space-scoped requests and merging client-side — which is
incorrect since staleness ordering must be server-computed.

**Fix:** Add `GET /users/me/tasks` (or `GET /tasks`) with filtering by window, assignee, and
ordering by staleness/due date.

**Files:** Missing from spec entirely.

---
