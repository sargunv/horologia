---
name: preview-android
description: >-
  Use when the user asks you to run, preview, install, screenshot, or manually verify the
  Android app on an emulator (or connected device). Covers the full CLI loop via `emulator` +
  `adb`: boot, install, launch, screenshot, tail logs. Prefers CLI over Android Studio.
---

# Preview / verify the Android app on an emulator

CLI-only workflow for driving the Android emulator end-to-end. Use this whenever you need visual
proof (a screenshot) or runtime evidence (logcat) that an Android change works.

## Prerequisites

- `mise run //clients/native:setup` has been run at least once. That installs the SDK, the emulator
  binary, the system image, and creates the `horologia` AVD.
- The mise `android-sdk` plugin exposes only `cmdline-tools` on `PATH`, so `emulator` and `adb` are
  NOT on `PATH`. Use full paths via `$ANDROID_HOME`:

  ```bash
  EMULATOR="$ANDROID_HOME/emulator/emulator"
  ADB="$ANDROID_HOME/platform-tools/adb"
  ```

  (Run commands under `mise x --` so `ANDROID_HOME` resolves.)

- The dev backend is reachable. The emulator can't reach the host's `localhost` directly — use the
  special host `10.0.2.2:<port>` instead, or set up `adb reverse` (below) and use
  `localhost:<port>`.

## The loop

1. **Boot the emulator.** Run in the background; foreground locks the shell.
   `-no-audio -no-boot-anim` makes it quieter:

   ```bash
   mise x -- bash -c '"$ANDROID_HOME/emulator/emulator" \
     -avd horologia -no-audio -no-boot-anim \
     -netdelay none -netspeed full > /tmp/emulator.log 2>&1' &
   ```

   Wait for ADB to see it, then wait for `sys.boot_completed`:

   ```bash
   "$ADB" wait-for-device
   for i in $(seq 1 60); do
     BOOT=$("$ADB" shell getprop sys.boot_completed 2>/dev/null | tr -d '\r')
     [ "$BOOT" = "1" ] && break
     sleep 2
   done
   "$ADB" devices   # confirm `emulator-5554 device`
   ```

2. **Build and install.**

   ```bash
   mise run //clients/native:android:installDebug
   ```

   Or step-by-step if you need the intermediate APK path:

   ```bash
   cd mobile
   mise x -- ./gradlew :compose-app:assembleDebug
   "$ADB" install -r compose-app/build/outputs/apk/debug/compose-app-debug.apk
   ```

3. **Launch.** `am start` prints the intent it resolved:

   ```bash
   "$ADB" shell am force-stop dev.horologia.mobile.compose
   "$ADB" shell am start -n dev.horologia.mobile.compose/.MainActivity
   ```

   Or the combined task:

   ```bash
   mise run //clients/native:android:run
   ```

4. **Screenshot.** `exec-out screencap -p` pipes a PNG directly to stdout — no on-device temp file
   to clean up:

   ```bash
   sleep 3
   "$ADB" exec-out screencap -p > /tmp/android.png
   ```

   Then `Read` the PNG to inspect.

## Drive interaction

`adb shell input` and `uiautomator` cover scripted taps, typing, and basic assertions from the CLI.
Good for smoke tests and for exercising flows that `screencap` alone can't reach.

- **Dump the view hierarchy** to find a target element's on-screen bounds or resource id:

  ```bash
  "$ADB" shell uiautomator dump /sdcard/window_dump.xml
  "$ADB" pull /sdcard/window_dump.xml /tmp/window_dump.xml
  ```

  The XML contains `bounds="[x1,y1][x2,y2]"` for each node. Tap the center of the bounds.

- **Tap** at screen coordinates (after uiautomator tells you where):

  ```bash
  "$ADB" shell input tap <x> <y>
  ```

- **Type text** (the focused field receives it):

  ```bash
  "$ADB" shell input text "hello%sworld"   # %s is the escape for space
  ```

