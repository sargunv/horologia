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

## Core Task Features

1. **Tags** — CLI, web
2. **Task relations** — parent/child, blocks/blocked-by, relates-to, duplicates
3. **Recurrence** — all 5 types (one-off, completion-based, fixed accumulating, fixed
   non-accumulating, on-dependency)
4. **Staleness tracking** — derived calculation, urgency gradient in UI
5. **Completion history / activity log** — log all task actions with user+token attribution
6. **Rotation pools** — rotating assignees on completion
7. **Space-level defaults** — default recurrence, staleness toggle, assignees for new tasks

## Views & UX

8. **Web editing** — create/edit tasks, spaces, members from the web UI
9. **Today view** — unified dashboard across all spaces, sorted by staleness/urgency
10. **Search** — full-text search on title/description, filterable by space/status/assignee/tags
11. **Markdown rendering** — render descriptions as Markdown in web UI

## Infrastructure

12. **MCP endpoint** — Streamable HTTP at `/mcp`
13. **Dockerfile** — single container deployment
