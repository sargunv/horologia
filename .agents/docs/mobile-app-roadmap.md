# Horologia Expo app roadmap

Status: accepted direction; implementation has not started\
Planning baseline: July 2026, Expo SDK 57\
Targets now: iPhone/iPad, Android phones/tablets/foldables, and the iPad build on Apple-silicon Macs

## Decision

Build the Horologia mobile app with Expo, React Native's New Architecture, strict TypeScript, Expo
Router, TanStack Query, and Expo UI.

Ship one adaptive Apple mobile app for iPhone and iPad and one adaptive Android app for phones,
tablets, foldables, ChromeOS, and desktop-style resizable windows. Keep the iPad build available on
Apple-silicon Macs and verify it as an iOS app running on Mac. Do not add a native AppKit macOS or
Mac Catalyst target to the Expo project.

Expo UI is the default presentation layer because its components are backed by SwiftUI on Apple
platforms and Jetpack Compose on Android. Shared universal components are appropriate where their
behavior is genuinely native on both platforms. Platform-specific Expo UI components and screen
implementations are expected where SwiftUI and Compose differ. Native code is a normal, contained
part of this architecture: iOS widgets use `expo-widgets`, Android widgets use Jetpack Glance in a
local Expo module, and missing platform capabilities can be exposed through Expo modules.

This is not a plan to make the existing web UI run inside a mobile shell. It is a plan to reuse the
portable parts of the client stack—React, TypeScript, generated API contracts, authentication
concepts, query behavior, validation, and domain transformations—while building an idiomatic
interface for each operating system.

## Product outcome

The app should eventually match the user-facing functionality of the web app, with global owner
administration intentionally excluded from the initial parity target. It should feel like a current,
high-quality app on each platform without creating a large bespoke design language.

Widgets are a core product surface, not a post-launch enhancement. A release without useful widgets
does not satisfy the reason for building the app.

### Goals

- Ship delightful, simple, adaptive iPhone/iPad and Android phone/tablet/foldable experiences using
  current SwiftUI and Material 3 conventions.
- Make the iPad build a good keyboard, pointer, and resizable-window app on Apple-silicon Macs
  without creating a separate macOS binary.
- Make the fastest everyday workflows—seeing, opening, and acting on tasks—excellent.
- Ship useful home-screen widgets on both platforms early in development.
- Reach behavioral parity with the non-global-admin portions of the web app.
- Keep API, domain, persistence, and feature logic reusable without forcing every platform into the
  same presentation.
- Keep all persisted client data scoped to a server even while the first UI presents only one active
  server.
- Keep writes behind a small boundary that could later gain an outbox without implementing an
  offline-write system now.
- Keep native projects reproducible from repository-owned sources and runnable without a required
  hosted Expo service.

### Non-goals for v1

- Multiple-server management in the UI.
- Offline writes, conflict resolution, a local-first domain database, or background synchronization.
- Push or local notifications.
- Global owner administration: user provisioning, owner promotion, and server About/admin screens.
- Replacing the web app with Expo Web.
- Pixel-identical Android and iOS interfaces.
- A native AppKit React Native macOS app or Mac Catalyst target. The first release's Mac binary is
  the unchanged iPad app running on Apple silicon.
- Windows, Linux, HarmonyOS/OpenHarmony, wearables, TV, or other new binaries in the first release.
- A custom animation system or an expansive cross-platform design system.

## Product and design principles

### Native by default

Use `@expo/ui` universal components when they preserve native behavior and ergonomics. Use
`@expo/ui/swift-ui` and `@expo/ui/jetpack-compose` directly when a platform has a better native
idiom. A `.ios.tsx`/`.android.tsx` split is a useful tool, not a failure of code sharing.

Use a local Expo module to expose a missing SwiftUI or Compose primitive when it materially improves
the product. Do not replace a coherent native screen with a lowest-common-denominator React Native
component tree merely to improve a code-sharing percentage.

### Simple can still be delightful

Start with system typography, spacing, icons, controls, sheets, navigation transitions, dynamic
color, haptics, and accessibility behavior. These supply polish that ages with the operating system.
Horologia-specific styling should be restrained: a small semantic color set, clear information
hierarchy, generous touch targets, and consistent empty/loading/error states.

Motion should communicate state or navigation. Prefer system transitions and platform-standard
feedback. Respect Reduce Motion and do not introduce a general-purpose animation dependency until a
specific interaction needs it.

### Share behavior, adapt presentation

Domain types, API calls, query keys, validation, date/recurrence behavior, feature state, and widget
projections should be portable TypeScript. Navigation chrome, screen composition, widgets, secure
storage, background execution, and platform integrations may differ.

Behavioral parity with the web app does not mean porting DOM components or CSS. Reuse a web module
only after verifying that it is runtime-independent. Tiptap/Radix/daisyUI components remain web
implementation details.

### Online-first without painting over the future

