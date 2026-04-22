package dev.horologia.mobile.core.session

import android.content.Context

/**
 * Android server-URL prefs backed by a plain SharedPreferences file. The URL is not sensitive, so
 * encryption is skipped; it lives at `/data/data/<pkg>/shared_prefs/horologia_server_prefs.xml`.
 */
actual class ServerPrefs(context: Context) {
  private val prefs = context.applicationContext.getSharedPreferences(NAME, Context.MODE_PRIVATE)

  actual suspend fun loadServerUrl(): String? = prefs.getString(KEY_URL, null)

  actual suspend fun saveServerUrl(url: String) {
    prefs.edit().putString(KEY_URL, url).apply()
  }

  actual suspend fun clearServerUrl() {
    prefs.edit().remove(KEY_URL).apply()
  }

  private companion object {
    const val NAME = "horologia_server_prefs"
    const val KEY_URL = "server_url"
  }
}
