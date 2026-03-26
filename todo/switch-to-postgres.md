# Switch from SQLite to PostgreSQL

## Why

SQLite lacks native enum types, timestamp-with-timezone, and JSONB. This forces workarounds
throughout the codebase:

- **Enums** are defined three times: DB CHECK constraints, Go domain types, TypeSpec. PostgreSQL
  `CREATE TYPE ... AS ENUM` is the single source of truth; sqlc generates Go types automatically.
  Domain types in `internal/types/enums.go`, sqlc overrides in `sqlc.yaml`, and the sync tests in
  `convert_test.go` all become unnecessary.

- **Timestamps** are stored as epoch seconds (INTEGER) with a custom `EpochSeconds` wrapper
  implementing Scan/Value. PostgreSQL `TIMESTAMPTZ` maps directly to `time.Time` via sqlc — no
  wrapper needed.

- **Custom fields** (planned feature) need queryable semi-structured data. SQLite offers only TEXT
  blobs or EAV tables. PostgreSQL `JSONB` with GIN indexes provides storage, indexing, and querying
  in one column type.

- **Migrations** are painful in SQLite because `ALTER TABLE` is severely limited (can't drop/rename
  columns in older versions, can't modify constraints). Several existing migrations recreate entire
  tables. PostgreSQL has full DDL support.

- **Concurrent writes** — SQLite's single-writer lock means the cron job and HTTP handlers contend.
  PostgreSQL MVCC handles this properly.

## What changes

### Schema

- Replace `CHECK (column IN (...))` with `CREATE TYPE ... AS ENUM (...)` for: `RecurrenceType`,
  `StatusCategory`, `SpaceRole`, `AuthTokenKind`, `StoredRelationKind`
- Replace `INTEGER` epoch-seconds columns with `TIMESTAMPTZ`
- Remove `due_tz` column — `TIMESTAMPTZ` preserves timezone context
- Add `JSONB` column for custom fields (when that feature is built)

### Go code

- Remove `internal/types/enums.go` — sqlc generates enum types from `CREATE TYPE`
- Remove `internal/types/epochseconds.go` — `time.Time` used directly
- Remove sqlc column overrides for enums and timestamps in `sqlc.yaml`
- Remove enum sync tests in `convert_test.go` — no longer two sources of truth
- Update `internal/types/duedate.go` — simplify since timezone is in the timestamp
- Switch DB driver from `modernc.org/sqlite` to `pgx/v5`
- Update all migration files

### Infrastructure

- Tend becomes a two-process deployment: app server + PostgreSQL
- Update Docker Compose / Tilt for local dev
- Document PostgreSQL version requirement (14+ for `JSONB` path operators)

## What stays the same

- Pivot tables for assignees, tags, rotation pool — PostgreSQL arrays don't support FK constraints,
  and these tables carry metadata (created_at, position)
- TypeSpec as the API spec source of truth
- sqlc for query codegen (switch engine from `sqlite` to `postgresql`)
- goose for migrations

## Sequencing

This should be done before v0.1. It touches the entire persistence layer so it's best done before
the schema stabilizes and users have data to migrate.
