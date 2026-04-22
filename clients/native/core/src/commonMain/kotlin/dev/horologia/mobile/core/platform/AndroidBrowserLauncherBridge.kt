package dev.horologia.mobile.core.platform

/**
 * Bridge between the Android [BrowserLauncher] (`AndroidBrowserLauncher` in androidMain) and the
 * Activity-bound Custom Tabs implementation that lives in `:compose-app/androidMain`.
 *
 * Custom Tabs requires an `Activity` context (Launcher must resolve a chooser,
 * OAuthRedirectActivity must call `startActivity`), which only `:compose-app` can supply. `:core`
 * stays free of direct `androidx.browser` / `android.app.Activity` dependencies; MainActivity
 * constructs its bridge implementation and hands it in to `AndroidBrowserLauncher`.
 */
interface AndroidBrowserLauncherBridge {
  suspend fun launchAndAwait(authorizeUrl: String, redirectUri: String): String
}
