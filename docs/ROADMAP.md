# Tend v0.1 — Roadmap

## Done

- Spaces CRUD (API, server, CLI, web read-only)
- Tasks CRUD (title, description, status, assignees, due date)
- Space-defined statuses (initial/intermediate/completion categories)
- Space members with roles (admin/member/viewer)
- Auth: OIDC, password login, session tokens, personal API tokens
- CLI: full CRUD for spaces, tasks, auth
- Web: login, space list, task list, task detail (read-only)
- Tags — schema, API (CLI and web remaining)
- Task relations — schema, API (CLI and web remaining)
- Effort and priority levels on tasks — schema, API, CLI flags, web display
- CRUD for statuses, effort levels, and priority levels (PUT replace-all API)

## Core Task Features

1. **Recurrence** — all 5 types (one-off, completion-based, fixed accumulating, fixed
   non-accumulating, on-dependency)
2. **Completion history / activity log** — log all task actions with user+token attribution
3. **Rotation pools** — rotating assignees on completion
4. **Space-level defaults** — default recurrence, staleness toggle, assignees for new tasks
5. **Today view** — unified dashboard across all spaces, sorted by staleness/urgency
6. **Search** — full-text search on title/description, filterable by space/status/assignee/tags

## Views & UX

1. **CLI feature parity with backend**
2. **Web feature parity with backend and CLI**
3. **Web pagination** — handle `nextCursor` in list views
4. **Web editing** — create/edit tasks, spaces, members from the web UI
5. **Markdown rendering** — render descriptions as Markdown in web UI

## Infrastructure

1. **MCP endpoint** — Streamable HTTP at `/mcp`
2. **Dockerfile** — single container deployment
