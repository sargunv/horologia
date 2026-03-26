# DB Schema Review

## Critical

### Activity log / completion history table is entirely absent

The product brief lists the activity log as a v0.1 in-scope feature: "All actions on a task are
logged: creation, status changes, assignment changes, completions, relations changes, etc. Each
entry records: `{user_id, token_id (nullable), action, timestamp, details}`."

The schema has no `task_events`, `activity_log`, or `completion_history` table. The
`last_completed_at` column on `tasks` is a scalar timestamp, not a history.

**Files:** `server/internal/database/migrations/00001_initial.sql` — entire file

---

## Important

### `spaces` uses a mutable slug as its primary key

The `spaces` table uses `slug TEXT NOT NULL PRIMARY KEY`. The slug cascades to `task_statuses`,
`task_effort_levels`, `task_priority_levels`, `tasks`, `space_members`, and `tags` via
`ON DELETE CASCADE`, but the FK definitions do not include `ON UPDATE CASCADE`. If the slug of a
space is ever renamed, all child rows would be orphaned or cause an FK violation.

The `UpdateSpace` query only updates `name` and `description`, not `slug`, so this is not an
immediate bug — but the absence of `ON UPDATE CASCADE` across all six child tables is a structural
trap. Using a surrogate BIGSERIAL PK for spaces (with slug as a UNIQUE column) would be safer.

**Files:** `00001_initial.sql` lines 11-17, 21, 30, 39, 63, 115, 137

---

### `task_relations.space_slug` FK lacks `ON UPDATE CASCADE`

The `tasks -> task_relations` FK does not include `ON UPDATE CASCADE`. If the space slug ever
changes (cascading from `spaces` to `tasks.space_slug`), `task_relations.space_slug` would not
cascade further, creating a silent data inconsistency.

**Files:** `00001_initial.sql` lines 164-165

---

### `show_staleness` column missing from `tasks`

The brief specifies `show_staleness: boolean` as a per-task toggle that controls whether the urgency
gradient appears on the task card. Inherited from space defaults, overridable per task. Neither the
tasks table nor the spaces table has this column.

**Files:** `00001_initial.sql` lines 48-74

---

### `auto_advance_after` column missing for `fixed_non_accumulating`

The brief specifies `auto_advance_after` for `fixed_non_accumulating` tasks: `null` = stays overdue
forever, `0` = advances immediately when past due, `> 0` = advances after the grace duration. There
is no such column in the `tasks` table, and the cron only handles `fixed_accumulating`.

**Files:** `00001_initial.sql` lines 48-74

---

## Notable

### Missing partial index on `task_relations` for trigger lookups

The `ListTriggerTargets` query filters on `source_task_id`, `space_slug`, and `kind = 'triggers'`. A
partial index `WHERE kind = 'triggers'` on `(source_task_id)` would be more efficient for this
high-frequency path (called on every task completion that might trigger dependents).

**Files:** `00001_initial.sql` line 170

---

### Slug-based keyset pagination on `spaces` is fragile

`spaces.sql` uses `WHERE slug > $1 ORDER BY slug ASC`. Keyset pagination on a text slug means sort
order is lexicographic, not insertion order. Inserting a space with a slug that sorts earlier than
the cursor will never appear in a subsequent page. Cursors based on a stable surrogate integer ID
(as used for tasks, tags, users, and tokens) avoid this class of problem.

**Files:** `queries/spaces.sql` lines 9-13

---

## Observations (not issues)

- `tags` table correctly stores both `name` and `name_folded` with case-insensitive deduplication.
- `EnsureTag` upsert preserves the original display name on conflict — correct behavior.
- `created_at` and `updated_at` are application-provided rather than `DEFAULT NOW()` — consistent
  throughout but means clock skew between application servers could cause inconsistent timestamps.
- `auth_tokens.name` defaults to `''` (empty string) rather than `NULL`, conflating "unnamed" with
  "explicitly empty."
- `DeleteTaskRelation` includes `space_slug` in WHERE as a redundant ownership check — deliberate
  and correct safety measure.
