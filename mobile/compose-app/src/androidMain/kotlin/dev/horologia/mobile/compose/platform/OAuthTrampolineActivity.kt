package dev.horologia.mobile.compose.platform

import android.app.Activity
import android.content.Context
import android.content.Intent
import android.net.Uri
import android.os.Bundle
import androidx.browser.customtabs.CustomTabsIntent

/**
 * No-UI trampoline that owns the Custom Tabs lifecycle so the app can detect "user closed the tab
 * without signing in". Custom Tabs itself reports nothing on dismissal — the standard workaround is
 * to put an invisible Activity between the app and the tab:
 *
 * 1. [onCreate] launches Custom Tabs and records a "tab is open" flag in instance state.
 * 2. Custom Tabs covers this Activity; it goes to `onPause`/`onStop`.
 * 3. Either [OAuthRedirectActivity] fires (deep-link delivered → channel completes) **or** the user
 *    closes the tab. In both cases this Activity returns to the foreground and [onResume] fires.
 * 4. If the channel is still pending at that point, the user cancelled — signal it and finish.
 *
 * `launchMode="singleTask"` + `noHistory="true"` in the manifest keeps the trampoline off the main
 * back stack so pressing back from Profile after sign-in doesn't rewind through it.
 */
class OAuthTrampolineActivity : Activity() {
  // Lifecycle on launch: onCreate → onStart → onResume → (Custom Tabs covers us) → onPause →
  // onStop. When the user returns: onRestart → onStart → onResume. We need to distinguish the
  // first resume (tab is about to cover) from the second (tab dismissed). `onPause` sets the
  // flag after the first resume, so only a post-pause resume triggers the cancel/finish path.
  private var returnedFromTab: Boolean = false

  override fun onCreate(savedInstanceState: Bundle?) {
    super.onCreate(savedInstanceState)
    returnedFromTab = savedInstanceState?.getBoolean(KEY_RETURNED_FROM_TAB, false) == true
    if (savedInstanceState != null) return
    val url = intent?.getStringExtra(EXTRA_AUTHORIZE_URL)
    if (url.isNullOrBlank()) {
      if (OAuthResultChannel.isPending()) OAuthResultChannel.cancel()
      finish()
      return
    }
    CustomTabsIntent.Builder().build().launchUrl(this, Uri.parse(url))
  }

  override fun onPause() {
    super.onPause()
    returnedFromTab = true
  }

  override fun onResume() {
    super.onResume()
    if (!returnedFromTab) return
    if (OAuthResultChannel.isPending()) {
      // No deep-link delivered — the user closed the tab.
      OAuthResultChannel.cancel()
    }
    finish()
  }

  override fun onSaveInstanceState(outState: Bundle) {
    super.onSaveInstanceState(outState)
    outState.putBoolean(KEY_RETURNED_FROM_TAB, returnedFromTab)
  }

  companion object {
    private const val EXTRA_AUTHORIZE_URL = "authorize_url"
    private const val KEY_RETURNED_FROM_TAB = "returned_from_tab"

    fun newIntent(context: Context, authorizeUrl: String): Intent =
      Intent(context, OAuthTrampolineActivity::class.java)
        .putExtra(EXTRA_AUTHORIZE_URL, authorizeUrl)
        .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
  }
}
