package dev.horologia.mobile.core.platform

/**
 * Bridge between [BrowserLauncher] actuals and SwiftUI's `ASWebAuthenticationSession`.
 *
 * SwiftUI implements this interface via SKIE's Swift-side interface bridging and registers it at
 * app init before any screen attempts to launch OAuth.
 */
interface IosBrowserLauncher {
  suspend fun launchAndAwait(authorizeUrl: String, redirectUri: String): String
}
