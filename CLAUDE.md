# CLAUDE.md

## Project

Tend is a self-hosted task manager. See `docs/BRIEF.md` for the full product brief.

## Roadmap

Progress is tracked in Linear (project: Tend).

## Development

This project uses `mise` for tooling and task orchestration. Run `mise tasks` to see all available
tasks. Key commands:

- `mise run dev` — start local dev environment (Tilt)
- `mise run test` — run all tests across all packages
- `mise run check` — run all linting/checks (hk)
- `mise run fix` — auto-fix linting issues

Packages (`api`, `cli`, `server`, `web`) have their own tasks scoped by name, e.g.
`mise run server:generate`, `mise run server:build`, `mise run server:test`.

## Packages

- ./api - TypeSpec definition for API.
- ./server - Golang backend service and API implementation.
- ./web - React SPA served by the backend, built with Skeleton (React) design system.
- ./cli - TODO: Golang CLI client

## Conventions

- Never use `context.Background()` when a context is available from a caller (e.g. `cmd.Context()`,
  function parameter). Thread contexts through from the top.

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

## Browser Automation

Use `playwright-cli` for web automation. Run `playwright-cli --help` for available commands.