The server remains authoritative. Reads may be cached and the last widget snapshot may remain
visible without a connection. Writes go directly to the server and report success only after the
server accepts them.

Do not build an outbox, local identifiers, merge policies, or sync protocol for v1. Do ensure that
feature code invokes writes through a command boundary, preserves server IDs and `updatedAt` values,
and does not encode assumptions that every mutation must always be an immediate HTTP call.

## Repository shape

There is no shared TypeScript client package today. The generated OpenAPI schema, concrete browser
client, TanStack Query definitions, mutation invalidation rules, Query defaults, and portable domain
helpers all live under `clients/web`. Building the app therefore includes extracting a real shared
client core and reshaping web around it; it does not include creating mobile equivalents beside the
web implementations.

Add two workspace members and make both applications depend on the shared package:

```text
clients/
  mobile/
    app/                         # Expo Router routes and route layouts
    src/
      app/                       # composition root, providers, navigation shell
      features/                  # task, recipe, space, search, account feature slices
      persistence/               # mobile implementations of storage contracts
      platform/                  # deep links, haptics, sharing, lifecycle adapters
      widgets/                   # shared widget projections and iOS widget components
    modules/
      horologia-android-widget/  # Kotlin/Glance local Expo module
    app.config.ts
    mise.toml
  web/
    src/client/                 # browser transport, cookie session, IndexedDB adapters
packages/
  client-core/
    src/
      api/                       # generated types and transport-agnostic client factory
      auth/                      # shared session lifecycle plus injected auth strategy
      commands/                  # mutations and cache effects behind the write boundary
      domain/                    # portable parsing, recurrence, ordering, and editor state
      persistence/               # durable models and storage contracts
      queries/                   # server-scoped keys, options, defaults, and cache policy
      runtime/                   # client composition and future sync/offline coordination
      servers/                   # server identity and URL normalization
      widgets/                   # versioned widget snapshot schema/projection
```

### Extraction plan

`packages/client-core` is created by moving code out of `clients/web`, not by starting clean and
leaving web behind. The first extraction should make these changes:

| Current web source                                                                        | Shared destination and treatment                                                                                                                                                                                             |
| ----------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `src/api/schema.d.ts`                                                                     | Generate directly into `packages/client-core`; remove the web-owned generated copy.                                                                                                                                          |
| `src/api/client.ts`                                                                       | Move API error handling and the `openapi-fetch` factory into core. Keep cookie credentials and the browser unauthorized event in a web transport/session adapter. Add bearer injection and refresh through a mobile adapter. |
| `src/lib/queries.ts`                                                                      | Replace global-client query constants with server-bound factories in core. Web and mobile use the same keys, pagination, stale times, fetch behavior, and response selection.                                                |
| `src/lib/mutations.ts` and inline component mutations                                     | Move commands, mutation options, and cache update/invalidation semantics into core as each feature is touched. Keep toast presentation and router side effects in the applications.                                          |
| `src/lib/query-client.ts`                                                                 | Move shared retry, stale, hydration, mutation, and eventual offline policy into the runtime. Supply IndexedDB and SQLite persistence as adapters.                                                                            |
| `dates.ts`, `staleness.ts`, `recipeInputs.ts`, `keyedCollections.ts`, `queuedAutosave.ts` | Move immediately with their tests; these are already platform-independent behavior.                                                                                                                                          |
| React hooks and editor state machines                                                     | Share them when they depend only on React/core contracts. Keep hooks that read DOM, Expo, router, toast, or platform lifecycle APIs in the relevant application.                                                             |
| Components, routes, Radix/daisyUI/Tiptap code                                             | Keep in web. Mobile builds native presentation against the same queries, commands, and feature state.                                                                                                                        |

The migration rule is **extract, migrate web, then consume from mobile**. An extraction PR updates
web to import the shared implementation in the same change. A temporary old web path may re-export
from core to keep a diff manageable, but it may not retain a copied implementation. If a mobile
feature needs behavior that web currently implements inline, extracting that behavior is part of the
feature—not optional cleanup after mobile ships.

This intentionally means that most non-visual client machinery is shared:

- TypeSpec-generated types and API request/response handling;
- server identity, capabilities, authentication/session state, and error classification;
- TanStack Query keys, query/mutation options, pagination, invalidation, hydration, and persistence
  policy;
- command semantics and, eventually, outbox/sync/conflict state machines;
- domain transformations, recurrence/date behavior, validation, autosave, and ordered collection
  editing;
- widget projections and other constrained read models.

Web and mobile supply different implementations only at real platform boundaries:

- cookie session versus OAuth bearer credentials;
- IndexedDB versus SQLite/SecureStore and platform widget containers;
- browser versus native linking, connectivity, background execution, sharing, and lifecycle;
- DOM presentation versus SwiftUI/Compose presentation.

