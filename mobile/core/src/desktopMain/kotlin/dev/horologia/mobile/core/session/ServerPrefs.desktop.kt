package dev.horologia.mobile.core.session

import java.util.prefs.Preferences

/**
 * Desktop server-URL prefs backed by `java.util.prefs.Preferences` at the user root. The concrete
 * backing store is platform-dependent (macOS plist, Windows registry, Linux .java/.userPrefs XML)
 * but we don't care — Preferences abstracts that away and it's already on the JDK, no extra dep.
 */
class DesktopServerPrefs : ServerPrefs {
  private val node: Preferences = Preferences.userRoot().node(NODE_PATH)

  override suspend fun loadServerUrl(): String? = node.get(KEY_URL, null)

  override suspend fun saveServerUrl(url: String) {
    node.put(KEY_URL, url)
    node.flush()
  }

  override suspend fun clearServerUrl() {
    node.remove(KEY_URL)
    node.flush()
  }

  private companion object {
    const val NODE_PATH = "dev/horologia/mobile/server_prefs"
    const val KEY_URL = "server_url"
  }
}
