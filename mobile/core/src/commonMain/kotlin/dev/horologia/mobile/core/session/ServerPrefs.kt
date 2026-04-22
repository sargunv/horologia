package dev.horologia.mobile.core.session

/**
 * Non-sensitive cross-launch prefs. Currently just the most-recently-chosen server URL, so a
 * returning user lands on ServerPicker with the URL pre-filled.
 *
 * Implemented with platform-native prefs (DataStore / NSUserDefaults /
 * `java.util.prefs.Preferences`) — no encryption required because the URL is public-knowledge. PKCE
 * verifiers and OAuth `state` never land here.
 */
expect class ServerPrefs {
  suspend fun loadServerUrl(): String?

  suspend fun saveServerUrl(url: String)

  suspend fun clearServerUrl()
}
