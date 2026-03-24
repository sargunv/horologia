# Batch C: Localized Refactors

Independent refactors within `internal/api`. No dependencies on each other or on Batches A/B.

## 1. Extract generic `replaceLevels` helper

`task_level_handlers.go` has three near-identical ~80-line `Replace` handlers for statuses, effort
levels, and priority levels. Extract the shared skeleton into a generic helper:

```go
type levelOps[T any] struct {
    list   func(ctx context.Context, spaceSlug string) ([]T, error)
    name   func(T) string
    create func(ctx context.Context, spaceSlug, name string, pos int64) error
    update func(ctx context.Context, spaceSlug, name string, pos int64) error
    delete func(ctx context.Context, spaceSlug, name string) error
}

func replaceLevels[T any](...) error { ... }
```

The status handler adds its own `validateRemoval` hook (checking `CountTasksByStatusName` before
deleting). Effort/priority can use the helper directly.

## 2. Split `validateOptionalLevel` into typed helpers

`task_handlers.go:671–706` dispatches on a `label string` and panics on unknown input. Replace with
two explicit functions:

- `validateEffortLevel(ctx, q, spaceSlug, name *string) error`
- `validatePriorityLevel(ctx, q, spaceSlug, name *string) error`

This eliminates the string dispatch and the panic path.

## 3. Use structured SQLite error types

`handler.go:68–74` — `isUniqueViolation` and `isForeignKeyViolation` use `strings.Contains` on error
messages. Replace with type assertion to `*sqlite.Error` and check the numeric `ExtendedCode` field
(`SQLITE_CONSTRAINT_UNIQUE`, `SQLITE_CONSTRAINT_FOREIGNKEY`).