Core contracts describe capabilities the shared machinery actually needs; they must not expose DOM,
React Native, Expo, Swift/Kotlin, router, or toast types. Do not invent an abstraction until there
is shared behavior or an unavoidable platform boundary. When behavior truly differs, use two
explicit strategies behind the narrow contract instead of filling core with platform conditionals.

TypeSpec remains the sole serialized API contract. Generation emits the portable TypeScript client
artifacts into `packages/client-core`; web and mobile must not maintain separate handwritten
response types. The web application also adopts `serverId`-scoped query and persistence keys, using
its origin as the initial server profile, so later offline and multi-server work benefits both
applications.

## Runtime architecture

```text
TypeSpec -> OpenAPI -> packages/client-core
                         |              |
                         v              v
                    clients/web    clients/mobile
                                         |
                   +---------------------+---------------------+
                   |                     |                     |
              Expo UI screens       iOS WidgetKit       Android Glance
              (SwiftUI/Compose)      via expo-widgets    local Expo module
                   |                     ^                     ^
                   +------ query data -> versioned widget snapshot --------+
```

### Application state

- Expo Router owns route/deep-link state.
- TanStack Query owns remote server state. Every query key begins with `serverId`, for example
  `[serverId, "tasks", "mine", params]`.
- Small ephemeral editor state stays inside its feature or screen.
- Non-secret persisted metadata lives in a small, migrated SQLite schema.
- OAuth access/refresh credentials live in SecureStore and are keyed by both server and account.
- Widget render data lives in the platform's shared widget container and never contains credentials.
- Avoid a general global-state library until state appears that Router, Query, and local React state
  cannot represent clearly.

### Server identity and the single-server UI

The first UI asks for one server URL during onboarding and treats it as the active server.
Internally, persist it as a profile rather than as global scalar configuration:

```ts
type ServerProfile = {
  id: string; // client-generated stable ID
  baseUrl: string; // normalized URL, mutable without changing identity
  displayName: string;
  lastUsedAt: string;
};
```

Persist `activeServerId` separately. Key credentials, query caches, widget configuration, and any
future local records by `serverId`. Domain DTOs should remain identical to server responses; attach
server identity in cache/persistence envelopes instead of adding it to every API model.

The repository APIs should accept an explicit server context. They must not import a singleton URL.
The UI may expose only `profiles[0]` until multi-server interaction becomes a real project.

This gives a future server switcher a safe storage model without building account aggregation,
cross-server search, or simultaneous connections now.

### Authentication

Use the server's OAuth 2.1 authorization-code flow with PKCE, following the working CLI flow rather
than using the cookie-only web login API.

Early server work must:

- Register a public first-party `horologia-mobile` OAuth client with explicit app redirect URIs.
- Request user-facing scopes only; omit `admin:*` from the mobile client.
- Add a small unauthenticated server-information endpoint to TypeSpec containing an API
  compatibility version and capability identifiers. Keep `/healthz` as a liveness endpoint rather
  than turning it into product discovery.
- Verify authorization, refresh-token rotation, logout/revocation, OIDC-backed login,
  password-backed login, cancellation, and expired credentials on both platforms.

Use the system browser authentication session, Expo Linking, and SecureStore. Never place refresh
tokens in SQLite, AsyncStorage, logs, query persistence, or widget storage.

Production should support servers with valid HTTPS. During the foundation spike, decide how narrowly
development and local-network HTTP can be supported. Do not globally disable TLS validation or allow
arbitrary cleartext traffic merely to accommodate self-signed deployments; document the supported
deployment contract if safe platform-scoped exceptions are insufficient.

### Read cache and future offline writes

For v1, TanStack Query may persist selected read queries to improve cold starts and allow graceful
read-only stale views. Persistence is an optimization, not a second source of truth.

All writes flow through feature command functions such as `updateTask(serverId, input)`. The v1
implementation performs the request immediately and invalidates/updates affected queries. A future
outbox can implement the same command surface, but no outbox-specific states or schemas should be
added now.

Avoid:

- local IDs for server-owned objects;
- UI code that directly writes SQLite domain rows;
- “success” UI before the server has accepted a mutation;
- cache keys without a server namespace;
- destructive cache resets on ordinary connectivity errors;
- silently hiding stale data.

## UI architecture

The compact-window navigation matches the web app's current mobile information architecture:

- Tasks: My Tasks, task details, and task activity.
- Library: Spaces and Recipes, with each space opening its own task/recipe areas.
- Search: cross-space task and recipe search.
- Account: user settings, connection state, server information, and logout.

### First-release form factors

The layout responds to the current app window, not a one-time phone/tablet device check. A tablet in
split-screen can be compact; a foldable or desktop window can move between compact and expanded
while the app is running. Route, selection, scroll, and unsaved editor state must survive those
changes.

Use three semantic layout modes:

