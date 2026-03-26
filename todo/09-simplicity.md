# Simplicity / Code Golf Review

## Important

### Duplicate `timeToTS` / `types.Timestamptz`

`convert.go` has a private `timeToTS` function identical to `types.Timestamptz`. The `api` package
already imports `types`, and `taskengine` uses `types.Timestamptz`.

```go
// convert.go lines 89-91 — REMOVE THIS
func timeToTS(t time.Time) pgtype.Timestamptz {
    return pgtype.Timestamptz{Time: t, Valid: true}
}

// types/pgtype.go lines 10-12 — KEEP THIS
func Timestamptz(t time.Time) pgtype.Timestamptz {
    return pgtype.Timestamptz{Time: t, Valid: true}
}
```

**Fix:** Delete `timeToTS` and replace all call sites in `api` with `types.Timestamptz(...)`.

**Files:** `internal/api/convert.go` lines 89-91, `internal/types/pgtype.go` lines 10-12

---

### `copyOnSpawn` duplicated between `spawn.go` and `types/enums.go`

Same logic in two places. The method on `types.StoredRelationKind` has no callers — it is dead code.

**Fix:** Remove `CopyOnSpawn()` from `types/enums.go`.

**Files:** `internal/taskengine/spawn.go` lines 16-24, `internal/types/enums.go` lines 143-151

---

### Dead SQLite-era files

Three files from the SQLite era have no callers after the PostgreSQL migration:

- `types/boolint.go` — `BoolInt` with SQLite INTEGER encoding
- `types/epochseconds.go` — `EpochSeconds` with SQLite epoch seconds
- `types/enums.go` — `stringEnum`, `Scan`/`Value` implementations for `database/sql`

**Fix:** Delete `boolint.go` and `epochseconds.go`. For `enums.go`, audit whether any types are
still imported; if not, delete the file.

---

### `applyTaskCollections` pre-fetch uses `&&` where `||` was intended

```go
if len(assigneeIDs) > 0 && len(poolIDs) > 0 {
    memberSet, err = fetchMemberSet(ctx, q, spaceSlug)
}
```

The pre-fetch only runs when both are non-empty. When only one is provided, the sub-function fetches
separately — correct but misses the optimization opportunity.

**Fix:** Change `&&` to `||` to pre-fetch eagerly whenever at least one collection needs validation.

**Files:** `internal/api/task_handlers.go` lines 431-455

---

### Unnecessary transaction in `SpacesUpdate`

The transaction wraps a read + write on the same PK. The read is only to merge optional fields. This
could be a single-query update with `COALESCE` or just use `h.Pool` directly.

**Files:** `internal/api/space_handlers.go` lines 82-109

---

### `convertEach` always returns nil error

```go
func convertEach[DB any, API any](f func(DB) *API) func([]DB) ([]API, error) {
    return func(rows []DB) ([]API, error) {
        items := make([]API, len(rows))
        for i, r := range rows {
            items[i] = *f(r)
        }
        return items, nil  // always nil
    }
}
```

Three call sites in `task_level_handlers.go` check the always-nil error:

```go
items, err := convertEach(statusFromDB)(rows)
if err != nil { return nil, err }  // dead check
```

**Fix:** Add a non-error variant `convertAll` for direct use, or remove the dead error checks.

**Files:** `internal/api/convert.go` lines 256-264, `internal/api/task_level_handlers.go`

---

## Minor

### `dbgen.New` called twice in `requireSpaceRole`

```go
if user.IsOwner {
    q := dbgen.New(h.Pool)   // first
    ...
}
q := dbgen.New(h.Pool)       // second
```

**Fix:** Hoist `q` to a single declaration before the `if`.

**Files:** `internal/api/space_access.go` lines 15-39

---

### Redundant OIDC route registration

```go
mux.Handle("/auth/oidc", oidcHandler)
mux.Handle("/auth/oidc/", oidcHandler)
```

The trailing-slash pattern is a subtree match which covers the non-trailing-slash case.

**Files:** `internal/api/oidc.go` lines 188-191

---

### Repeated TypeSpec preamble

Every `.tsp` file repeats:

```
import "@typespec/http";
using Http;
namespace Tend;
```

This could potentially be centralized depending on TypeSpec's scoping rules.

**Files:** `api/src/*.tsp`