- **Key events** — Enter, Back, Home, volume, etc.:

  ```bash
  "$ADB" shell input keyevent KEYCODE_ENTER
  "$ADB" shell input keyevent KEYCODE_BACK
  ```

  Full list: `adb shell input keyevent --longpress ...` docs via `adb shell input` with no args.

- **Swipe / drag** over N milliseconds:

  ```bash
  "$ADB" shell input swipe <x1> <y1> <x2> <y2> 300
  ```

- **Clipboard-based paste** (when `input text` mis-escapes or is too slow for long strings):

  ```bash
  "$ADB" shell service call clipboard 2 i32 1 i32 1 i32 1 s16 "long string" \
    i32 1 i32 0 i32 0 i32 0
  "$ADB" shell input keyevent KEYCODE_PASTE
  ```

  (Ugly; reach for it only when `input text` fails.)

- **Deep link** — fastest way to land on a specific screen with params:

  ```bash
  "$ADB" shell am start -W -a android.intent.action.VIEW -d "https://example.com/path"
  ```

A typical scripted flow is: dump → tap → screenshot → assert pixel or dump again. For anything
richer (multi-step flows with retries, flakiness tolerance), reach for Espresso or Maestro and flag
the need.

## Common variations

- **Tail the app's logcat** (essential when the UI shows a generic error state and you need the real
  stack trace):

  ```bash
  PID=$("$ADB" shell pidof dev.horologia.mobile.compose | tr -d '\r')
  "$ADB" logcat -d --pid=$PID | tail -80
  ```

  Or stream live:

  ```bash
  "$ADB" logcat --pid=$PID '*:W'
  ```

- **Emulator → host reachability test** (when "Network error" shows up but the host server is
  actually reachable):

  ```bash
  "$ADB" shell ping -c 1 -W 2 10.0.2.2
  ```

  If that fails, the emulator network is broken — `kill` the emulator and rerun
  `//clients/native:setup`.

- **Use `adb reverse` to expose host ports as emulator-localhost** (an alternative to `10.0.2.2`
  when you can't change the baked URL):

  ```bash
  "$ADB" reverse tcp:8080 tcp:8080
  "$ADB" reverse --list
  ```

  The app can then hit `http://localhost:8080/` on the device and traffic is forwarded to the host.

- **Inspect baked BuildConfig values** (catches env-var plumbing bugs):

  ```bash
  mkdir -p /tmp/apk && cd /tmp/apk
  unzip -o -q <path>/compose-app-debug.apk
  /usr/bin/strings -a *.dex | grep -E 'HOROLOGIA|10\.0\.2\.2|localhost'
  ```

- **Shut down the emulator** cleanly when done:

  ```bash
  "$ADB" emu kill
  ```

## Tips for reliable automation

- Cleartext HTTP to arbitrary hosts is blocked on API 28+. The project's
  `network_security_config.xml` permits `10.0.2.2` and `localhost` only; if you need a different dev
  host, add it there (and file a follow-up to drop the exception once TLS lands).
- `INTERNET` permission must be in `AndroidManifest.xml` — without it HTTP clients surface opaque
  connection failures with no useful message. If a generic "Network error" appears and the server is
  reachable from the host, check the manifest first.
- `am start` before `force-stop` will merge into an existing task; if the app looks "stuck" on old
  state, `force-stop` first.
- The `horologia` AVD uses `arm64-v8a` on Apple Silicon hosts for hardware acceleration. On x86_64
  hosts, `//clients/native:setup` still installs arm64 — override by passing `--device` and the
  matching `system-images;...;x86_64` to `avdmanager create avd` manually.

## When this isn't enough

- Multi-step flows with retries or flakiness tolerance: reach for Espresso or Maestro (see the
  "Drive interaction" section) and flag the need.
- Installing on a physical device: plug it in with USB debugging enabled, `"$ADB" devices` should
  show it, and the same `install`/`launch`/`screencap` commands work. Mention to the user that a
  physical device is being targeted.