- Compact: one navigation destination or detail pane at a time, with bottom/tab navigation.
- Medium: wider content and an adaptive rail/sidebar where it improves navigation.
- Expanded: persistent navigation plus list/detail panes for tasks, recipes, spaces, search, and
  settings where useful.

Apple requirements:

- Set `ios.supportsTablet` to `true` and leave full-screen-only mode disabled so iPad multitasking
  and resizable windows work.
- Treat iPad portrait, landscape, Split View, and Stage Manager sizes as supported configurations.
- Support keyboard navigation, pointer/trackpad input, hover/context menus where native controls
  provide them, and sensible toolbar placement.
- Keep the iPad app available on Apple-silicon Macs. Test it directly with Xcode's iOS-app-on-Mac
  destination, including authentication, file/link opening, keyboard shortcuts, pointer behavior,
  window resizing, and widgets/deep links where the iOS runtime exposes them.
- Do not promise Mac-specific widgets, menu-bar integration, AppKit controls, Intel Mac support, or
  other native macOS features from this compatibility build.

Android requirements:

- Use one resizable activity and do not lock orientation or aspect ratio.
- Treat phones, tablets, foldables, ChromeOS, split-screen, and desktop windowing as supported
  configurations of the same binary.
- Meet Android's Adaptive Optimized (Tier 2) guidance for relevant flows: multi-pane layouts,
  continuity across resize/posture changes, and keyboard, mouse, trackpad, and stylus basics.
- Use current Material 3 adaptive patterns for navigation and list/detail. If a required adaptive
  scaffold is not exposed by Expo UI, add it to the local Expo UI module rather than reimplementing
  the pattern with legacy Views.

Use Expo UI universal primitives for ordinary text, lists, fields, toggles, pickers, and layout when
their behavior is strong on both platforms. Prefer direct platform APIs for navigation bars, search,
complex sheets, context menus, toolbar placement, and any newer system affordance whose
cross-platform abstraction loses fidelity.

Create thin Horologia wrappers only for repeated product semantics such as `TaskRow`, `PropertyRow`,
`EmptyState`, `ErrorState`, and `SaveBar`. Do not create a parallel button/input/card system over
Expo UI.

For Markdown, begin with a native read view and a simple native multiline editor. Evaluate an Expo
DOM component for rich editing only if the basic editor is materially worse; keep the serialized
format Markdown and keep autosave behavior portable. Recipe reordering should use a native-feeling
interaction and does not need to share the web drag-and-drop implementation.

## Widget architecture

### Product scope

The first widget family is **My Tasks**:

- Small: overdue/due count plus the next task.
- Medium: the next few assigned tasks across spaces.
- Large: a longer task list grouped or annotated by space.
- Tap a task to deep-link to its detail screen.
- Clearly show when the snapshot is stale or the user needs to sign in.
- Follow system light/dark, Dynamic Type where supported, accessibility, and platform widget quality
  guidance.

The MVP widget is read-only except for deep links. Interactive completion/status actions are the
next widget increment, after their recurrence, error, authentication, and refresh semantics are
proven in the full app. A button that appears to complete a task but cannot reliably report server
failure is not acceptable.

### Shared data, separate renderers

Define a small versioned `WidgetSnapshotV1` in `packages/client-core` containing already-projected,
sanitized display data:

```ts
type WidgetSnapshotV1 = {
  version: 1;
  serverId: string;
  accountId: string;
  generatedAt: string;
  tasks: Array<{
    id: string;
    spaceSlug: string;
    title: string;
    due: string | null;
    status: string;
  }>;
};
```

The main app produces this snapshot after relevant fetches and successful mutations. Renderers do
not reproduce task filtering, recurrence, or ordering rules. The snapshot contains no token, server
secret, private profile fields, or arbitrary cached API response.

On iOS, use `expo-widgets` and Expo UI's SwiftUI widget components. Its widget runtime is isolated
and synchronous, so pass all data as widget props/timeline snapshots. On Android, implement an app
widget with Jetpack Glance in `modules/horologia-android-widget`; Glance uses a Compose-style API
but is not regular Compose UI and should remain a separate renderer.

Refresh is opportunistic because both operating systems control widget execution budgets. The app
should update snapshots when it is foregrounded, after task mutations, and through permitted
background work. A cached snapshot remains useful without network access and should display its age
instead of going blank.

Widget configuration records include `serverId` even while the UI has one server. The first release
can hard-code the “My Tasks” view, but the schema should allow a future widget to select a server,
space, assignee, or saved filter without changing the identity model.

### Widget release gate

Before any public release, both platforms must demonstrate in the installed iOS Simulator and
Android Emulator environments that widgets:

- install from the system gallery;
- render every supported size;
- survive app/process termination and simulator/emulator restart;
- show a last-known snapshot without connectivity;
- update after an in-app mutation within platform constraints;
- deep-link to the correct server-scoped task;
- behave correctly after logout, token expiry, and account deletion;
- meet platform accessibility, contrast, touch-target, and preview requirements.

