# Contributing to Horologia

## Repository Layout

| Package   | Language             | Description                                            |
| --------- | -------------------- | ------------------------------------------------------ |
| `api/`    | TypeSpec             | API definition and generated OpenAPI schema            |
| `server/` | Go                   | HTTP API, MCP server, task engine, database migrations |
| `web/`    | TypeScript / React   | Single-page application served by the backend          |
| `cli/`    | Go                   | `horo` command-line client                             |
| `mobile/` | Kotlin Multiplatform | Android, iOS, and desktop app                          |

## Development Setup

Horologia uses [mise](https://mise.jdx.dev/) for tooling. Install it, then run `mise install`.

## Key Commands

| Command             | What it does                                                               |
| ------------------- | -------------------------------------------------------------------------- |
| `mise run dev`      | Starts the local dev environment via Tilt: PostgreSQL, server, and web app |
| `mise run generate` | Runs all code generation — run after changing TypeSpec or SQL              |
| `mise run ci`       | Full suite: generate → fix → build → test                                  |

Package-scoped tasks use a `//` prefix, e.g. `mise run //server:test`.

## Before opening a pull request

Run `mise run ci`. If you changed TypeSpec or SQL, run `mise run generate` first and commit the
generated files.
