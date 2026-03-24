# Batch A: Package Restructuring

Prerequisite: Batch B (typed enums) should land first so extracted packages start with proper types.

## 1. Extract `internal/taskengine` from `internal/api`

Move business logic files out of the handler package into a new `internal/taskengine` package:

- `recurrence.go` — `validateRecurrence`, `computeNextDueAt`
- `rotation.go` — `advanceRotation`
- `spawn.go` — `spawnTaskFromTemplate`, `processAccumulatingTask`
- `cron.go` — cron job scheduling logic

The new package should accept `*dbgen.Queries` as a dependency. `internal/api` imports `taskengine`
as a consumer.

## 2. Extract `handleCompletionTransition` from `SpaceTasksUpdate`

`internal/api/task_handlers.go:313–512` is a ~200-line handler doing too many things. Extract the
completion detection and recurrence handling (lines ~362–451) into a function:

```go
handleCompletionTransition(ctx, q, existing, newStatus, recurrenceType, recurrenceRule, now) ->
  (updatedRecurrenceType, updatedRecurrenceRule, newDueAt, justCompleted, error)
```

This likely lives in the new `internal/taskengine` package.

## 3. Move business logic out of `internal/database`

`internal/database/db.go` contains `CreateSpaceWithDefaults` and `CreateUserWithPassword` which
encode policy (default statuses, bcrypt hashing). Move these to `internal/api` (or
`internal/taskengine` if appropriate). Keep `internal/database` limited to connection setup,
migrations, and the goose provider.

Note: `main.go` calls `CreateUserWithPassword` directly for the `create-admin` CLI command — this
call site needs updating too.
