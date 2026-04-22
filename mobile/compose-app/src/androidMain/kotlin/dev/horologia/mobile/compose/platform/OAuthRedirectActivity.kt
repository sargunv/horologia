package dev.horologia.mobile.compose.platform

import android.app.Activity
import android.content.Intent
import android.os.Bundle
import kotlinx.coroutines.CompletableDeferred

/**
 * Lightweight activity that catches the `horologia://oauth` deeplink coming back from Custom Tabs,
 * forwards the URI to [OAuthResultChannel], and finishes without popping a UI.
 *
 * Declared with `taskAffinity=""` and `launchMode="singleTask"` in the manifest so the OAuth
 * handoff doesn't get pushed onto the main task's back-stack — that avoids the "pressing back from
 * Profile after sign-in reopens Login" bug you'd otherwise see with a MainActivity-hosted
 * intent-filter.
 */
class OAuthRedirectActivity : Activity() {
  override fun onCreate(savedInstanceState: Bundle?) {
    super.onCreate(savedInstanceState)
    handleIntent(intent)
    finish()
  }

  override fun onNewIntent(intent: Intent) {
    super.onNewIntent(intent)
    handleIntent(intent)
    finish()
  }

  private fun handleIntent(intent: Intent?) {
    val data = intent?.data ?: return
    OAuthResultChannel.deliver(uri = data.toString())
  }
}

/**
 * Process-wide single-shot channel the Custom Tabs launcher awaits. Unit tests aren't relevant here
 * (Activity scope), and Custom Tabs guarantees only one flow is in-flight at a time — so a single
 * `CompletableDeferred<String>` is sufficient.
 */
object OAuthResultChannel {
  @Volatile private var pending: CompletableDeferred<String>? = null

  fun arm(): CompletableDeferred<String> {
    val fresh = CompletableDeferred<String>()
    pending = fresh
    return fresh
  }

  fun deliver(uri: String) {
    pending?.let { if (!it.isCompleted) it.complete(uri) }
  }

  fun cancel() {
    pending?.let { if (!it.isCompleted) it.cancel() }
    pending = null
  }
}
