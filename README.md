# Horologia

A self-hosted task management application.

## Features

- Self-hosted web app. You own your data.
- Task model optimized for personal or household use cases:
  - Completion-based recurrence scheduling
  - Calendar-based recurrencescheduling
  - Rotating assignees
  - Customizeable entries for status, effort, priority, etc.
  - Task relations (depends, contains, relates, etc)
- OIDC authentication with option to disable password login
- MCP server integration
- CLI client (`horo`) for command-line access

## Usage

See [docs/README.md](./docs/README.md) for details.

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md) for details.

## License

The CLI client (`clients/cli/`) is licensed under the [MIT](LICENSE-MIT) license.

The server components (`server/`, `clients/web/`) are licensed under the
[AGPL-3.0](LICENSE-AGPL-3.0) license.
