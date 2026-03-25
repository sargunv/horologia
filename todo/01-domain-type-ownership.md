# Domain Type Ownership

The biggest structural issue: domain concepts are defined in the generated API layer (`api/gen`) and
consumed raw by `taskengine`, rather than owned by the domain layer.

## Issues

### `taskengine` imports `api/gen`

- Files: `internal/taskengine/recurrence.go`, `completion.go`, `cron.go`, `spawn.go`,
  `space_creation.go`
- `taskengine` uses `apigen.TaskRecurrenceType`, `apigen.TaskStatusCategory`, `apigen.SpaceRole`
  directly. Changing the OpenAPI schema forces changes in business logic even when behavior hasn't
  changed.
- `space_creation.go` (moved from `api`) also references `apigen.TaskStatusCategoryInitial`,
  `apigen.TaskStatusCategoryCompletion`, and `apigen.SpaceRoleAdmin`.
- Fix: define canonical domain types in `taskengine` (e.g. `taskengine.RecurrenceType`) and confine
  `apigen` references to the `api` layer's conversion functions.

### `SpawnTaskFromTemplate` takes `string` instead of typed enum

- File: `internal/taskengine/spawn.go:24`
- Every call site converts `apigen.TaskRecurrenceType` to `string` before calling.
- Fix: accept `apigen.TaskRecurrenceType` (or a domain type) and convert to `string` internally for
  the DB write.

### `CopyOnSpawnKinds` keyed on untyped strings

- File: `internal/taskengine/engine.go:14`
- No `storedRelationKind` type or constants exist. A typo in `directedKindMap` silently prevents
  relation copying with no compile-time error.
- Fix: introduce a named type (e.g. `type StoredRelationKind string`) with constants.

### `StoredKindCopyOnSpawn()` exports domain policy from `api`

- File: `internal/api/convert.go:363`
- The copy-on-spawn policy is a business rule defined in the transport package and injected into
  `taskengine.Engine`. The data flow is inverted.
- Fix: move the policy into `taskengine` alongside the domain types.
