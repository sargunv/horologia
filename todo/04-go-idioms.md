# Go Idioms Review

## Critical

### `context.Background()` used instead of `cmd.Context()` in shutdown path

This is a direct violation of the project's CLAUDE.md convention: "Never use `context.Background()`
when a context is available from a caller."

At line 178, when a shutdown signal is received the code creates a 30-second timeout context from
`context.Background()` instead of deriving it from `cmd.Context()`:

```go
shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
```

**Fix:**

```go
shutdownCtx, shutdownCancel := context.WithTimeout(cmd.Context(), 30*time.Second)
```

**Files:** `cmd/server/main.go` line 178

---

## Important

### `init()` used to initialise `inverseKindMap`

`init()` is generally discouraged in Go because it introduces implicit ordering dependencies. In
this case, `inverseKindMap` can be initialised as a package-level `var` using a helper function:

```go
var inverseKindMap = buildInverseKindMap()

func buildInverseKindMap() map[relationKey]apigen.TaskRelationKind {
    m := make(map[relationKey]apigen.TaskRelationKind, len(directedKindMap))
    for apiKind, c := range directedKindMap {
        m[relationKey{c.storedKind, c.flip}] = apiKind
    }
    return m
}
```

**Files:** `internal/api/convert.go` lines 141-154

---

### `init()` used for bcrypt sentinel hash

Same pattern — the `sentinelHash` computed in `init()` can be assigned via a `var` + closure:

```go
var sentinelHash = func() []byte {
    h, err := bcrypt.GenerateFromPassword([]byte("sentinel"), bcrypt.DefaultCost)
    if err != nil {
        panic("bcrypt: generate sentinel hash: " + err.Error())
    }
    return h
}()
```

**Files:** `internal/api/web.go` lines 19-27

---

### Read-only handlers open unnecessary transactions

`SpaceTasksList` and `SpaceTasksRead` both open a `pgx` transaction for purely read operations, and
neither ever calls `tx.Commit()`. This is functionally correct but wasteful — it can increase
connection pressure under load. All other read-only handlers (e.g., `SpacesRead`, `UsersMe`,
`SpaceTagsList`) correctly use `h.Pool` directly.

**Fix:** Remove the transaction and use `dbgen.New(h.Pool)` directly.

**Files:** `internal/api/task_handlers.go` lines 258-284, 295-303

---

### Duplicate `copyOnSpawn` logic

`spawn.go` defines a standalone `copyOnSpawn(k dbgen.StoredRelationKind) bool` function.
`types/enums.go` defines the same logic as the method `(k StoredRelationKind) CopyOnSpawn() bool`.
The method on `types.StoredRelationKind` has no callers — it is dead code.

**Fix:** Remove the `CopyOnSpawn()` method from `types/enums.go`.

**Files:**

- `internal/taskengine/spawn.go` lines 16-24
- `internal/types/enums.go` lines 143-151

---

### `init()` for cobra command wiring in `migrate.go`

```go
func init() {
    migrateCmd.AddCommand(migrateUpCmd, migrateStatusCmd)
}
```

Move to `main()` alongside the existing `rootCmd.AddCommand` call.

**Files:** `cmd/server/migrate.go` lines 97-99

---

### `errors.Is` not used for sentinel error in `dev-oidc`

```go
if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
```

The main server at `cmd/server/main.go` line 185 already uses `errors.Is` correctly for the same
sentinel. This is inconsistent.

**Fix:**

```go
if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
```

**Files:** `cmd/dev-oidc/main.go` line 106
