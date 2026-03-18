# Tend — Product Brief

## What is Tend?

Tend is a self-hosted task manager for personal and shared use. It combines household chore management, one-off personal tasks, and project tracking into a single unified system. The core insight is that no existing tool does all three well — especially the household chore side, where tasks go "stale" over time and need rotation between household members.

### Why build it?

- Existing self-hosted tools (Vikunja, Plane, etc.) treat recurring chores as an afterthought — they lack staleness tracking, completion-based recurrence, and rotating assignees.
- Tody does chores well but isn't self-hosted, isn't open source, and doesn't handle project/issue tracking.
- The goal is one place to see everything: household chores, errands, project work — with collaboration for shared spaces.

### Inspiration

- **Todoist** — general task management UX
- **Tody** — staleness visualization, chore-specific UX, completion-based recurrence

---

## How It Works

### Spaces

A **space** is the top-level organizational unit. Spaces can be shared or private.

Examples:
- "Household" — shared with your partner, contains chores and house tasks
- "Personal" — private, your own errands and todos
- "Side Project X" — shared or private, project-style work

Each space has:
- **Members** with roles: admin, writer, reader
- **Custom statuses** — the space defines its own status workflow (see below)
- **Tags** — scoped to the space
- Everything in a space is visible to all members of that space

### Projects

A **project** is an optional grouping of tasks within a space. Tasks always belong to a space. They optionally belong to a project within that space.

Examples:
- Space "Household" → Project "Kitchen" (recurring kitchen chores)
- Space "Household" → standalone task "Call landlord" (no project)
- Space "Side Project X" → Project "Backend" → tasks

A task's `space_id` is required. Its `project_id` is nullable but must reference a project within the same space (enforced by composite constraint).

### Tasks

There is one universal task model. Any task can use any combination of features — there are no "task types." A chore is just a task with recurrence. A project task is just a task with comments and links.

