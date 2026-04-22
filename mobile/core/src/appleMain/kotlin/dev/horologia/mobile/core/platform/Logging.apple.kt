package dev.horologia.mobile.core.platform

import platform.Foundation.NSLog

/**
 * iOS-side `actual` for [platformLog] — routes Kotlin logs to `NSLog` so the unified-log stream
 * (`xcrun simctl spawn booted log stream --predicate 'processImagePath CONTAINS "iosApp"'`)
 * surfaces them. Plain `println` on Kotlin/Native goes to stderr, which the Xcode simulator console
 * shows but `log stream` does not.
 *
 * Two traps to avoid:
 *
 * 1. **Format-string injection.** `NSLog` is printf-style; any `%` in the message would be
 *    interpreted as a format directive. Escape by doubling — `%` → `%%`.
 * 2. **Varargs ↔ Kotlin String.** Passing a Kotlin `String` as a `%@` arg to NSLog crashes with
 *    `EXC_BAD_ACCESS` in `objc_opt_respondsToSelector`: Kotlin/Native strings are not Obj-C
 *    objects, and NSLog's variadic bridge doesn't auto-box them. The safe path is to build the
 *    final line in Kotlin, escape `%`, then hand NSLog a single-arg format string.
 */
actual fun platformLog(tag: String, message: String) {
  val line = "[$tag] $message".replace("%", "%%")
  NSLog(line)
}
