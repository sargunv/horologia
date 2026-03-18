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
- **New task defaults** — space-level defaults for new tasks (e.g., `show_staleness: true`, default recurrence, default assignees). Per-task values override these.
- **Task templates (future)** — named presets with pre-filled field values (e.g., "Weekly chore": recurrence=completion-based 7d, show_staleness=true, rotation_pool=[everyone]). Not in v0.1 — space defaults are sufficient to start.
- Everything in a space is visible to all members of that space

### Tasks

Every task belongs to a space. There are no "projects" as a separate concept — a project is just a top-level task with children. Hierarchy is freeform via parent/child relations, as deep as needed.

There is one universal task model. Any task can use any combination of features — there are no "task types." A chore is just a task with recurrence. A project task is just a task with links and child tasks.

**IDs:**
- Global auto-incrementing numeric ID, prefixed by type: `T42` (task), `I13` (inventory item, future).
- The number is a global DB auto-increment, not scoped per space or type. The prefix is display-only based on the object type.
- IDs are stable, short, and unambiguous across types — usable in CLI (`tend show T42`), conversation, and MCP.

**Core fields:**
- Title
- Description (Markdown — rendered with WYSIWYG and raw editors in GUIs)
- Status (from the space's defined statuses)
- Assignees (list of users — empty = unassigned, one = sole assignee, multiple = any of them can do it)
- Due date (hard date, nullable)
- Tags (from the space's tag set)
- Manual ordering via fractional indexing (future — v0.1 sorts by due date/staleness)

**Recurrence (five types):**

1. **One-off** — no recurrence. Completed = done permanently.
2. **Completion-based** — recur N days after last completion. On completion, due date = `completed_at + interval`. Stays overdue until completed (no schedule to advance to).
3. **Fixed, accumulating** — cron schedule (e.g., "1st of every month"). Each scheduled occurrence spawns a new task. Missed occurrences pile up as separate tasks. A server-side cron job creates tasks on schedule.
4. **Fixed, non-accumulating** — cron schedule (e.g., "every Saturday"). One task, due date advances to next occurrence on completion. Auto-advance on missed: configurable `auto_advance_after` — `null` = stays overdue forever, `0` = advances to next occurrence immediately when past due, `> 0` = advances after the grace duration.
5. **On-dependency** — recurs when a specific other task is completed. Enables chains (e.g., "load dishwasher" completion triggers "unload dishwasher", and vice versa).

**Staleness:**
- A derived value, never stored. Calculated as `(now - last_completed_at) / interval`.
- At 100% the task is due. Above 100% it's overdue.
- Applies to all recurring task types. The visualization scales naturally — a week late on a weekly task (200%) is more urgent than a week late on an annual task (107%).
- Displayed as an urgency gradient (green → yellow → red), inspired by Tody.
- Computed by the server for sorting/filtering (e.g., today view ordered by staleness) and by the client for display.
- **Per-task toggle**: `show_staleness: boolean` — controls whether the urgency gradient appears on the task card. Inherited from space defaults, overridable per task.

**Activity log:**
- All actions on a task are logged: creation, status changes, assignment changes, completions, relation changes, etc.
- Each entry records: `{user_id, token_id (nullable), action, timestamp, details}`
- Completion history is a subset of this — filtered view of completion events.
- Entries attributed as "Sargun" (direct) or "Sargun (via Claude)" (via named API token).

**Assignee visibility:**
- Unassigned tasks (`assignees: []`) appear only in their space view, not on anyone's home page
- Assigned tasks appear on the home page of each assignee

**Rotation:**
- A task can have a `rotation_pool` — a list of users
- When rotation is active, completing the task sets `assignees` to `[next_person]` from the pool
- If someone outside the pool completes the task, they go to the end of the rotation queue

**Relations (hardcoded, system-defined):**
- **Parent / child** — hierarchy. A task can have one parent, many children. Used for views and grouping.
- **Blocks / blocked-by** — dependency. Blocked tasks are not actionable until the blocker is completed.
- **Relates to** — informational link, no system semantics.
- **Duplicates / duplicated-by** — marks duplicate tasks.

**Comments (future, not v0.1):**
- Flat comment list on any task

**Custom fields (future):**
- Spaces will eventually define a schema for custom fields (e.g., URLs, links to external systems, structured data)
- Not in v0.1 — tasks have description for freeform context

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

**Space views:**
- Task list within a space (optionally filtered to a subtree via parent task)
- Sortable by staleness, due date, status, assignee, manual order

---

## Technical Decisions

### Architecture

```
tend/
  api/          # OpenAPI spec (source of truth)
  server/       # tend-server binary
  cli/          # tend binary (client CLI only)
  web/          # SPA source (built → embedded in server)
```

```
┌───────────────────────────────┐
│    tend-server (Go binary)    │
│                               │
│  ┌─────────────────────────┐  │
│  │     Service layer       │  │
│  │  (all business logic)   │  │
│  │       SQLite            │  │
│  └────┬────────────────┬───┘  │
│       │                │      │
│  ┌────┴─────────┐ ┌───┴───┐  │
│  │ HTTP server  │ │ Admin │  │
│  │ REST API +   │ │  CLI  │  │
│  │ embedded SPA │ │       │  │
│  └──────────────┘ └───────┘  │
└───────────────────────────────┘
        ▲              ▲
        │              │
   ┌────┴───┐    ┌─────┴────┐
   │  tend  │    │  Tauri   │
   │  CLI   │    │  (SPA)   │
   └────────┘    └──────────┘
```

**Two Go binaries in a monorepo. Shared OpenAPI spec and generated types.**

- **`tend-server`** — the server. Owns the service layer, SQLite, REST API, MCP endpoint, and embedded web SPA.
  - `tend-server serve` — runs the HTTP server (REST API + MCP endpoint + SPA). SQLite location set by `TEND_DB` (default: `~/.local/share/tend/tend.db`)
  - `tend-server migrate` — run database migrations
  - `tend-server create-user` — bootstrap first user, etc.
- **`tend`** — the CLI client. Go + Charm. Always talks to `tend-server` over HTTP (no local/embedded mode).
  - `tend add`, `tend list`, etc. — CLI commands
  - `tend login` — obtains an API token (via browser-based OIDC flow or username/password prompt) and stores it in the OS keychain
  - Configured via `TEND_SERVER=https://tend.local`
- **REST API** — OpenAPI spec as source of truth. Typed clients generated for TS, Kotlin, Swift, Go.
- **MCP endpoint** — Streamable HTTP on `tend-server` (e.g., `https://tend.local/mcp`). Auth via OAuth 2.1, delegating to the configured OIDC provider. No binary installation needed for MCP clients.
- **Web SPA** — TanStack Router + Skeleton.dev. Compiled to static files, embedded in `tend-server`. Calls the API on the same origin. No Node.js in production.
- **Tauri app** — bundles the same SPA static files. User configures the server URL on first launch. Separate build artifact.
- **Database** — SQLite. Single file, easy backup, self-hosted friendly. Designed so Postgres could be added later if needed.
- **Auth** — OIDC support (for use behind Authelia, Authentik, etc.) + username/password. CLI auth via API tokens stored in OS keychain. MCP auth via OAuth 2.1.
- **API tokens** — users can create named personal API tokens (like GitHub PATs). Actions taken via a token are attributed to the user but tagged with the token name (e.g., "Sargun (via Claude)"). Every action records `user_id` + nullable `token_id`. No separate bot accounts — tokens inherit the user's permissions. OIDC users can create tokens; admins can create username/password users.
- **Deployment** — single Docker container running `tend-server`. Volume-mount for the SQLite file.

### Future platforms (post v0.1)

- **Native apps** — KMP shared core (networking + data layer) with platform UI:
  - SwiftUI for iOS, optionally macOS (shared codebase)
  - Jetpack Compose for Android, optionally Linux and Windows (Compose for Desktop, shared codebase)
  - Tauri (SPA in webview) for any remaining platforms
- **Inventory** — a second object type (`I13`). Track owned items: home appliances, consumables, collections (e.g., retro handhelds). Items have quantity, condition, maintenance schedules. Links to tasks (e.g., "replace air filter" triggered when inventory count is low, or recurring maintenance tasks tied to an item).
- **Offline support** — local SQLite + sync protocol. v0.1 is online-only but the API is designed with idempotent operations and timestamps to support offline later.
- **Notifications** — email, webhooks, push via relay service. Not in v0.1.

### v0.1 Scope

**In scope:**
- `tend-server`: Go, REST API + MCP endpoint + embedded web SPA + admin CLI (migrations, user bootstrap)
- `tend`: Go CLI (Charm stack), talks to `tend-server` over HTTP
- Web SPA (TanStack Router + Skeleton.dev)
- Spaces with members and roles (admin/writer/reader)
- Tasks with: title, description, status, assignees, due date, tags, parent/child hierarchy
- Space-defined statuses (initial/intermediate/completion)
- Recurrence (all five types: one-off, completion-based, fixed accumulating, fixed non-accumulating, on-dependency)
- Staleness tracking and visualization
- Rotating assignees
- Completion history
- Task relations
- Markdown descriptions
- Today view (unified dashboard)
- OIDC + username/password auth
- Single container Docker deployment

**Out of scope for v0.1:**
- Comments
- Task templates
- Custom fields / space-defined schemas
- Native mobile and desktop apps
- Offline support and sync
- Notifications (email, push, webhooks)
- Fractional indexing for manual ordering (sort by due date/staleness for now)
- Structured integrations (GitHub, etc.)

### Time zones

- Recurrence schedules are stored timezone-aware (e.g., "every Monday in America/Los_Angeles") and resolved server-side to UTC (next occurrence = specific unix timestamp).
- UI uses the client's detected time zone for display, and makes the detected zone visible so the user can correct it if wrong.

### Search

- Global search across all spaces the user has access to.
- Filterable by space, assignee, status, tags, etc.
- Full-text search on task title and description.

### License

- **AGPL-3.0** for the server (`tend-server`) — protects against third-party hosting without contributing back.
- **MIT** for client code (CLI, web SPA, mobile apps) — keeps the App Store distribution path clear for iOS/Android. The SPA source is MIT even though it's embedded in the AGPL `tend-server` binary; the AGPL applies to the combined work but the SPA source remains reusable under MIT.

### Distribution

Open source, self-hosted.

### Name

**Tend** — as in tending to things. Tasks are things you tend to; chores go stale like an untended garden. The staleness gradient is literally "how untended is this?"

- No major naming conflicts found
- tend.com is a farm management tool (different market)
- `tend` on npm is an abandoned file watcher from 10+ years ago
- CLI binary: `tend`
- Potential domain: tend.dev (availability unconfirmed)
