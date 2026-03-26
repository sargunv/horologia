# Code Organization Review

## Important

### Triple enum definition creates maintenance burden

Domain enum values are defined in three separate places that must stay in sync manually:

1. `internal/types/enums.go` — hand-written Go constants with `database/sql/driver` scanning
2. `internal/database/gen/models.go` — sqlc-generated Go constants
3. `internal/database/migrations/00001_initial.sql` — PostgreSQL `CREATE TYPE ... AS ENUM`

The `types` package enums appear to be a leftover from before the PostgreSQL migration. With
`sql_package: "pgx/v5"` in `sqlc.yaml`, sqlc emits its own enum types that map to Postgres native
enums without needing custom scanner boilerplate.

The duplication is already slightly out of sync: `types.go` defines `CopyOnSpawn()` as a method on
`StoredRelationKind`, but `spawn.go` defines a package-level `copyOnSpawn()` function on
`dbgen.StoredRelationKind` with the exact same logic.

**Fix:** Audit whether `internal/types/enums.go` is imported anywhere outside tests. If not, remove
it and consolidate on `dbgen` types.

**Files:**

- `internal/types/enums.go`
- `internal/database/gen/models.go`
- `internal/taskengine/spawn.go` lines 16-24

---

### `context.Background()` in test helpers

CLAUDE.md convention violation. In `testhelpers_test.go`:

- Line 35: `ctx := context.Background()` — `t.Context()` (Go 1.21+) provides a context cancelled
  when the test ends.
- Line 139: `taskengine.CreateUserWithPassword(context.Background(), ...)` — `*testing.T` is in
  scope.

**Files:** `internal/api/testhelpers_test.go` lines 35, 60, 139

---

### `Handler` struct exposes infrastructure fields as public

```go
type Handler struct {
    apigen.UnimplementedHandler
    Pool *pgxpool.Pool
    Log  *slog.Logger
}
```

Both fields are exported. All handlers access `h.Pool` directly and call `dbgen.New(h.Pool)` inline.
As the service grows toward the full feature set in the brief, the handler will accumulate more
dependencies. There's no interface or service layer between the HTTP handler and the database
queries.

**Files:** `internal/api/handler.go`

---

## Positives

- Package structure follows Go conventions well: `cmd/` for binaries, `internal/` for non-exported
  packages, flat package names, no stutter.
- Dependency enforcement via depguard in `.golangci.yml` prevents accidental coupling.
- Generated code is cleanly isolated in `gen/` packages.
- `tools.go` pattern for pinning generator versions is correct Go practice.
- Database schema uses native PostgreSQL enum types, composite FKs, deferred constraints.
- `taskengine` isolates domain logic as pure functions accepting `*dbgen.Queries`.
- Test setup creates a per-test PostgreSQL database from a template.
- TypeSpec source-of-truth approach avoids schema drift.
- Single migration file is appropriate for pre-launch.
- Context threading is generally correct (violations are noted exceptions).
