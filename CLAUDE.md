# CLAUDE.md

## Project

Horologia is a self-hosted task manager.

## Roadmap

Progress is tracked in Linear (project: Horologia).

## Development

This project uses `mise` for tooling and task orchestration. Run `mise tasks` to see all available
tasks. Key commands:

- `mise run dev` — start local dev environment (Tilt)
- `mise run generate` — run all code generation (run after changing TypeSpec or route files)
- `mise run check` — run all linting/checks (hk)
- `mise run test` — run all tests across all packages
- `mise run fix` — auto-fix linting issues
- `mise run ci` — run the full suite: generate → fix → build → test

Package-scoped tasks use a `//` prefix, e.g. `mise run //server:generate`,
`mise run //server:build`, `mise run //server:test`.

Native-client bootstrap/setup uses package-scoped tasks as well:

- `mise run //clients/native:setup` — install Android SDK components into the mise-managed SDK root
- `mise run //clients/native:android:assembleDebug` — build the Android target in `:compose-app`
- `mise run //clients/native:desktop:run` — run the desktop target in `:compose-app`
- `mise run //clients/native:ios:gen` — generate the Xcode project from `swiftui-app/project.yml`
  via xcodegen (provisioned automatically by `mise install`)
- `mise run //clients/native:ios:xcframework` — assemble `HorologiaCore.xcframework` for iOS +
  simulator + macOS
- `mise run //clients/native:ios:build` — build for the iOS simulator (depends on `ios:gen`)
- `mise run //clients/native:ios:open` — open the Xcode project (depends on `ios:gen`)
- `mise run //clients/native:macos:build` — build as a native macOS app (depends on `ios:gen`)
- `mise run //clients/native:macos:run` — build + launch the macOS app

The app launches into the sign-in flow; create a local dev account via `POST /app/auth/login`
against the running dev server and sign in through the app's server picker.

To run any tool managed by mise, use `mise x -- [COMMAND]`

### Database

PostgreSQL is managed by mise and started automatically by Tilt. Data is stored per-branch in
`.postgres/<branch>/data/` (gitignored), so switching branches won't corrupt your schema.

- **Reset the dev database**: `mise run db:reset`, then re-trigger the `postgres` resource in the
  Tilt UI (or restart Tilt).
- **Clean up old branch databases**: `mise run db:clean`
- **External postgres**: Set `HOROLOGIA_DB=postgres://user:pass@host/horologia?sslmode=disable` in
  `.env.local` to skip mise-managed postgres entirely (e.g. a shared team DB).

## Packages

- ./api — TypeSpec definition for the API.
- ./server — Go backend service and API implementation.
- ./clients/web — React SPA served by the backend. Built on Tailwind v4 + daisyUI 5, with primitives
  in `clients/web/src/ui/` that wrap Radix UI (umbrella), cmdk, Ark UI (TagsInput only), and sonner.
- ./clients/cli — Go CLI client (binary: `horo`).
- ./clients/native — Kotlin Multiplatform monorepo. `:core` is the shared KMP library (ViewModel +
  generated OpenAPI client); `:compose-app` is the Compose UI for Android + desktop (Linux / Windows
  / Mac dev); `swiftui-app/` is the SwiftUI app for iOS, iPadOS, and macOS (Xcode project generated
  from `project.yml` via xcodegen).

## Conventions

- Never use `context.Background()` when a context is available from a caller (e.g. `cmd.Context()`,
  function parameter). Thread contexts through from the top.
- Always thread context down from the system boundary: HTTP handlers in the server and
  `cmd.Context()` in the CLI. Do not create a fresh background context in lower layers.
- Never call `time.Now()` inside a function when a `now time.Time` is available from a caller.
  Capture `time.Now()` once at the system boundary (HTTP handler, cron tick) and thread it through.

## Codegen Pipeline

Changes flow through two codegen steps — run `mise run generate` after any of these:

1. **TypeSpec** (`api/src/*.tsp`) → `api/gen/openapi.yaml`
2. **ogen** consumes the OpenAPI YAML → `api/gen/go/ogen/`
3. **sqlc** (`server/internal/database/queries/*.sql`) → `server/internal/database/gen/`

Both must be re-run when you add new TypeSpec types or new/changed SQL queries. The order in
`mise run generate` handles this correctly.

## DB Migration Patterns

- Migrations live in `server/internal/database/migrations/`, named `NNNNN_description.sql`
- Use goose markers: `-- +goose Up` / `-- +goose Down`
- For nullable paired columns (like the overdue action `(after_days, action)` pair), prefer a
  `CHECK` constraint enforcing both-null-or-both-set rather than a composite type
- Partial indexes are effective for cron query patterns (filter on
  `WHERE overdue_action IS NOT NULL`)

## Task Engine Patterns

- Cron jobs: follow `RunAccumulatingCron` pattern — fire immediately on start, then tick. One
  function per concern.
- `ProcessOverdueActionTasks` / `ProcessOverdueTasks` both take `dbgen.DBTX` (not `*pgxpool.Pool`
  directly) — `*pgxpool.Pool` implements `dbgen.DBTX`, so the cron can pass pool directly.
