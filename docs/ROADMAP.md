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
- Recurrence — schema, API, completion triggers for completion-based, fixed non-accumulating, and
  on-dependency types. RRULE (RFC 5545) schedule format. Triggers relation kind.
- Recurrence: fixed accumulating — server-side cron job spawns one_off tasks for missed occurrences,
  completion spawns next task. Spawns/spawned_by relation kind. copyOnSpawn relation metadata.

## Core Task Features

1. **Completion history / activity log** — log all task actions with user+token attribution
2. **Rotation pools** — rotating assignees on completion

## Views & UX

1. **CLI feature parity with backend**
2. **MCP feature parity with API**
3. **Web feature parity with backend and CLI**
4. **Web pagination** — handle `nextCursor` in list views
5. **Web editing** — create/edit tasks, spaces, members from the web UI
6. **Markdown rendering** — render descriptions as Markdown in web UI
7. **Today view** — unified dashboard across all spaces, sorted by staleness/urgency

## Infrastructure

1. **CLI** - goreleaser
2. **Dockerfile** — single container deployment
