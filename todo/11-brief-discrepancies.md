# Discrepancies with docs/BRIEF.md

## Critical

### `show_staleness` field is completely absent

Brief:

> **Per-task toggle**: `show_staleness: boolean` — controls whether the urgency gradient appears on
> the task card. Inherited from space defaults, overridable per task.

And under Spaces:

> **New task defaults** — space-level defaults for new tasks (e.g., `show_staleness: true`, default
> recurrence, default assignees).

Neither the `Task` model in the TypeSpec nor the DB schema contains a `show_staleness` field.
Staleness visualization is a core product differentiator.

**Files:** `api/src/tasks.tsp` (absent), `00001_initial.sql` (absent)

---

### `auto_advance_after` for `fixed_non_accumulating` is missing

Brief:

> **Fixed, non-accumulating** — cron schedule. Auto-advance on missed: configurable
> `auto_advance_after` — `null` = stays overdue forever, `0` = advances immediately, `> 0` =
> advances after the grace duration.

No `auto_advance_after` field exists in the API, DB, or Go code. The cron only handles
`fixed_accumulating` — there is no automatic processing for `fixed_non_accumulating` tasks at all.

**Files:** `api/src/tasks.tsp`, `00001_initial.sql`, `internal/cron/cron.go`

---

### Space-level new task defaults are entirely missing

Brief:

> **New task defaults** — space-level defaults for new tasks (e.g., `show_staleness: true`, default
> recurrence, default assignees). Per-task values override these.

The `Space` model only has `slug`, `name`, and `description`. No space-level task defaults exist at
any layer.

**Files:** `api/src/spaces.tsp`

---

### Database uses PostgreSQL, brief specifies SQLite

Brief:

> **Database** — SQLite. Single file, easy backup, self-hosted friendly. **Deployment** — single
> Docker container. Volume-mount for the SQLite file.

The implementation uses PostgreSQL throughout. The deployment model described (volume-mount SQLite
file) no longer applies. This may be an intentional decision but the brief has not been updated.

**Files:** `cmd/server/main.go`, `go.mod`

---

## Important

### Activity log / completion history is entirely absent

Brief:

> **Activity log**: All actions on a task are logged: creation, status changes, assignment changes,
> completions, relation changes, etc. Each entry records:
> `{user_id, token_id (nullable), action, timestamp, details}`. Completion history is a subset —
> filtered view of completion events.

There is no activity log table, API endpoint, or handler. The `Task` model has `lastCompletedAt` (a
single timestamp) but no completion history. The `token_id` attribution described in the brief is
also absent.

**Files:** `00001_initial.sql` (absent), API spec (absent)

---

### MCP endpoint is missing

Brief:

> **MCP endpoint** — Streamable HTTP on `tend-server` (e.g., `https://tend.local/mcp`). Auth via
> OAuth 2.1, delegating to the configured OIDC provider.

v0.1 scope:

> `tend-server`: Go, REST API + **MCP endpoint** + embedded web SPA + admin CLI

There is no MCP handler, route, or package anywhere in the server.

**Files:** Missing entirely

---

### "Exactly one initial status" constraint not enforced

Brief:

> **Initial** (exactly one) — new tasks start here

The `SpaceTaskStatusesReplace` handler checks for at-least-one initial status but not exactly-one.
You can submit two statuses with `category: "initial"` and it succeeds.

**Files:** `internal/api/task_level_handlers.go` lines 122-135

---

## Minor

### Role naming inconsistency within the brief itself

Brief has two conflicting descriptions:

> Members with roles: admin, member, viewer _(Spaces section)_ Spaces with members and roles
> (admin/**writer**/**reader**) _(v0.1 Scope section)_

The implementation uses `admin`, `member`, `viewer` — matching the Spaces section.

---

### `triggers`/`triggered_by` are user-settable but not described in the brief's relation list

Brief lists only four hardcoded relation types:

> Parent / child, Blocks / blocked-by, Relates to, Duplicates

The API includes `triggers` and `triggered_by` as user-creatable relation kinds. The brief describes
`on_dependency` recurrence but doesn't list the enabling relation kind in the hardcoded types.

---

### `effort` and `priority` fields are undocumented additions

The brief's "Core fields" list does not mention effort or priority levels. These are additions
beyond the spec — fully implemented with space-level configuration, but not described in the brief.