## Feature parity target

Parity means user-visible behavior and permissions, not identical layout or implementation.

| Area             | Mobile v1 scope                                                                                                                                                               | Milestone |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------- |
| Authentication   | Server onboarding, OAuth/PKCE, refresh, logout, session recovery                                                                                                              | 1         |
| My Tasks         | Cross-space assigned task list, due/overdue treatment, activity entry point                                                                                                   | 2         |
| Widgets          | iOS WidgetKit and Android Glance My Tasks families, deep links, stale state                                                                                                   | 2         |
| Tasks            | Space lists; create/read/update/delete; description; status; effort; priority; due date/timezone; recurrence; rotation; assignees; tags; overdue actions; relations; activity | 3         |
| Search           | Cross-space task and recipe search with native search UI                                                                                                                      | 3         |
| Spaces           | List/create; task and recipe modules; activity                                                                                                                                | 4         |
| Recipes          | Cross-space/space lists; search; create/read/update/delete; yield/timing/tags; ingredient and instruction sections; Markdown; reordering; activity                            | 4         |
| Space settings   | General details, members/roles, statuses, effort levels, priority levels, tags, delete space                                                                                  | 5         |
| Account settings | Profile, system/light/dark appearance, password, API tokens, delete account, server information                                                                               | 5         |
| Global admin     | Owner user management and admin About view                                                                                                                                    | Excluded  |
| Notifications    | Push and scheduled local notifications                                                                                                                                        | Deferred  |
| Offline writes   | Outbox, sync, conflicts, local-first editing                                                                                                                                  | Deferred  |

## Milestones

Milestones are ordered by risk and product value, not by estimated duration. Do not begin a broad
parity port until the widget and native UI assumptions have survived Milestone 0.

### Milestone 0 — architecture spike

Build one disposable-but-reusable vertical slice in the installed iOS Simulator and Android Emulator
environments:

- Create the minimum `packages/client-core` by moving the generated schema, client factory,
  server-scoped query machinery, and already-portable helpers out of web. Migrate web in the same
  changes before mobile consumes them.
- Scaffold `clients/mobile` on the current stable Expo SDK with strict TypeScript, pnpm workspace
  support, Expo Router, and development builds.
- Enable iPad support and render the My Tasks list/detail shell at compact and expanded widths.
- Render the same adaptive My Tasks flow on the installed `Resizable_Experimental` Android virtual
  device across phone, tablet, and resizable window sizes.
- Deliberately exercise one platform-specific SwiftUI feature and one current Material 3 feature,
  plus Dynamic Type, dark mode, reduced motion, and screen-reader labels.
- Complete an OAuth/PKCE round trip against the local Horologia server using a temporary mobile
  client registration.
- Generate and render a static then app-provided iOS widget snapshot with `expo-widgets`.
- Build the equivalent Android Glance widget in a local Expo module.
- Deep-link from both widgets to the task-detail route.
- Run the iPad build directly as an iOS app on this Apple-silicon Mac and exercise authentication,
  keyboard/pointer input, window resizing, and deep links.
- Build and install both apps from a clean checkout.

Exit criteria:

- Expo UI provides a credible full-screen native foundation on both platforms.
- Web and mobile consume the same generated types, API client factory, server-scoped query
  definitions, and portable domain helpers; no parallel mobile implementation exists.
- The same Apple binary works as an adaptive iPhone app, iPad app, and iPad app on Apple-silicon
  macOS without a macOS/Catalyst target.
- The same Android binary works in compact, tablet, foldable-style, and resizable desktop window
  configurations without activity recreation losing user state.
- The Android widget can be built and updated without ad hoc edits inside generated Gradle output.
- The iOS widget extension can be reproduced from app config/source.
- The team chooses either Continuous Native Generation with generated `ios/` and `android/` folders,
  or committed native projects, based on which path actually keeps widget customization
  deterministic. CNG is preferred only if the clean regeneration test passes.
- Known Expo UI gaps have contained local-module solutions; none require a second application
  architecture.

If the spike finds a missing component, extend Expo UI or use a contained React Native/DOM island.
Do not revisit the overall framework choice unless the app/widget lifecycle itself is blocked.

### Milestone 1 — shared runtime, foundation, and authentication

- Complete the shared runtime composition, bearer-capable mobile auth strategy, browser cookie auth
  strategy, platform persistence adapters, and common Query defaults.
- Add the server-info/capabilities API and permanent `horologia-mobile` OAuth registration.
- Implement server URL onboarding, normalized `ServerProfile` persistence,
  reachability/compatibility checks, OAuth, refresh coordination, secure logout, and account
  recovery states.
- Establish the Tasks/Library/Search/Account route shell and platform-adaptive navigation.
- Establish semantic colors, native icon selection, typography rules, standard screen states,
  haptics policy, accessibility checks, and compact/medium/expanded screenshot fixtures.
