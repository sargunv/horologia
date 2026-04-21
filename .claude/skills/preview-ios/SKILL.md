---
name: preview-ios
description: Use when the user asks you to run, preview, install, screenshot, or manually verify the iOS app on a simulator. Covers the full CLI loop: boot simulator, build, install, launch, capture screenshot, inspect logs. Not a substitute for Xcode — prefers `xcrun simctl` + `xcodebuild` over the GUI.
---

# Preview / verify the iOS app on a simulator

CLI-only workflow for driving the iOS simulator end-to-end. Use this whenever you need visual proof
(a screenshot) or runtime evidence (logs) that an iOS change works.

## Prerequisites

- Xcode installed (`xcrun simctl list` works).
- `mise run //mobile:ios:gen` has been run at least once — the `.xcodeproj` is generated from
  `mobile/iosApp/project.yml` and gitignored.
- The dev backend is reachable at the URL baked into `Config.xcconfig` (or inject a different
  `HOROLOGIA_BASE_URL` via the xcodebuild env if the loop you care about requires a non-default
  host).

## The loop

1. **Pick a simulator** and remember the UDID. Prefer the newest stable iPhone available:

   ```bash
   xcrun simctl list devices available | grep -E 'iPhone 1[0-9]|iPhone 2[0-9]' | head -3
   ```

   Grab the UDID in parens. Export it to shorten subsequent commands:

   ```bash
   export DEVICE_UDID=<udid>
   ```

2. **Boot it** (idempotent — errors if already booted, safe to ignore). Wait until the OS reports
   `Finished` before installing:

   ```bash
   xcrun simctl boot "$DEVICE_UDID" 2>/dev/null || true
   xcrun simctl bootstatus "$DEVICE_UDID" -b
   ```

3. **Build for this specific simulator**. Pinning `-destination id=...` lands the `.app` in a
   predictable spot so you don't have to hunt through DerivedData:

   ```bash
   cd mobile/iosApp
   xcodebuild \
     -project iosApp.xcodeproj \
     -scheme iosApp \
     -sdk iphonesimulator \
     -destination "id=$DEVICE_UDID" \
     -configuration Debug \
     CODE_SIGNING_ALLOWED=NO \
     -derivedDataPath build/xcode \
     build
   ```

   The artifact lands at
   `mobile/iosApp/build/xcode/Build/Products/Debug-iphonesimulator/iosApp.app`.

4. **Install + launch**. `install` is idempotent (overwrites). `launch` prints the new PID:

   ```bash
   APP=mobile/iosApp/build/xcode/Build/Products/Debug-iphonesimulator/iosApp.app
   xcrun simctl install "$DEVICE_UDID" "$APP"
   xcrun simctl launch "$DEVICE_UDID" dev.horologia.mobile.iosApp
   ```

   (The bundle ID is set in `project.yml`. If that changes, update here.)

5. **Screenshot** once the UI has settled. A second or two of `sleep` is usually enough; for
   genuinely slow screens, poll logs instead of sleeping longer:

   ```bash
   sleep 2
   xcrun simctl io "$DEVICE_UDID" screenshot /tmp/ios.png
   ```

   Then `Read` the PNG to inspect.

## Common variations

- **Terminate before relaunch** (avoids stale state leaks between runs):
  ```bash
  xcrun simctl terminate "$DEVICE_UDID" dev.horologia.mobile.iosApp
  ```

- **Tail app logs** (useful when the UI shows a generic error state and you need the real
  exception):
  ```bash
  xcrun simctl spawn "$DEVICE_UDID" log stream \
    --predicate 'process == "iosApp"' --level debug
  ```
  Run in a background shell and grep for your bundle ID or for known error keywords. Kill with
  SIGTERM when done.

- **Inspect what got baked into `Info.plist`** (catches xcconfig escape bugs that silently corrupt
  build-setting substitutions):
  ```bash
  /usr/libexec/PlistBuddy -c 'Print :HorologiaBaseUrl' "$APP/Info.plist"
  ```

- **Pass a runtime override for a one-off test** without editing `Config.xcconfig`:
  ```bash
  HOROLOGIA_DEV_TOKEN=... HOROLOGIA_BASE_URL=... \
    xcodebuild ... build
  ```
  The values flow through `project.yml`'s `info.properties` into the generated `Info.plist`.

- **Open the project in Xcode** instead (for the GUI debugger / SwiftUI previews):
  ```bash
  mise run //mobile:ios:open
  ```

## Tips for reliable automation

- `xcrun simctl launch --console-pty` keeps the app tied to the shell; if the shell exits, the app
  is killed. Use plain `launch` when you plan to keep the app running while you screenshot.
- After an `info.plist` or code change, do **not** skip rebuild — iOS simulator caches aggressively.
  `xcrun simctl terminate` before `install` is cheap insurance.
- `CODE_SIGNING_ALLOWED=NO` is fine for simulators; physical devices need a real signing identity.
- If `xcodebuild` fails with a PhaseScriptExecution error about "Unable to locate a Java Runtime",
  the Run Script build phase couldn't find mise. Check `project.yml`'s pre-build script for the
  mise-discovery block.

## When this isn't enough

- You need to drive UI input (tap/type): `xcrun simctl` doesn't expose that. Reach for Xcode's UI
  testing or a dedicated tool (Appium, Maestro). Flag the need and ask the user instead of
  reinventing it.
- You need a physical device: `xcodebuild -destination 'platform=iOS,id=<udid>'` with code signing,
  which is out of scope for this skill.
