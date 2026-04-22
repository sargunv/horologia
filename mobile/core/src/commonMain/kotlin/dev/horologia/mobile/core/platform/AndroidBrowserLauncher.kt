package dev.horologia.mobile.core.platform

/**
 * Bridge between [BrowserLauncher] actuals and the Android Activity layer.
 *
 * Custom Tabs requires an `Activity` context (Launcher must resolve a chooser,
 * OAuthRedirectActivity must call `startActivity`), which only :compose-app can supply.
 * `:core/androidMain`'s [BrowserLauncher] actual therefore forwards through to the installed
 * launcher. `MainActivity.onCreate` installs it before any screen attempts to use it.
 */
interface AndroidBrowserLauncher {
  suspend fun launchAndAwait(authorizeUrl: String, redirectUri: String): String
}
