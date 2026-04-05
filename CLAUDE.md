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

## Browser Automation

Use `playwright-cli` for web automation. Run `playwright-cli --help` for available commands.

### Dev environment

`mise run dev` starts all services (postgres, server, web). The `seed` Tilt resource creates the
default admin user (`admin@localhost` / `password`) on first run.

### Capturing UI evidence

After implementing UI changes, capture a walkthrough video before committing to verify the feature
works end-to-end and provide visual evidence for the PR.
