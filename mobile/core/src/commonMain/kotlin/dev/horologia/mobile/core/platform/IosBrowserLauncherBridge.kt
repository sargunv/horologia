package dev.horologia.mobile.core.platform

/**
 * Bridge between the iOS [BrowserLauncher] (`IosBrowserLauncher` in iosMain) and SwiftUI's
 * `ASWebAuthenticationSession`. SwiftUI implements this interface via SKIE's Swift-side interface
 * bridging at app init, and `HorologiaApp` hands the implementation into `IosBrowserLauncher`
 * directly.
 */
interface IosBrowserLauncherBridge {
  suspend fun launchAndAwait(authorizeUrl: String, redirectUri: String): String
}
