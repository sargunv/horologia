# Abstractions / Layering Review

## Critical

### `taskengine` functions accept `*dbgen.Queries` directly

Business logic is coupled to the sqlc-generated code. Every function in `taskengine` receives
`*dbgen.Queries` — a concrete generated type — as a direct parameter:

- `HandleCompletionTransition(ctx, q *dbgen.Queries, ...)`
- `ApplyCompletionTriggers(ctx, q *dbgen.Queries, ...)`
- `ApplyPoolRotation(ctx, q *dbgen.Queries, ...)`
- `SpawnTaskFromTemplate(ctx, q *dbgen.Queries, ...)`

This means:

- Changing the DB access library requires modifying all business logic functions.
- Stateful functions like `HandleCompletionTransition` cannot be unit-tested without a real
  database.
- Only pure functions like `AdvanceRotation` (that don't take `*dbgen.Queries`) have unit tests;
  everything else is covered only by integration tests.

**Fix:** Define a narrow data-access interface in `taskengine` declaring only the methods it needs.
`*dbgen.Queries` satisfies it implicitly.

**Files:** `taskengine/completion.go`, `taskengine/rotation.go`, `taskengine/spawn.go`,
`taskengine/recurrence.go`

---

## Important

### Transaction ownership is inconsistent between layers

For simple CRUD handlers, the handler begins a transaction and passes `dbgen.New(tx)` into
taskengine. But `CreateSpaceWithDefaults` receives a `*pgxpool.Pool` and manages its own
transaction. `ProcessOverdueTasks` and `processOneOverdueTask` also receive and manage
`*pgxpool.Pool` transactions.

This means:

- It's not obvious from call sites whether a function guarantees atomicity internally.
- A future handler that needs to combine `CreateSpaceWithDefaults` with another write in the same
  transaction cannot — the function owns its transaction internally.

**Fix:** Pick one pattern: either taskengine functions always own their transactions, or they always
receive a connection/queries object from the caller.

**Files:**

- `taskengine/space_creation.go` lines 33-88
- `taskengine/overdue.go` lines 16-19, 39-61
- `api/task_handlers.go` lines 166-170, 258-262, 314-318

---

### `requireSpaceRole` runs outside the caller's transaction

Every handler begins by calling `requireSpaceRole`, which creates `dbgen.New(h.Pool)` — outside any
transaction. The authorization check and the subsequent write run as two separate, non-atomic
database round-trips. Between the auth check and the write, a member's role could theoretically
change or the space could be deleted.

**Files:** `api/space_access.go` lines 15-38

---

### `handleOIDCCallback` is inconsistent with the rest of the API layer

It's a free function (not a method on Handler), takes `*Handler` as a parameter, and produces
plain-text errors via `http.Error()` instead of the JSON error format the rest of the API uses. The
logging uses ad-hoc attribute keys rather than the structured convention.

**Files:** `api/oidc.go` lines 88-180

---

### Read-only handlers open unnecessary transactions

`SpaceTasksRead` and `SpaceTasksList` open transactions for purely read-only operations. Compare
with `SpacesRead` which correctly uses `dbgen.New(h.Pool)` without a transaction.

**Files:** `api/task_handlers.go` lines 258, 295

---

## Positives

- The `DBTX` interface in `db.go` is the correct foundation for testable data access.
- `enrichTasks` batch-fetch pattern (4 queries for N tasks) avoids N+1 problems.
- `replaceLevels` generic diff-and-sync logic avoids repetition across statuses/effort/priority.
- `types.ValidationError` / `types.ForbiddenError` as typed sentinel errors checked in `NewError` is
  clean cross-layer error signaling.
- `paginate` generic helper avoids repeated cursor logic.
