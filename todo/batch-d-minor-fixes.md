# Batch D: Minor Fixes

Small independent improvements. No particular ordering required.

## 1. Introduce `DueDate` domain struct

`DueAt` and `DueTz` are parallel nullable fields with no type-level invariant. Introduce:

```go
type DueDate struct {
    At types.EpochSeconds
    Tz string
}
```

Use `*DueDate` throughout internal code. Convert to/from separate DB columns only at the sqlc
boundary. This makes the "both nil or both non-nil" invariant structural.

## 2. Named struct for `inverseKindMap` key

`convert.go:122–125` uses an anonymous struct as a map key. Replace with a named type:

```go
type relationKey struct {
    kind string
    flip bool
}
```

## 3. `BoolInt` sqlc override for `User.IsOwner`

`IsOwner` is `int64` in the generated model, requiring manual `!= 0` coercion in two places
(`convert.go:343`, `security.go:68`). Add a sqlc `overrides` entry mapping `users.is_owner` to a
custom `types.BoolInt` type that implements `sql.Scanner` and has a `.Bool()` method.

## 4. Move main binary to `cmd/server/`

The main binary lives at the module root (`main.go`, `migrate.go`) alongside `go.mod` and config
files. Move to `cmd/server/` to match the `cmd/dev-oidc/` convention.

## 5. Deduplicate member-list fetch in `SpaceTasksUpdate`

When both `assigneeIds` and `rotationPool` are provided, `parseAndValidateUserIDs` is called twice,
each independently querying `ListSpaceMemberUserIDs`. Fetch the member set once at the top and pass
it into both helpers.
