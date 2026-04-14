# Horologia

Horologia is a self-hosted task manager for personal and household use.

## Components

**Web app** — A React single-page application with a Go backend. You host this yourself. The backend
also runs an MCP server for AI assistant integration.

**horo** — A command-line client for interacting with the server.

## Features

- **Spaces** — Shared task lists with multiple members.
- **Recurrence** — Two modes: completion-based (next due N days after you mark it done) and fixed
  calendar recurrence.
- **Assignee rotation** — Tasks rotate through space members on a configurable schedule.
- **API tokens** — Generate tokens for custom clients and automations.
- **OIDC** — Authenticate through an existing identity provider.

## License

| Package   | License                                                         |
| --------- | --------------------------------------------------------------- |
| `api/`    | MIT (see `LICENSE-MIT`)                                         |
| `cli/`    | MIT (see `LICENSE-MIT`)                                         |
| `mobile/` | MIT (see `LICENSE-MIT`)                                         |
| `server/` | GNU Affero General Public License v3.0 (see `LICENSE-AGPL-3.0`) |
