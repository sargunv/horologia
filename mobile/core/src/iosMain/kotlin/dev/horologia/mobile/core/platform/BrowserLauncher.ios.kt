package dev.horologia.mobile.core.platform

/**
 * iOS thin wrapper around `IosBrowserLauncher` (implemented Swift-side via
 * `ASWebAuthenticationSession`). SwiftUI installs the launcher at app init before any screen
 * attempts to use it.
 */
actual class BrowserLauncher {
  actual fun redirectUri(): String = REDIRECT_URI

  actual suspend fun launchAndAwait(authorizeUrl: String): String {
    val installed =
      requireNotNull(installedLauncher) {
        "IosBrowserLauncher has not been installed; HorologiaApp.init must call" +
          " BrowserLauncher.install() before the login flow runs."
      }
    return installed.launchAndAwait(authorizeUrl = authorizeUrl, redirectUri = REDIRECT_URI)
  }

  companion object {
    private const val REDIRECT_URI = "horologia://oauth"

    private var installedLauncher: IosBrowserLauncher? = null

    fun install(launcher: IosBrowserLauncher) {
      installedLauncher = launcher
    }
  }
}