- Activity log every cron action: use `activitylog.Log(ctx, db, entry, now)` with `From`/`To` detail
  fields.
- For "skip silently" cases in cron (e.g. referenced status was deleted), return `nil` — don't fail
  the whole batch.

## Handler Patterns

- **Patch semantics for optional-nullable fields**: if `req.Field.IsSet()` is false, preserve the
  existing value; if set to null (`IsNull()`), clear it; if set to a value, use it.
- New fields on `CreateTaskParams` / `UpdateTaskParams` are added by updating the SQL query column
  list and re-running sqlc codegen — the struct fields appear automatically from `RETURNING *`.

## UI Component Patterns

- Editor components (RecurrenceRuleEditor, OverdueActionEditor) follow a save/cancel pattern:
  - `editing` state + prop sync effect (sync only when `!editing`)
  - `cancellingRef` trick: `onMouseDown` sets ref so `onBlur` doesn't also fire save
  - `isDirty` comparison via serialized payload (JSON.stringify for nested objects)
  - Save/Cancel action bar only shown when `isDirty`
- Discriminated union for draft state prevents inconsistent field combinations (e.g. `set_status`
  action always paired with `status` string in the draft type)
- Conditional `PropertyRow` render guards keep the UI clean: wrap rows in
  `{condition && <PropertyRow...>}`

## Web App Conventions

### Design system

- The app's design system lives in `clients/web/src/ui/`. Import UI primitives from there first;
  only drop down to Radix / cmdk / Ark directly when the primitive doesn't cover the case, and if
  you do it more than once, add a wrapper in `clients/web/src/ui/`.
- Primitives in use: `Dialog`, `DropdownMenu` (+ `DropdownMenuSub*`), `Tooltip`, `Tabs`, `Avatar`,
  `TagsInput`, `Toaster`. Plus `surface.ts` (shared overlay classes `SURFACE`, `SURFACE_MOTION`,
  `MENU_ITEM`) and `cx.ts` (tiny classname combiner).
- Searchable menus: compose `DropdownMenuRoot` + `FieldPill` (trigger) +
  `components/SearchableMenuContent.tsx` + `lib/useMenuSearch.ts`. See `TaskDetail.tsx` / the task
  menu fields for the canonical pattern.
- Mutation toasts: `notifyStaleData()` from `lib/toaster.ts` covers "mutation succeeded but cache
  invalidation failed." For other toasts, import `toast` from `sonner` directly.

### Styling

- Lean on daisyUI's defaults (`btn`, `input`, `textarea`, `select`, `badge`, `alert`, `menu`,
  `loading loading-spinner`, `avatar`, `card`). Don't reinvent what daisyUI already provides.
- Use daisyUI semantic tokens for color (`bg-base-100/200/300`, `text-base-content`,
  `text-base-content/70`, `bg-primary`, `bg-error`, etc.) — never hardcoded Tailwind colors.
- Use daisyUI radius tokens: `rounded-box` for cards/dialogs/overlays, `rounded-field` for
  buttons/inputs/menu-items, `rounded-selector` for chips/toggles.
- daisyUI 5 ships borders on `input`/`textarea` by default — don't add `border border-base-300`
  alongside. Same for focus rings (`outline` is handled by the `input` class).
- daisyUI 5 input-with-icon pattern: wrap `<svg>` + `<input>` inside `<label class="input">`. See
  `TaskSearchCombobox.tsx`.
- Don't add a project-wide `:focus-visible` rule — daisyUI components ship their own and a global
  override fights with them. Add focus outlines inline on the few custom-styled elements that need
  them (sidebar links, etc.).
- Use `/frontend-design` when building new UI. Prefer restraint: match the feel of existing pages
  before reaching for bolder choices.

### Router

- Use `createLink()` from TanStack Router to attach client-side navigation to any component that
  accepts anchor props. For plain anchors, `createLink("a")`. Wrap custom primitives by passing the
  component as the argument, e.g. `createLink(DropdownMenuItem)`.

### Mutations

- Extract `useMutation` hooks into `lib/mutations.ts` only when reused across multiple components.
  One-off mutations with page-specific side effects belong inline in the component.
- Wrap `queryClient.invalidateQueries()` inside `onSuccess` with a try/catch and call
  `notifyStaleData()` on failure — a thrown invalidation turns a successful mutation into a
  misleading error alert otherwise. See the hooks in `lib/mutations.ts` for the canonical shape.

## Manual Testing

Use `playwright-cli` for web automation. Run `playwright-cli --help` for available commands.

Use `vhs` for recording CLI demos. Run `vhs --help` for details.

### Dev environment

`mise run dev` starts all services (postgres, server, web). The server bootstraps the default admin
user (`admin@localhost` / `password`) on first run via `HOROLOGIA_INIT_OWNER_*`.

### Capturing UI evidence

After implementing UI changes, capture a walkthrough video before committing to verify the feature
works end-to-end and provide visual evidence for the PR.
