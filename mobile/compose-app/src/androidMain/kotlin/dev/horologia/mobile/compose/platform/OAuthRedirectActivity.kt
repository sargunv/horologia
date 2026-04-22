package dev.horologia.mobile.compose.platform

import android.app.Activity
import android.content.Intent
import android.os.Bundle
import kotlinx.coroutines.CancellationException
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
    // Drop any intent that doesn't match the registered `horologia://oauth` deep-link.
    // Prevents a malicious sibling app from stuffing the OAuth state machine with crafted
    // intents that happen to target this Activity via a different scheme/host.
    if (data.scheme != "horologia" || data.host != "oauth") return
    OAuthResultChannel.deliver(uri = data.toString())
  }
}

/**
 * Process-wide single-shot channel the Custom Tabs launcher awaits. Access is protected by an
 * intrinsic lock so rapid arm/deliver/cancel interleavings can't drop a freshly-armed deferred or
 * deliver to a cancelled one. `arm()` cancels any prior pending deferred before replacing it.
 */
object OAuthResultChannel {
  private val lock = Any()
  private var pending: CompletableDeferred<String>? = null

  fun arm(): CompletableDeferred<String> {
    val fresh = CompletableDeferred<String>()
    synchronized(lock) {
      pending?.let { if (!it.isCompleted) it.cancel(CancellationException("rearmed")) }
      pending = fresh
    }
    return fresh
  }

  fun deliver(uri: String) {
    synchronized(lock) {
      val current = pending
      if (current != null && !current.isCompleted) {
        current.complete(uri)
      }
      pending = null
    }
  }

  fun cancel() {
    synchronized(lock) {
      pending?.let { if (!it.isCompleted) it.cancel() }
      pending = null
    }
  }
}