**Core fields:**
- Title
- Description (Markdown — rendered with WYSIWYG and raw editors in GUIs)
- Status (from the space's defined statuses)
- Assignee (one user, or unassigned)
- Due date (hard date, nullable)
- Tags (from the space's tag set)
- Manual ordering via fractional indexing

**Recurrence:**
- **Fixed schedule** — "every Saturday", "1st of every month" (calendar/cron-based)
- **Completion-based** — "7 days after last completed" (interval from actual completion, not from due date)
- When a recurring task is completed, it immediately resets to the space's initial status with a new due date
- If a completion-based task is overdue and then completed, the next due date is calculated from the completion date, not the original due date

**Staleness:**
- Calculated as `(now - last_completed_at) / interval`
- At 100% the task is due. Above 100% it's overdue.
- Displayed as an urgency gradient (green → yellow → red), inspired by Tody

**Completion history:**
- Each task has a list of completion records: `{completed_by, completed_at}`
- The current task state is the latest record
- This provides a history of when chores were done and by whom

**Rotation:**
- A task can have a `rotation_pool` — a list of users
- On each completion, the assignee advances to the next person in the pool
- If someone outside the pool completes the task, they go to the end of the rotation queue

**Relations (space-defined):**
- blocks / blocked-by
- parent / child
- related-to
- duplicates

Spaces can define which relation types are available. The above are good defaults.

**Comments:**
- Flat comment list on any task

**Custom fields (future):**
- Spaces will eventually define a schema for custom fields (e.g., URLs, links to external systems, structured data)
- Not in v0.1 — tasks have description + comments for freeform context

### Statuses

Each space defines its own status workflow with three categories:

1. **Initial** (exactly one) — new tasks start here; recurring tasks reset to this status on completion
2. **Intermediate** (zero or more) — workflow states with no special system semantics
3. **Completion** (one or more) — transitioning to a completion status triggers: recurrence reset, staleness reset, completion history record, rotation advance

Examples:
- Household space: `todo` (initial) → `done` (completion)
- Project space: `backlog` (initial) → `todo` → `in progress` → `in review` (intermediate) → `done` (completion)

A non-recurring task moved to a completion status with "done" semantics means it's finished permanently. A recurring task in a completion status immediately bounces back to initial.

### Views

**Today view (home/dashboard):**
- Unified view across all spaces the user belongs to
- Configurable window (default: next 7 days)
- Sections, ordered by urgency:
  1. Stale + overdue (ordered by staleness percentage, highest first)
  2. Overdue (ordered by days overdue)
  3. Due today
  4. Upcoming within window, by due date (including staleness indicators for recurring tasks approaching their interval)
- Links to recently updated or favorited spaces

**Space/project views:**
- Task list within a space or project
- Sortable by staleness, due date, status, assignee, manual order

---

## Technical Decisions

### Architecture

```
┌──────────────────────────────────┐
│          Go binary (tend)        │
│                                  │
│  ┌────────────────────────────┐  │
│  │      Service layer         │  │
│  │   (all business logic)     │  │
│  │         SQLite             │  │
│  └────┬──────────┬────────────┘  │
│       │          │               │
│  ┌────┴───┐ ┌────┴─────────┐    │
│  │  CLI   │ │ HTTP server  │    │
│  │(direct)│ │ REST API +   │    │
│  │        │ │ static SPA   │    │
│  └────────┘ └──────────────┘    │
└──────────────────────────────────┘
```

- **Single Go binary** — CLI and server are the same binary.
  - `tend serve` — runs the HTTP server (REST API + serves the web SPA as static files)
  - `tend add`, `tend list`, etc. — CLI commands that either call the service layer directly (local mode) or call the remote API (remote mode)
  - **Local mode** — CLI opens the SQLite file directly, no server needed. Good for personal use, scripting, MCP. Database location set by `TEND_DB` (default: `~/.local/share/tend/tend.db`).
  - **Remote mode** — CLI calls a Tend server over HTTP. Activated by setting `TEND_SERVER=https://tend.local`.
- **REST API with OpenAPI spec** — typed clients generated from the spec for TS, Kotlin, Swift.
- **Web frontend** — SPA built with TanStack Router + Skeleton.dev. Compiled to static files and embedded in the Go binary. No Node.js in production.
- **Database** — SQLite. Single file, easy backup, self-hosted friendly. Designed so Postgres could be added later if needed.
- **Auth** — OIDC support (for use behind Authelia, Authentik, etc.) + username/password.
- **Deployment** — single Docker container. Volume-mount for the SQLite file.

### Future platforms (post v0.1)

- **Native apps** — KMP shared core (networking + data layer) with platform UI:
  - SwiftUI for iOS, optionally macOS (shared codebase)
  - Jetpack Compose for Android, optionally Linux and Windows (Compose for Desktop, shared codebase)
  - Tauri (SPA in webview) for any remaining platforms
- **MCP server** — for AI assistant integration
- **Offline support** — local SQLite + sync protocol. v0.1 is online-only but the API is designed with idempotent operations and timestamps to support offline later.
- **Notifications** — email, webhooks, push via relay service. Not in v0.1.

### v0.1 Scope

**In scope:**
- Single Go binary: API server + CLI (Charm stack) + embedded web SPA
- CLI works locally (direct SQLite) or against a remote server
- Web SPA (TanStack Router + Skeleton.dev)
- Spaces with members and roles (admin/writer/reader)
- Projects (optional grouping within spaces)
- Tasks with: title, description, status, assignee, due date, tags
- Space-defined statuses (initial/intermediate/completion)
- Recurrence (fixed schedule + completion-based)
- Staleness tracking and visualization
- Rotating assignees
- Completion history
- Task relations
- Markdown descriptions
- Today view (unified dashboard)
- OIDC + username/password auth
- Single container Docker deployment

**Out of scope for v0.1:**
- Custom fields / space-defined schemas
- Native mobile and desktop apps
- MCP server
- Offline support and sync
- Notifications (email, push, webhooks)
- Fractional indexing for manual ordering (sort by due date/staleness for now)
- Structured integrations (GitHub, etc.)

### Distribution

Open source, self-hosted.

### Name

**Tend** — as in tending to things. Tasks are things you tend to; chores go stale like an untended garden. The staleness gradient is literally "how untended is this?"

- No major naming conflicts found
- tend.com is a farm management tool (different market)
- `tend` on npm is an abandoned file watcher from 10+ years ago
- CLI binary: `tend`
- Potential domain: tend.dev (availability unconfirmed)
