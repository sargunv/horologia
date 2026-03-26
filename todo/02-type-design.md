# Type Design Review

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
