# CLAUDE.md

## Project

Tend is a self-hosted task manager. See `docs/BRIEF.md` for the full product brief.

## Roadmap

`docs/ROADMAP.md` tracks v0.1 progress. When you complete a roadmap item, move it from its current
section into the "Done" section.

## Local Development

This project uses `mise` for tooling and task orchestration. Run `mise tasks` to see all available
tasks. Key commands:

- `mise run dev` — start local dev environment (Tilt)
- `mise run test` — run all tests across all packages
- `mise run check` — run all linting/checks (hk)
- `mise run fix` — auto-fix linting issues

Packages (`api`, `cli`, `server`, `web`) have their own tasks scoped by name, e.g.
`mise run server:generate`, `mise run server:build`, `mise run server:test`.

## Documentation

- [Skeleton design system](https://www.skeleton.dev/llms-react.txt)
