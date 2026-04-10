# CLAUDE.md

## Project

Tend is a self-hosted task manager. See `docs/BRIEF.md` for the full product brief.

## Roadmap

Progress is tracked in Linear (project: Tend).

## Development

This project uses `mise` for tooling and task orchestration. Run `mise tasks` to see all available
tasks. Key commands:

- `mise run dev` — start local dev environment (Tilt)
- `mise run generate` — run all code generation (run after changing TypeSpec or route files)
- `mise run check` — run all linting/checks (hk)
- `mise run test` — run all tests across all packages
- `mise run fix` — auto-fix linting issues
- `mise run ci` — run the full suite: generate → fix → test

Package-scoped tasks use a `//` prefix, e.g. `mise run //server:generate`,
`mise run //server:build`, `mise run //server:test`.

To run any tool managed by mise, use `mise x -- [COMMAND]`

### Database

PostgreSQL is managed by mise and started automatically by Tilt. Data is stored per-branch in
`.postgres/<branch>/data/` (gitignored), so switching branches won't corrupt your schema.

- **Reset the dev database**: `mise run db:reset`, then re-trigger the `postgres` resource in the
  Tilt UI (or restart Tilt).
- **Clean up old branch databases**: `mise run db:clean`
- **External postgres**: Set `TEND_DB=postgres://user:pass@host/tend?sslmode=disable` in
  `.env.local` to skip mise-managed postgres entirely (e.g. a shared team DB).

## Packages

- ./api - TypeSpec definition for API.
- ./server - Golang backend service and API implementation.
- ./web - React SPA served by the backend, built with Skeleton (React) design system.
- ./cli - TODO: Golang CLI client

## Conventions

- Never use `context.Background()` when a context is available from a caller (e.g. `cmd.Context()`,
  function parameter). Thread contexts through from the top.
- Never call `time.Now()` inside a function when a `now time.Time` is available from a caller.
  Capture `time.Now()` once at the system boundary (HTTP handler, cron tick) and thread it through.

## Codegen Pipeline

Changes flow through two codegen steps — run `mise run generate` after any of these:

1. **TypeSpec** (`api/src/*.tsp`) → `api/tsp-output/schema/openapi.yaml`
2. **ogen** consumes the OpenAPI YAML → `server/internal/api/gen/`
3. **sqlc** (`server/internal/database/queries/*.sql`) → `server/internal/database/gen/`

Both must be re-run when you add new TypeSpec types or new/changed SQL queries. The order in
`mise run generate` handles this correctly.

## DB Migration Patterns

- Migrations live in `server/internal/database/migrations/`, named `NNNNN_description.sql`
- Use goose markers: `-- +goose Up` / `-- +goose Down`
- For nullable paired columns (like the overdue action `(after_days, action)` pair), prefer a
  `CHECK` constraint enforcing both-null-or-both-set rather than a composite type
- Partial indexes are effective for cron query patterns (filter on
  `WHERE overdue_action IS NOT NULL`)

## Task Engine Patterns

- Cron jobs: follow `RunAccumulatingCron` pattern — fire immediately on start, then tick. One
  function per concern.
- `ProcessOverdueActionTasks` / `ProcessOverdueTasks` both take `dbgen.DBTX` (not `*pgxpool.Pool`
  directly) — `*pgxpool.Pool` implements `dbgen.DBTX`, so the cron can pass pool directly.
- Activity log every cron action: use `activitylog.Log(ctx, db, entry, now)` with `From`/`To` detail
  fields.
- For "skip silently" cases in cron (e.g. referenced status was deleted), return `nil` — don't fail
  the whole batch.

## Handler Patterns

- **Patch semantics for optional-nullable fields**: if `req.Field.IsSet()` is false, preserve the
  existing value; if set to null (`IsNull()`), clear it; if set to a value, use it.
- New fields on `CreateTaskParams` / `UpdateTaskParams` are added by updating the SQL query column
  list and re-running sqlc codegen — the struct fields appear automatically from `RETURNING *`.

## UI Component Patterns

- Editor components (RecurrenceRuleEditor, OverdueActionEditor) follow a save/cancel pattern:
  - `editing` state + prop sync effect (sync only when `!editing`)
  - `cancellingRef` trick: `onMouseDown` sets ref so `onBlur` doesn't also fire save
  - `isDirty` comparison via serialized payload (JSON.stringify for nested objects)
  - Save/Cancel action bar only shown when `isDirty`
- Discriminated union for draft state prevents inconsistent field combinations (e.g. `set_status`
  action always paired with `status` string in the draft type)
- Conditional `PropertyRow` render guards keep the UI clean: wrap rows in
  `{condition && <PropertyRow...>}`

## Web App Conventions

- Use [Skeleton (React)](https://www.skeleton.dev/llms-react.txt) as the design system. Prefer
  Skeleton components over custom implementations wherever possible.
- Lean on Skeleton's built-in theming for all styling. Do not hand-roll colors, typography, or
  spacing tokens.
- Our concerns are layout, functionality, and correct use of Skeleton components — not bespoke
  styling.
- Use `/frontend-design` when building UI to ensure high quality.
- Use `createLink()` from TanStack Router to wrap Skeleton components for client-side navigation. Do
  not use Skeleton's `element` render prop for router integration — it's a power-user escape hatch
  that doesn't forward children and is not needed for normal usage.
- Read Skeleton's component docs before hand-rolling UI. Check whether a Skeleton component already
  exists for the pattern (e.g. Navigation, AppBar) rather than building it from raw HTML + Tailwind.
- Extract `useMutation` hooks into `lib/mutations.ts` only when reused across multiple components.
  One-off mutations with page-specific side effects belong inline in the component.

## Manual Testing

Use `playwright-cli` for web automation. Run `playwright-cli --help` for available commands.

Use `vhs` for recording CLI demos. Run `vhs --help` for details.

### Dev environment

`mise run dev` starts all services (postgres, server, web). The `seed` Tilt resource creates the
default admin user (`admin@localhost` / `password`) on first run.

### Capturing UI evidence

After implementing UI changes, capture a walkthrough video before committing to verify the feature
works end-to-end and provide visual evidence for the PR.
