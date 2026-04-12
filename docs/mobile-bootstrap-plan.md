# Mobile Bootstrap Plan

## Goal

Bootstrap a native-mobile codebase for Tend with:

- one thick Kotlin Multiplatform core module for shared mobile logic
- one thin Compose Multiplatform app module with Android and desktop targets
- toolchains and tasks managed through `mise`

This document covers the initial tooling decision and the proposed repository/build layout. It does
not commit us to a full mobile implementation yet.

Current bootstrap status:

- `mobile/` now exists with `:core` and `:compose-app`
- root `mise.toml` owns the tools; `mobile/mise.toml` owns the mobile tasks
- the generated KMP client is committed under `mobile/core/src/commonMain/generated`
- `openapi-kmp-gen` currently needs one small compatibility step for Tend: inline reusable OpenAPI
  parameter refs before generation

## Recommendation

Use the existing TypeSpec definition as the source of truth, continue emitting OpenAPI from it, and
use **`openapi-kmp-gen` as the first bootstrap spike**.

In practice:

1. TypeSpec remains the canonical contract.
2. `api/` continues to emit OpenAPI.
3. `mobile/` consumes the emitted OpenAPI and generates a Kotlin Multiplatform client.
4. The generated client lives inside the shared KMP layer, not as its own user-facing app-facing
   surface.
5. If the generator becomes a problem later, migrate mechanically to a different client generator or
   a handwritten client then.

## Research Summary

### TypeSpec

TypeSpec's official codegen story does not currently include a Kotlin client emitter.

- The TypeSpec 1.0-RC announcement lists preview HTTP client generators for C#, JS, Java, and
  Python, but not Kotlin.
- TypeSpec's emitter framework is explicitly positioned for custom emitters when built-in output is
  missing.

Implication: generating KMP directly from TypeSpec would mean building and maintaining a custom
emitter. That may become the right long-term move, but it is too much for bootstrap.

### `openapi-kmp-gen`

`openapi-kmp-gen` is a purpose-built KMP OpenAPI generator with an official Gradle plugin and CLI.
Its README advertises:

- Kotlin Multiplatform support
- Ktor-based generated clients
- kotlinx-serialization for JSON
- kotlinx-datetime for date types
- security support
- filtering generated APIs by tag

Its own current dependency matrix is also materially more modern than the documented multiplatform
stack in OpenAPI Generator.

Implication: this is the best first thing to try for Tend.

### OpenAPI Generator

OpenAPI Generator remains the fallback, not the first choice.

Reasons:

- it is more broadly known and widely used
- but its documented Kotlin multiplatform template is older
- and its official feature matrix still shows notable gaps around some OAS 3 schema features

Implication: keep it as a backup plan if `openapi-kmp-gen` fails on Tend's spec.

## Why This Is Still The Right Bootstrap

For Tend today, the alternatives are worse:

- **Custom TypeSpec emitter now**: best eventual fit, but too much up-front work
- **Handwritten KMP client now**: viable, but duplicates boilerplate before we've proven the mobile
  shape
- **Generate Java and wrap it**: not acceptable for iOS

`openapi-kmp-gen` is the shortest path to learning whether Tend's API shape works cleanly in KMP.

## Proposed Repository Layout

Create a new top-level `mobile/` package root, parallel to `api/`, `server/`, `cli/`, and `web/`.

```text
mobile/
  settings.gradle.kts
  build.gradle.kts
  gradle.properties
  gradle/libs.versions.toml
  gradle/wrapper/...
  gradlew
  gradlew.bat
  mise.toml
  core/
    build.gradle.kts
    src/commonMain/generated/...
    src/commonMain/kotlin/...
    src/commonTest/kotlin/...
    src/androidMain/kotlin/...
    src/iosMain/kotlin/...
  compose-app/
    build.gradle.kts
    src/commonMain/kotlin/...
    src/androidMain/AndroidManifest.xml
    src/androidMain/kotlin/...
    src/desktopMain/kotlin/...
```

Alternative layout if generation pressure makes separation useful later:

```text
mobile/
  client/
    build.gradle.kts
    src/commonMain/generated/...
    src/commonMain/kotlin/...
  core/
    build.gradle.kts
    src/commonMain/kotlin/...
    src/commonTest/kotlin/...
    src/androidMain/kotlin/...
    src/iosMain/kotlin/...
  compose-app/
    build.gradle.kts
    src/commonMain/...
    src/androidMain/...
    src/desktopMain/...
```

