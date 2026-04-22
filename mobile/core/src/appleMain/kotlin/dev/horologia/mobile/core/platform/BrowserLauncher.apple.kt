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
@OptIn(kotlinx.cinterop.ExperimentalForeignApi::class)
class IosBrowserLauncher(private val bridge: IosBrowserLauncherBridge) : BrowserLauncher {
  override fun redirectUri(): String = REDIRECT_URI

  override suspend fun launchAndAwait(authorizeUrl: String): String =
    try {
      bridge.launchAndAwait(authorizeUrl = authorizeUrl, redirectUri = REDIRECT_URI)
    } catch (ce: kotlinx.coroutines.CancellationException) {
      throw ce
    } catch (t: Throwable) {
      // Swift-thrown NSError comes over as a Kotlin Throwable whose underlying NSError is
      // reachable via NSErrorKt.nsError(). Check domain+code to decide which typed exception
      // to re-throw; fall through to a generic BrowserFailedException for anything else.
      val ns = t.extractNSError()
      if (ns != null && ns.domain == IosBrowserLauncherBridge.ErrorDomain) {
        when (ns.code.toInt()) {
          IosBrowserLauncherBridge.CancelledCode -> throw BrowserCancelledException()
          IosBrowserLauncherBridge.FailedCode ->
            throw BrowserFailedException(message = ns.localizedDescription)
        }
      }
      throw BrowserFailedException(message = t.message ?: "Sign-in failed.")
    }

  private fun Throwable.extractNSError(): platform.Foundation.NSError? =
    // Kotlin/Native exposes Swift `Error` types as instances of an internal subclass whose
    // .cause isn't a Kotlin Throwable — the actual NSError lives as a platform type. Safe-cast
    // via the bridge helper; if it isn't an NSError we just return null and fall through.
    (this as? platform.Foundation.NSError) ?: (this.cause as? platform.Foundation.NSError)

  private companion object {
    const val REDIRECT_URI = "horologia://oauth"
  }
}