- Wire `clients/mobile` into pnpm, mise generation/check/test tasks, and dependency update policy.

Exit criteria: a clean install can connect to a supported server, authenticate with password- or
OIDC-backed web authorization, restore the session after restart, refresh credentials, and log out
without leaving credentials or widget data behind.

### Milestone 2 — widget-backed My Tasks MVP

- Implement the complete My Tasks list, refresh, pagination, empty/error/stale states, and task
  detail read view.
- Implement `WidgetSnapshotV1`, update triggers, platform storage adapters, all initial widget
  sizes, stale/signed-out presentations, and deep links.
- Update the snapshot after task queries and accepted task mutations.
- Distribute internal iOS and Android builds and test launcher/widget behaviors in the installed
  simulator/emulator environments across the supported OS range.

Exit criteria: the app is already useful as a task viewer and both widgets meet the widget release
gate. This is the first meaningful dogfood release.

### Milestone 3 — task workflows

- Add task creation and deletion.
- Extract existing task commands, query/cache semantics, recurrence logic, and portable editor state
  from web before mobile consumes them; keep both clients on the shared implementations.
- Add native editors for every task field, including recurrence and overdue actions.
- Add relations and task activity.
- Add cross-space task/recipe search.
- Add reliable mutation error recovery and widget refresh after relevant writes.
- Evaluate and then add safe interactive widget actions; keep deep-link-only behavior if platform
  failure feedback is not trustworthy.

Exit criteria: daily task work no longer requires the web app, including advanced recurring task
configuration.

### Milestone 4 — spaces and recipes

- Add space creation, space task lists, space activity, and adaptive space navigation.
- Extract existing portable space and recipe commands, query/cache behavior, parsers, ordering, and
  autosave machinery from web before mobile consumes them.
- Add recipe lists, search, detail, creation, editing, deletion, sections, ordering, and activity.
- Preserve Markdown as the wire format and validate the chosen editing experience on both platforms.
- Add share-sheet support for useful task/recipe links where server URLs are safe to expose.

Exit criteria: the mobile app covers the web app's main Tasks, Spaces, Recipes, Search, and Activity
product surfaces.

### Milestone 5 — settings parity

- Add space general settings, member roles, statuses, priority/effort levels, tags, and danger zone.
- Add account profile, appearance, password, API-token, server-info, logout, and delete-account
  controls.
- Keep global owner administration out of the app unless a later product decision adds it.

Exit criteria: all non-global-admin web workflows have either mobile parity or a recorded,
intentional exclusion.

### Milestone 6 — production release

- Complete accessibility audits with VoiceOver and TalkBack, large text, high contrast, RTL layout,
  reduced motion, and keyboard/pointer/tablet navigation.
- Test slow, captive, intermittent, and unavailable networks; expired/revoked credentials; server
  upgrades; incompatible API versions; and process death.
- Establish crash reporting and privacy-preserving diagnostics without requiring a cloud service for
  core operation.
- Automate signed Android and iOS release builds, store metadata, privacy declarations, dependency
  license notices, versioning, and upgrade rehearsal.
- Run the complete parity checklist and widget release gate in the installed iOS Simulator and
  Android Emulator environments. A physical-device smoke test is useful but is not a release gate.
- Run the iPad build as an iOS app on this Mac and complete the Mac compatibility checklist. This
  does not create or gate on a native macOS target.

Exit criteria: production builds are reproducible, independently usable with a self-hosted server,
and ready for the App Store and Play Store.

## Testing and delivery

### Test layers

- `packages/client-core`: fast unit tests for server URL identity, auth refresh coordination, query
  key scoping, recurrence/date transformations, command behavior, and widget projection/versioning.
- `clients/mobile`: component tests only for feature-specific state and accessibility behavior; do
  not snapshot every native control.
- API/server: real HTTP/PostgreSQL integration tests for the mobile OAuth client, scopes,
  capabilities endpoint, token rotation/revocation, and permission boundaries.
- Native modules: focused Kotlin tests for Android widget storage/projection boundaries and an
  Android Emulator Glance smoke test; iOS widget timeline/serialization tests plus an iOS Simulator
  smoke test.
- End to end: a small Maestro suite on iOS and Android for onboarding, login, My Tasks, a task
  mutation, widget deep link, recipe editing, and logout.
- Adaptive matrix: run core navigation and editing flows at iPhone, iPad portrait/landscape and
  split-window widths, Android phone/tablet/resizable widths, and the iOS-app-on-Mac destination.
  Verify continuity while resizing rather than only taking static screenshots.

Every shared persistence test must include at least two synthetic `serverId` values and prove that
credentials, caches, deep links, and widget snapshots do not cross namespaces even though the UI
only exposes one server.

### Repository tasks

Add contributor-facing mise tasks rather than documenting raw Expo commands as the main workflow:

