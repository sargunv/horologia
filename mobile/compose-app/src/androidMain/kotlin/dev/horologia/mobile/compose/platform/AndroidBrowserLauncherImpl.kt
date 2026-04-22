package dev.horologia.mobile.compose.platform

import android.content.Context
import android.net.Uri
import androidx.browser.customtabs.CustomTabsIntent
import dev.horologia.mobile.core.platform.AndroidBrowserLauncherBridge
import dev.horologia.mobile.core.platform.BrowserCancelledException
import kotlin.time.Duration.Companion.minutes
import kotlinx.coroutines.TimeoutCancellationException
import kotlinx.coroutines.withTimeout

/**
 * Concrete [AndroidBrowserLauncherBridge] — opens Custom Tabs on [authorizeUrl] and awaits a
 * URI-scoped callback from [OAuthRedirectActivity]. Requires an application-context `Context` so it
 * can issue `startActivity` with the `FLAG_ACTIVITY_NEW_TASK` flag demanded from a non-Activity
 * context.
 *
 * The wait is bounded by a 5-minute timeout so an abandoned Custom Tabs session (back-swipe, home
 * press, etc.) doesn't keep `LoginViewModel.flowJob` suspended forever. Timeout surfaces as
 * [BrowserCancelledException] with a user-facing message.
 */
class AndroidBrowserLauncherImpl(private val context: Context) : AndroidBrowserLauncherBridge {
  override suspend fun launchAndAwait(authorizeUrl: String, redirectUri: String): String {
    val pending = OAuthResultChannel.arm()
    val intent = CustomTabsIntent.Builder().build()
    intent.intent.flags = intent.intent.flags or android.content.Intent.FLAG_ACTIVITY_NEW_TASK
    intent.launchUrl(context.applicationContext, Uri.parse(authorizeUrl))
    return try {
      withTimeout(5.minutes) { pending.await() }
    } catch (_: TimeoutCancellationException) {
      OAuthResultChannel.cancel()
      throw BrowserCancelledException("Sign-in timed out. Try again.")
    } catch (t: Throwable) {
      OAuthResultChannel.cancel()
      throw t
    }
  }
}