The default recommendation is the simpler shape: **`:core` includes the generated client**.

## Module Responsibilities

### `:core`

Kotlin Multiplatform library module and the main shared mobile runtime.

Owns:

- generated API client code
- auth/session abstractions
- DTO-to-domain mapping
- repositories and use-case logic
- local persistence interfaces
- sync logic
- shared view-model/state-holder logic as appropriate

This should be the thick layer. Platform apps should stay thin and mostly compose UI plus platform
integration.

### Optional `:client`

Only introduce a separate `:client` module if the generator or compile boundaries make it materially
useful.

If it exists, it should own only:

- generated API client code
- generation task wiring

Then `:core` depends on `:client`.

### `:compose-app`

Compose Multiplatform application module with Android and desktop targets.

Owns:

- shared Compose UI shell
- Android entrypoint and Android application wiring
- desktop entrypoint and packaging
- platform-specific integrations that should stay outside the shared core

The app should depend on `:core`, not on generated sources directly.

## Gradle Structure

### `mobile/settings.gradle.kts`

Use the simpler shape from `spatial-k` plus the ergonomics from `maplibre-compose`:

- `rootProject.name = "tend-mobile"`
- `enableFeaturePreview("TYPESAFE_PROJECT_ACCESSORS")`
- `pluginManagement` repositories: `google()`, `mavenCentral()`, `gradlePluginPortal()`
- `dependencyResolutionManagement` repositories: `google()`, `mavenCentral()`
- include `:core` and `:compose-app`
- optionally include `:client` later if generation needs to be split out
- optionally add the Foojay resolver convention plugin for JDK toolchains

### `mobile/build.gradle.kts`

Keep the root intentionally thin.

Initial responsibilities:

- shared repository-wide configuration
- aggregate tasks only if needed
- no heavy build logic plugin on day one

`maplibre-compose` uses custom convention plugins. That is a good direction later, but it is more
indirection than Tend needs for a two-module bootstrap.

### `mobile/gradle/libs.versions.toml`

Use a version catalog from day one.

This matches the `maplibre-compose` style and keeps Kotlin/AGP/Compose/Coroutines/Ktor versions in
one place. It also makes the inevitable KMP compatibility matrix easier to reason about.

At minimum, the version catalog should define:

- Kotlin
- Android Gradle Plugin
- Compose BOM or Compose libraries
- Kotlinx Coroutines
- `openapi-kmp-gen` version
- test libraries

## How To Handle Generated Code

Generated code should live in `src/`, be committed, and follow the same repo conventions as the
existing generated code in `api/gen/` and `server/internal/mcp/gen/`.

Example shape:

```text
mobile/core/src/commonMain/generated/...
```

Recommended conventions:

- generated sources live under a clearly named generated source root
- generated sources are committed to git
- `.gitattributes` should mark them as generated for diff/language stats purposes
- regeneration should be deterministic and happen through `mise run generate`

## Proposed Codegen Flow

The mobile codegen should consume the emitted OpenAPI spec, not TypeSpec directly.

Proposed flow:

1. `mise run //api:generate`
2. OpenAPI spec is emitted from TypeSpec
3. `mise run //mobile:generate`
4. `mobile` preprocesses the emitted spec into the subset `openapi-kmp-gen` handles today, then runs
   the generator
5. generated Kotlin sources are written into `mobile/core/src/commonMain/generated`
6. `:core` compiles generated code plus handwritten shared mobile code

Important note: Tend's root `mise run generate` currently aggregates package-scoped generate tasks.
Once `mobile/` exists, the mobile generate task should explicitly depend on `//api:generate`, so
OpenAPI exists before client generation runs.

Default mobile `build` / `test` / `check` tasks should stay scoped to host-safe compile steps in
`:core` plus the desktop target in `:compose-app` so the root monorepo tasks do not require Android
device/emulator setup or Xcode link/test setup. Android assembly/install remains explicit under
`//mobile:android:*`.

## KMP Target Strategy

### Initial library targets

The `:core` module should be KMP from the start, with:

- `androidTarget()`
- `iosArm64()`
- `iosSimulatorArm64()`
- `iosX64()` if still needed by the current Apple toolchain setup

This keeps the shared library honest even before an iOS app exists.

### Initial app targets

The first app module should be a **Compose Multiplatform app** with:

- `androidTarget()`
- `jvm("desktop")`

