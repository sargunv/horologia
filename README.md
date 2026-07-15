# Horologia

A self-hosted application for organizing household information and routines.

## Features

- Self-hosted web app. You own your data.
- Spaces provide ownership and collaboration boundaries.
- Tasks support personal and household workflows, including:
  - Completion-based recurrence scheduling
  - Calendar-based recurrence scheduling
  - Rotating assignees
  - Customizable statuses, effort levels, and priorities
  - Task relations such as depends, contains, and relates
- Recipes with structured ingredient and instruction sections
- Tags for organizing items within a space
- Password and OIDC authentication
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
