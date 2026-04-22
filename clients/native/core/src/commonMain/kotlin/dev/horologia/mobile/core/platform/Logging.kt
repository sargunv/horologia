package dev.horologia.mobile.core.platform

/**
 * Diagnostic log hook with per-platform routing.
 *
 * - Android: `Log.i(tag, message)` so the lines show up in `adb logcat`.
 * - iOS: `NSLog("[tag] message")` so `xcrun simctl spawn booted log stream` surfaces them (plain
 *   `println` goes to stderr which unified logging doesn't pick up).
 * - Desktop: `println("[tag] message")` to stdout.
 *
 * Use sparingly — this is for the kind of trace that has to survive into a device bug report, not
 * for general chatter.
 */
expect fun platformLog(tag: String, message: String)
