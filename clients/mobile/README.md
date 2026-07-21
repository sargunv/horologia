# Horologia mobile

The mobile application uses a Kotlin Multiplatform shared module, a Jetpack Compose Android
application, and a tracked SwiftUI Xcode project. WidgetKit and Glance extensions consume snapshots
projected by the shared core.

Run mobile workflows from anywhere in the repository with these mise tasks:

- `mise run //clients/mobile:generate` — emit `api/gen/kmp/openapi.yaml`, then generate the Kotlin
  API client in `api-generated`.
- `mise run //clients/mobile:check` — run Kotlin and Gradle verification.
- `mise run //clients/mobile:test` — run shared tests on the Android host target and, on macOS, the
  iOS Simulator target.
- `mise run //clients/mobile:build` — build `api-generated`, `shared`, and `androidApp`.
- `mise run //clients/mobile:android` — install and launch the debug app on the connected Android
  device. A single ready device is selected automatically; select among multiple devices with
  `ANDROID_SERIAL=<serial> mise run //clients/mobile:android`.
- `mise run //clients/mobile:ios` — build `iosApp/Horologia.xcodeproj`, install the app, and launch
  it in Simulator. The task prefers a booted simulator and otherwise selects the first available
  iPhone; override it with `IOS_SIMULATOR_UDID=<udid> mise run //clients/mobile:ios`.
- `mise run //clients/mobile:clean` — remove Gradle and Xcode build outputs.

The Android task requires a connected device or emulator visible to the Android SDK. The iOS task
requires Xcode, its command-line tools, and an installed iOS simulator runtime.
