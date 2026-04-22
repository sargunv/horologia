package dev.horologia.mobile.core.platform

/**
 * Android thin wrapper around `AndroidBrowserLauncher` (which lives in `:compose-app/androidMain`
 * because Custom Tabs needs an Activity). The installer pattern keeps `:core` free of direct
 * `androidx.browser` or `android.app.Activity` dependencies.
 */
actual class BrowserLauncher {
  actual fun redirectUri(): String = REDIRECT_URI

  actual suspend fun launchAndAwait(authorizeUrl: String): String {
    val installed =
      requireNotNull(installedLauncher) {
        "AndroidBrowserLauncher has not been installed; :compose-app/MainActivity must call" +
          " BrowserLauncher.install() before the login flow runs."
      }
    return installed.launchAndAwait(authorizeUrl = authorizeUrl, redirectUri = REDIRECT_URI)
  }

  companion object {
    private const val REDIRECT_URI = "horologia://oauth"

    @Volatile private var installedLauncher: AndroidBrowserLauncher? = null

    fun install(launcher: AndroidBrowserLauncher) {
      installedLauncher = launcher
    }
  }
}
