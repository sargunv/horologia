# Type Design Review

## Critical

### Duplicate enum definitions between `types/enums.go` and `database/gen/models.go`

Both files independently declare identical enum types: `RecurrenceType`, `StatusCategory`,
`SpaceRole`, `AuthTokenKind`, and `StoredRelationKind`. The `internal/types` package defines them
for the service layer, while sqlc regenerates its own in `internal/database/gen/models.go`. In
`convert.go` and `task_handlers.go`, the code casts between them with bare type conversions like
`dbgen.RecurrenceType(req.RecurrenceType...)`. These casts are unchecked — any future divergence
(e.g., adding a value to one but not the other) produces a silent runtime error rather than a
compile-time failure.

The `types` package enums are now entirely shadowed by the `dbgen` generated ones. The
`types.StoredRelationKind` / `dbgen.StoredRelationKind` distinction is particularly fragile:
`convert.go` line 193 casts `apigen.TaskRelationKind` directly to `dbgen.StoredRelationKind` as a
fallback for symmetric kinds, without any validation that the string value is actually a member of
the `StoredRelationKind` set.

**Fix:** Audit whether `internal/types/enums.go` is imported anywhere outside tests. If not, remove
it and consolidate on `dbgen` types.

**Files:**

- `server/internal/types/enums.go`
- `server/internal/database/gen/models.go`
- `server/internal/api/convert.go` lines 173, 193, 226, 349, 359

---

## Important

### `TaskStatus.category` and `AuthToken.kind` are anonymous string unions

In `task_statuses.tsp`, `category` is typed as `"initial" | "intermediate" | "completion"` (an
anonymous union literal) rather than a named enum. Same for `AuthToken.kind` in `auth.tsp`. Every
other categorical field in the API uses a proper named enum (`TaskRecurrenceType`,
`TaskRelationKind`, `SpaceRole`). This inconsistency means generated clients get anonymous string
union types with no shared name to reference.

**Fix:** Promote these to named `enum TaskStatusCategory { ... }` and `enum AuthTokenKind { ... }`
in the TypeSpec.

**Files:**

- `api/src/task_statuses.tsp` lines 9, 19
- `api/src/auth.tsp` line 21

---

### `TaskDue.at` is `utcDateTime` but the intent is a calendar date

The brief describes due dates as "hard date, nullable." The server stores this as `pgtype.Date`. The
API wire format carries a full ISO 8601 timestamp, not a plain date string. Clients must be aware
that the time component is an artifact of the timezone encoding, not a meaningful time-of-day.

**Fix:** Use TypeSpec's `plainDate` scalar for `TaskDue.at`, together with the separate `timezone`
field.

**Files:** `api/src/tasks.tsp` line 41

---

### `BoolInt` and `EpochSeconds` are dead SQLite-era types

These types implement `sql.Scanner`/`driver.Valuer` for SQLite's INTEGER encoding. The project now
uses PostgreSQL with native `BOOLEAN` and `TIMESTAMPTZ` columns via `pgtype`. Neither type is
referenced anywhere. If either were accidentally used with PostgreSQL at runtime, they would
silently return a scan error or panic.

**Fix:** Delete both files.

**Files:**

- `server/internal/types/boolint.go`
- `server/internal/types/epochseconds.go`

---

## Moderate

### `Task.status`, `effort`, `priority` are bare strings with no type safety

These carry user-defined names from the space's configuration but are typed as bare `string`. There
is no indication in the API types that these values must be valid names from `TaskStatus`,
`TaskEffortLevel`, or `TaskPriorityLevel`. The asymmetry with `RecurrenceType` (which gets a proper
enum) will confuse API consumers.

**Fix:** At minimum use `@format("task-status-name") statusName: string` or document the constraint.

**Files:** `api/src/tasks.tsp` lines 48-50

---

### `TaskEffortLevel` and `TaskPriorityLevel` are structurally identical but fully duplicated

Both are space-configurable name lists with identical models (`name: string`, `position: int64`),
with full duplication across four TypeSpec files, four Go handler files, and the entire DB layer.

**Fix:** Consider a generic pattern in TypeSpec (e.g., a templated `SpaceLevelList<T>`) rather than
full copy-paste.

**Files:**

- `api/src/task_effort_levels.tsp`
- `api/src/task_priority_levels.tsp`

---

### Missing `show_staleness` field on `Task`

The brief explicitly describes `show_staleness: boolean` as a per-task toggle, inherited from space
defaults and overridable per task. Neither the TypeSpec `Task` model nor the DB `Task` struct
includes this field.

**Files:** `api/src/tasks.tsp` (absent), `server/internal/database/gen/models.go` (absent)

---

### No `parentTaskId` on `TaskCreate`

The brief describes hierarchy as "freeform via parent/child relations" and parent/child is a
hardcoded relation type. But `TaskCreate` has no way to set a parent at creation time — the caller
must create the task and then POST a separate relation.

**Files:** `api/src/tasks.tsp` lines 63-75

---

### `TaskRelation.taskId` uses `Id` suffix instead of Go-idiomatic `ID`

The generated Go API structs use `TaskId` while internal Go code universally uses `ID`. This is a
codegen artifact from TypeSpec.

**Files:** `api/src/tasks.tsp` line 30
