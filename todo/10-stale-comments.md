# Stale Comments Review

## Important

### Three entire files with stale SQLite architecture references

After the project migrated from SQLite to PostgreSQL, two files in `internal/types/` still describe
themselves as SQLite-specific and contain types that are entirely dead code:

`internal/types/boolint.go`:

- Line 9: `// BoolInt is a boolean stored as INTEGER (0 or 1) in SQLite.`
- Line 22: `// Scan implements sql.Scanner, reading an int64 from SQLite.`
- Line 35: `// Value implements driver.Valuer, writing an int64 to SQLite.`

`internal/types/epochseconds.go`:

- Line 10: `// EpochSeconds is a time.Time stored as unix epoch seconds (INTEGER) in SQLite.`
- Line 28: `// Scan implements sql.Scanner, reading an int64 from SQLite.`
- Line 41: `// Value implements driver.Valuer, writing an int64 to SQLite.`

`internal/types/enums.go`:

- Line 9: `// stringEnum is a generic type for string-backed enums stored in SQLite TEXT columns.`

**Fix:** Delete `boolint.go` and `epochseconds.go` entirely. For `enums.go`, remove "in SQLite TEXT
columns" at minimum, or delete the file if the types are unused.

---

### `directedKindMap` comment describes a field that no longer exists

`internal/api/convert.go` lines 112-118:

```go
// directedKindMap maps API-facing directed relation kinds to their stored canonical kind,
// whether source/target should be flipped, and whether the relation should be copied
// when spawning a new task from a fixed_accumulating template.
//
// copyOnSpawn policy: true for relations that describe a task's role in a workflow
// (parent/child, blocking, triggering); false for relations specific to a particular
// instance (duplicates, spawn lineage).
```

The anonymous struct only has `storedKind` and `flip` — there is no `copyOnSpawn` field. The
spawn-copy logic was moved to `taskengine/spawn.go` and `types/enums.go`.

**Fix:** Remove the sentences about "whether the relation should be copied" and the
`copyOnSpawn
policy` paragraph.

---

## Minor

### `requireSpaceRole` comment could be more precise

`internal/api/space_access.go` lines 12-14:

```go
// requireSpaceRole checks that the authenticated user has one of the given roles
// in the specified space. Global owners always pass but the space must exist
// (prevents returning empty 200 for nonexistent spaces).
```

The parenthetical says "prevents returning empty 200" — the actual concern is preventing a 200 OK
response for a nonexistent space when the caller is a global owner.

**Fix:** Tighten to:
`// Global owners always pass, but the space must still exist; this prevents a
200 OK response for a nonexistent space when the caller is a global owner.`

---

### `main.go` comment references wrong method name

`cmd/server/main.go` line 184:

```go
// ListenAndServe returns ErrServerClosed after Shutdown; drain it.
```

But the code actually calls `srv.Serve(ln)`, not `ListenAndServe`. Both return `ErrServerClosed`
after `Shutdown`, so the described behavior is correct but the method name is wrong.

**Fix:** Change to `// Serve returns ErrServerClosed after Shutdown; drain it.`