That keeps the app layer thin while giving us one extra non-mobile target for fast shared UI
iteration without committing to SwiftUI packaging yet.

## App Conventions

The app module should be a Compose Multiplatform application:

- Kotlin Multiplatform plugin
- Android application plugin
- Kotlin Compose compiler plugin
- `org.jetbrains.compose` plugin for shared UI and desktop packaging
- Android-specific wiring kept in `androidMain`
- desktop-specific wiring kept in `desktopMain`

## `mise` Plan

Add `mobile` as a monorepo config root in the repository root `mise.toml`.

Keep tool versions in the repository root `mise.toml`. Use `mobile/mise.toml` for package-local
tasks only.

### Recommended root tools

- `java = "21"`
- `android-sdk`
- `hk`
- `pkl`
- `ktfmt` via `aqua:` or `npm:` to match the style used in your other KMP repos

Use a `mise`-managed Android SDK root and keep component installation explicit through
`mise run //mobile:setup`. That avoids hidden postinstall behavior while still making Android
tooling reproducible.

If and when we start compiling iOS targets regularly in this repo, add the same optional macOS
`xcodes` pattern you use in `maplibre-compose`.

### Recommended tasks

`mobile/mise.toml` should define at least:

- `generate` -> run KMP client generation from the emitted OpenAPI spec
- `setup` -> provision Android SDK components into the `mise`-managed SDK root
- `build` -> host-safe compile steps for `:core` plus `:compose-app` desktop
- `test` -> host-safe shared test compile steps
- `android:assembleDebug` -> build the debug APK
- `android:installDebug` -> install on a connected device or emulator
- `desktop:run` -> run the desktop app target
- `check` -> host-safe shared mobile compile checks
- `fix` -> formatting for Kotlin/Gradle files

## Conventions Borrowed From Your Other Repos

### From `spatial-k`

- keep the root simple
- package-local `mise.toml`
- use `mise` as the task entrypoint

### From `maplibre-compose`

- use `gradle/libs.versions.toml`
- use `TYPESAFE_PROJECT_ACCESSORS`
- prefer a KMP library plus consumer app split
- keep Java in `mise`
- use Kotlin-specific formatting tooling instead of pretending the web formatter can cover `.kt`

### What I would not copy yet

- custom convention plugins
- iOS app packaging and SPM export

Those make sense later, not for the first scaffold.

## Risks And Mitigations

### Risk: `openapi-kmp-gen` fails on Tend's emitted OpenAPI or has generator gaps

Mitigation:

- treat generation as a spike first
- commit generated code so failures and deltas are inspectable in review
- preserve the option to switch to OpenAPI Generator or a custom TypeSpec emitter later

### Risk: Generated code shape does not match Tend's nullable patch semantics well

Mitigation:

- validate this immediately with a real generator spike against Tend's spec
- only add handwritten adaptation when a concrete problem appears

### Risk: Mobile build complexity spills into the existing root

Mitigation:

- keep `mobile/` as an isolated Gradle build and `mise` config root
- avoid adding Gradle to the repo root

## Recommended Next Implementation Steps

1. Add `mobile` to the root `mise` monorepo config.
2. Scaffold a standalone Gradle build under `mobile/`.
3. Add `:core` and `:compose-app`.
4. Wire `mobile/mise.toml` to Java 21 and Gradle wrapper tasks.
5. Add an `openapi-kmp-gen` task that consumes the emitted Tend OpenAPI spec and writes committed
   generated sources under `src/`.
6. Add `.gitattributes` entries for the generated mobile client code.
7. Evaluate the generated client quality before building real repositories or screens on top of it.

## Decision

Proceed with:

- **source of truth**: TypeSpec
- **bootstrap generator**: `openapi-kmp-gen`
- **generated code convention**: committed generated sources under `src/`, marked in
  `.gitattributes`
- **repository shape**: new top-level `mobile/` package root
- **initial modules**: `:core` KMP library and `:compose-app` Compose Multiplatform app
- **module split**: keep the generated client inside `:core` initially; add `:client` only if a
  concrete generator/build boundary makes that worthwhile
- **toolchain ownership**: Java, Android SDK setup, and Kotlin formatting in `mise`; Android SDK
  component installation stays explicit under `//mobile:setup`

If the generator spike goes badly, the fallback order should be:

1. OpenAPI Generator Kotlin multiplatform
2. custom TypeSpec emitter
