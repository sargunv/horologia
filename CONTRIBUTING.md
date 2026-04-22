# Contributing to Horologia

## Repository Layout

- `api/` — TypeSpec API definition and generated OpenAPI schema
- `server/` — Go backend: HTTP API, MCP server, task engine, database migrations
- `clients/web/` — React SPA served by the backend
- `clients/cli/` — Go CLI client (`horo`)
- `clients/native/` — Kotlin Multiplatform monorepo: `:core` shared library, `:compose-app` for
  Android + desktop, `swiftui-app/` for iOS / iPadOS / macOS

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
