package dev.horologia.mobile.core.platform

/**
 * Platform browser handoff for the OAuth authorize → callback round-trip.
 *
 * Implementations:
 * - iOS: `ASWebAuthenticationSession` (Swift-side; Kotlin delegates via
 *   `IosBrowserLauncher` + [IosBrowserLauncherBridge]).
 * - Android: Custom Tabs + `OAuthRedirectActivity` (Kotlin delegates via
 *   `AndroidBrowserLauncher` + [AndroidBrowserLauncherBridge]).
 * - Desktop: `com.sun.net.httpserver.HttpServer` on a random loopback port +
 *   `java.awt.Desktop.browse()` (see `DesktopBrowserLauncher`).
 *
 * [redirectUri] is stable per-platform except on desktop, where the port is known only after the
 * loopback listener binds; the desktop actual binds lazily on first call so that `redirectUri()`
 * can always return a populated URI.
 *
 * [launchAndAwait] opens the browser on [authorizeUrl] and suspends until the browser returns to
 * [redirectUri], at which point it resolves to the raw callback URL string (so the caller can parse
 * `code` / `state` / `error`). Throws [BrowserCancelledException] if the user aborts the flow.
 */
interface BrowserLauncher {
  fun redirectUri(): String

  suspend fun launchAndAwait(authorizeUrl: String): String
}

/**
 * Raised when the user cancels the OAuth flow in the system browser, or when the flow times out
 * without a callback (e.g. user abandoned Custom Tabs, closed the desktop browser tab).
 * LoginViewModel maps this to a `Sign-in cancelled.` banner.
 */
class BrowserCancelledException(message: String = "OAuth sign-in cancelled by user") :
  RuntimeException(message)

/**
 * Raised when the platform browser launcher can't actually run the flow (e.g. desktop loopback port
 * collision, missing Custom Tabs handler). LoginViewModel surfaces the [message] in the banner.
 */
class BrowserFailedException(message: String) : RuntimeException(message)
