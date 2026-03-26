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

### "Exactly one initial status" constraint not enforced

The brief says "Initial (exactly one) — new tasks start here." The `SpaceTaskStatusesReplace`
handler only checks for at-least-one initial status, not exactly-one. You can submit two statuses
with `category: "initial"` and it will succeed.

**Fix:** Add a counter check: if `initialCount > 1`, return a 400.

**Files:** `server/internal/api/task_level_handlers.go:122-135`

---

## Minor

### `Task.status`, `effort`, `priority` are undocumented plain strings

These fields reference space-scoped values but are typed as bare `string` with no documentation that
they must be valid names from the corresponding space configuration endpoints.

**Fix:** Add `description` annotations to these fields in the TypeSpec.

**Files:** `api/src/tasks.tsp:47,68,81`

---

### Error responses use `default:` in OpenAPI rather than explicit status codes

All error paths are expressed as `default:` responses rather than explicit `400:`, `403:`, `404:`,
`409:` responses. The handler clearly produces distinct status codes which should be reflected in
the spec.

**Fix:** Add explicit response codes to each operation in the TypeSpec using union types with
`@statusCode`.

**Files:** `api/tsp-output/schema/openapi.yaml`

---

### No `GET /auth/tokens/{id}` read endpoint

After `POST /auth/tokens` returns the `AuthTokenCreateResponse`, the only way to retrieve token
metadata again is to list all tokens and find it by ID.

---

### Access-control error semantics undocumented

The same 404 is returned for "space does not exist" and "you are not a member." The spec does not
document these access-control semantics, making it hard for API consumers to handle these cases.

**Files:** `server/internal/api/space_access.go:28-38`

---

### `TaskRelation` kinds are undocumented

The spec exposes all 10 relation kinds with no documentation on their semantics — which have system
behavior (triggers, spawns) and which are informational (relates_to).

**Fix:** Add `description` annotations to each `TaskRelationKind` enum value.

**Files:** `api/src/tasks.tsp:15-26`

---

### `Space.description` default undocumented

`Space.description` is required in responses but optional on create. The default (empty string) is
not documented in the spec.

**Files:** `api/src/spaces.tsp:7-13`

---
