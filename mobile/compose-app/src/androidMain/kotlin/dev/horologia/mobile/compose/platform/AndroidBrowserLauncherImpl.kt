package dev.horologia.mobile.compose.platform

import android.content.Context
import android.net.Uri
import androidx.browser.customtabs.CustomTabsIntent
import dev.horologia.mobile.core.platform.AndroidBrowserLauncher

/**
 * Concrete `AndroidBrowserLauncher` — opens Custom Tabs on [authorizeUrl] and awaits a URI-scoped
 * callback from [OAuthRedirectActivity]. Requires an application-context `Context` so it can issue
 * `startActivity` with the `FLAG_ACTIVITY_NEW_TASK` flag demanded from a non-Activity context.
 */
class AndroidBrowserLauncherImpl(private val context: Context) : AndroidBrowserLauncher {
  override suspend fun launchAndAwait(authorizeUrl: String, redirectUri: String): String {
    val pending = OAuthResultChannel.arm()
    val intent = CustomTabsIntent.Builder().build()
    intent.intent.flags = intent.intent.flags or android.content.Intent.FLAG_ACTIVITY_NEW_TASK
    intent.launchUrl(context.applicationContext, Uri.parse(authorizeUrl))
    return try {
      pending.await()
    } catch (t: Throwable) {
      OAuthResultChannel.cancel()
      throw t
    }
  }
}
