package dev.horologia.mobile.core.platform

/**
 * Bridge between the iOS [BrowserLauncher] (`IosBrowserLauncher` in iosMain) and SwiftUI's
 * `ASWebAuthenticationSession`. SwiftUI implements this interface at app init, and `HorologiaApp`
 * hands the implementation into `IosBrowserLauncher` directly.
 *
 * Swift can throw `NSError` from the implementation:
 * - domain = [ErrorDomain], code = [CancelledCode] → [BrowserCancelledException]
 * - domain = [ErrorDomain], code = [FailedCode] → [BrowserFailedException] (the error's
 *   `localizedDescription` becomes the exception message)
 *
 * Any other thrown value is wrapped in [BrowserFailedException] with a generic message.
 */
interface IosBrowserLauncherBridge {
  suspend fun launchAndAwait(authorizeUrl: String, redirectUri: String): String

  companion object {
    const val ErrorDomain: String = "dev.horologia.mobile.browser"
    const val CancelledCode: Int = 1
    const val FailedCode: Int = 2
  }
}
