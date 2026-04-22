package dev.horologia.mobile.core.platform

/**
 * Android [BrowserLauncher] — delegates the outbound Custom Tabs trip to the
 * [AndroidBrowserLauncherBridge] that `:compose-app/MainActivity` supplies at construction.
 *
 * The installer-global pattern used earlier was removed in favour of constructor injection: the
 * bridge ships with the AppContainer directly, so there's no `install()` side-channel and the
 * contract is enforced by the type system.
 */
class AndroidBrowserLauncher(private val bridge: AndroidBrowserLauncherBridge) : BrowserLauncher {
  override fun redirectUri(): String = REDIRECT_URI

  override suspend fun launchAndAwait(authorizeUrl: String): String =
    bridge.launchAndAwait(authorizeUrl = authorizeUrl, redirectUri = REDIRECT_URI)

  private companion object {
    const val REDIRECT_URI = "horologia://oauth"
  }
}