```text
mise run //clients/mobile:dev
mise run //clients/mobile:ios
mise run //clients/mobile:android
mise run //clients/mobile:generate
mise run //clients/mobile:check
mise run //clients/mobile:test
mise run //clients/mobile:e2e
mise run //clients/mobile:build
```

Root `generate`, `check`, and `test` must include portable/mobile work that can run on the current
host. Native build jobs should be explicit by platform: Android on Linux and/or macOS, iOS on macOS.
Pin Node, pnpm, Java, Android command-line tools, formatters, and test runners through the existing
mise/lockfile conventions; Xcode remains host-managed. Install dependencies from the frozen root
lockfile.

EAS Build/Submit may be used when it reduces release friction, but local and CI builds must remain
possible without EAS. Treat OTA updates as a separate security/operations decision, not a condition
of the initial architecture.

## Platform strategy

Portable TypeScript and TypeSpec contracts are the long-lived assets. Expo, Expo UI, and each native
integration are platform adapters.

| Platform                                                                      | Decision                                                                                                                                                                                                                                                                     |
| ----------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| iPhone and iPad                                                               | First-release native Expo target. One adaptive iOS binary with WidgetKit widgets.                                                                                                                                                                                            |
| Android phones, tablets, foldables, ChromeOS, and desktop windows             | First-release native Expo target. One adaptive Android binary with Glance widgets where the host launcher supports them.                                                                                                                                                     |
| Apple-silicon macOS                                                           | First-release compatibility target using the unchanged iPad binary distributed as an iPhone/iPad app on Mac. Test and support its large-screen, keyboard, pointer, resize, auth, and deep-link behavior. It is not a native macOS app.                                       |
| Native macOS                                                                  | Deferred. React Native macOS is a separate AppKit out-of-tree platform and Expo-module support on it is experimental. A future native Mac app would be a separate `clients/macos` project consuming `packages/client-core`, not another target forced into `clients/mobile`. |
| Web                                                                           | Existing React/TanStack web app remains the primary browser and full desktop experience, sharing client core and advanced persistence/offline machinery.                                                                                                                     |
| Windows, Linux/postmarketOS, HarmonyOS/OpenHarmony, TV, vision, and wearables | Future product decisions. Use the web app where practical and preserve the shared core; research and choose each platform's UI/runtime when that platform enters the roadmap. Do not select React Native or create adapters for them now.                                    |

The native macOS decision is final for this roadmap: do not create a Mac Catalyst target, run
`react-native-macos-init`, or audit all Expo dependencies for AppKit during the mobile project.
React Native macOS requires a separate `macos/` project aligned to its React Native fork; Expo's own
documentation calls macOS support for Expo modules experimental, and Expo Router's supported
platforms do not include macOS. The iPad-on-Mac path provides the useful low-cost compatibility win
without adopting that experimental toolchain.

Use platform file suffixes and narrow adapter contracts for storage, browser authentication, widget
publication, background work, sharing, and haptics. Do not add empty implementations for platforms
that are not being built. The contract is introduced when a second real implementation exists or
when the current platform boundary is already unavoidable.

## Deferred tracks

### Notifications

Notifications remain deferred. A self-hosted service cannot assume a vendor push relay, reliable
inbound connectivity, or continuously running background delivery. When revisited, decide separately
among local notifications computed from fetched data, an optional Horologia push relay, direct
APNs/FCM credentials owned by each server administrator, and platform-specific background refresh.
Do not let notification tokens or assumptions enter the v1 server schema.

### Offline writes

Offline writes require a product and server protocol, not just SQLite: stable mutation identity,
idempotency, a change feed or version checks, conflict semantics, permission changes, deleted
objects, and user-visible pending/failed states. The v1 command boundary and server-scoped
persistence preserve room for that work. They are not an incomplete sync engine.

### Multiple servers

When real demand appears, add server management, per-widget server selection, and explicit account
switching on top of `ServerProfile`/`serverId`. Cross-server aggregation and search are separate
features and should not be implied by a server switcher.

## Risks and responses

| Risk                                                      | Response                                                                                                                                                                                             |
| --------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Expo UI is stable but still a young full-app surface      | Prove the hardest native interactions in Milestone 0, keep wrappers thin, and extend through local SwiftUI/Compose modules when necessary.                                                           |
| Widget APIs have different execution and rendering models | Share only a versioned projection; keep WidgetKit and Glance renderers separate; test the installed simulator/emulator environments early.                                                           |
| Native project generation becomes fragile                 | Require clean regeneration in the spike. Keep customization in app config, config plugins, module manifests/resources, and local modules; commit native projects if generation is not deterministic. |
| Reusing web code imports DOM assumptions                  | Move only proven runtime-independent modules; use generated contracts as the primary reuse mechanism.                                                                                                |
| Rich Markdown and ordered recipe editors feel non-native  | Start simple, test on both platforms, and use a bounded DOM/native island only where it wins materially.                                                                                             |
| Self-hosted URLs conflict with mobile transport security  | Make HTTPS the production baseline and explicitly test/document any narrowly supported local-network exception.                                                                                      |
| A future server leaks data from the current one           | Namespace all persisted/query/widget/auth state by `serverId` from day one and test with two IDs.                                                                                                    |
| Expo/RN upgrades move quickly                             | Pin the SDK/toolchain, use frozen installs, upgrade one Expo SDK at a time, and keep a simulator/emulator upgrade rehearsal.                                                                         |

