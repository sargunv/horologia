# Contributing to Horologia

## Repository Layout

- `api/` — TypeSpec API definition and generated OpenAPI schema
- `server/` — Go backend: HTTP API, MCP server, task engine, database migrations
- `web/` — React SPA served by the backend
- `cli/` — Go CLI client (`horo`)
- `mobile/` — Kotlin Multiplatform app (Android, iOS, desktop)

## Development Setup

Horologia uses [mise](https://mise.jdx.dev/) for tooling. Install it, then run `mise install`.

## Key Commands

- `mise run dev` — start the local dev environment (Tilt: PostgreSQL, server, web app)
- `mise run generate` — run all code generation; run after changing TypeSpec or SQL
- `mise run ci` — full suite: generate → fix → build → test

Package-scoped tasks use a `//` prefix, e.g. `mise run //server:test`.

## Before opening a pull request

Run `mise run ci`. If you changed TypeSpec or SQL, run `mise run generate` first and commit the
generated files.
