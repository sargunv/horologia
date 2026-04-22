package dev.horologia.mobile.core.platform

/**
 * iOS [BrowserLauncher] — delegates the outbound `ASWebAuthenticationSession` trip to the
 * [IosBrowserLauncherBridge] that SwiftUI supplies at construction (the concrete Swift-side
 * `OAuthLauncher`).
 *
 * The installer-global pattern used earlier was removed in favour of constructor injection: the
 * bridge ships with the AppContainer directly, so there's no `install()` side-channel and the
 * contract is enforced by the type system.
 */
class IosBrowserLauncher(private val bridge: IosBrowserLauncherBridge) : BrowserLauncher {
  override fun redirectUri(): String = REDIRECT_URI

  override suspend fun launchAndAwait(authorizeUrl: String): String =
    bridge.launchAndAwait(authorizeUrl = authorizeUrl, redirectUri = REDIRECT_URI)

  private companion object {
    const val REDIRECT_URI = "horologia://oauth"
  }
}