## Definition of done for a feature

A mobile feature is complete when:

- it implements the intended web behavior and permission boundary, or documents an intentional
  mobile difference;
- loading, empty, stale, offline, error, retry, and destructive-confirmation states are designed;
- it uses native platform semantics and works at supported phone/tablet sizes;
- it preserves state while moving between compact, medium, and expanded windows;
- it supports screen readers, large text, dark mode, reduced motion, and keyboard/focus behavior
  where applicable;
- routes and deep links restore the correct server-scoped state;
- portable behavior is implemented once in `packages/client-core`, with web migrated before mobile
  consumes it and only actual platform capabilities left in adapters;
- writes pass through the command boundary and update/invalidate server-scoped queries;
- relevant widget projections update after accepted mutations;
- valuable core/component/integration/end-to-end tests pass through mise tasks;
- it has been exercised in the installed iOS Simulator and Android Emulator environments before
  release, including iPad/tablet/resizable configurations;
- Apple flows have also been exercised with the iPad build running as an iOS app on this Mac.

## First implementation tranche

The first implementation PRs should be narrow and independently reviewable:

1. Add `packages/client-core` by moving shared TypeScript API generation, the client factory,
   server-scoped queries, Query defaults, and portable helpers out of web; migrate web in the same
   changes.
2. Add the mobile OAuth client, compatibility/capabilities endpoint, and integration tests.
3. Scaffold `clients/mobile`, Expo development builds, mise tasks, and a native route shell that
   consumes `packages/client-core`.
4. Enable iPad and Android adaptive layouts, add compact/medium/expanded shell fixtures, and verify
   the iPad build as an iOS app on this Mac.
5. Implement server onboarding and OAuth/PKCE with secure credential persistence through the shared
   runtime and mobile adapters.
6. Extract any remaining portable My Tasks behavior from web, then implement the mobile read-only My
   Tasks vertical slice and native task-detail shell against it.
7. Implement `WidgetSnapshotV1` and the iOS `expo-widgets` proof.
8. Implement the Android Glance local module and prove clean native regeneration.
9. Record the Milestone 0 findings and make the CNG-versus-committed-native-project decision before
   expanding into feature parity.

## Current primary references

- [Expo UI](https://docs.expo.dev/versions/latest/sdk/ui/) and its
  [universal native components](https://docs.expo.dev/versions/latest/sdk/ui/universal/)
- [Building SwiftUI apps with Expo UI](https://docs.expo.dev/guides/expo-ui-swift-ui/)
- [Jetpack Compose components in Expo UI](https://docs.expo.dev/versions/latest/sdk/ui/jetpack-compose/)
  and
  [extending Expo UI with Compose](https://docs.expo.dev/guides/expo-ui-jetpack-compose/extending/)
- [Expo monorepo support](https://docs.expo.dev/guides/monorepos/)
- [Expo iPad configuration](https://docs.expo.dev/versions/latest/config/app/)
- [Expo Widgets](https://docs.expo.dev/versions/latest/sdk/widgets/)
- [Jetpack Glance](https://developer.android.com/develop/ui/compose/glance)
- [Android adaptive app guidance](https://developer.android.com/develop/adaptive-apps/guides/get-started-with-adaptive-apps)
- [Android large-screen quality](https://developer.android.com/docs/quality-guidelines/large-screen-app-quality)
- [Local and standalone Expo modules](https://docs.expo.dev/more/create-expo-module/)
- [React Native out-of-tree platforms](https://reactnative.dev/docs/next/out-of-tree-platforms)
- [React Native macOS](https://microsoft.github.io/react-native-macos/docs/intro) and
  [React Native Windows](https://microsoft.github.io/react-native-windows/)
- [Experimental Expo modules on React Native macOS](https://microsoft.github.io/react-native-macos/docs/guides/installing-expo-modules)
- [Expo additional platform support](https://docs.expo.dev/modules/additional-platform-support/)
- [Running iPhone and iPad apps on Apple-silicon Macs](https://developer.apple.com/documentation/apple-silicon/running-your-ios-apps-in-macos/)
- [OpenHarmony SIG React Native port](https://gitee.com/openharmony-sig/ohos_react_native)

These links describe a fast-moving ecosystem. Revalidate SDK versions, supported components, widget
constraints, and out-of-tree platform compatibility at the start of each implementation milestone.
