package dev.horologia.mobile.core.session

import android.content.Context

/**
 * Android server-URL prefs backed by a plain SharedPreferences file. The URL is not sensitive, so
 * encryption is skipped; it lives at `/data/data/<pkg>/shared_prefs/horologia_server_prefs.xml`.
 */
class AndroidServerPrefs(context: Context) : ServerPrefs {
  private val prefs = context.applicationContext.getSharedPreferences(NAME, Context.MODE_PRIVATE)

  override suspend fun loadServerUrl(): String? = prefs.getString(KEY_URL, null)

  override suspend fun saveServerUrl(url: String) {
    prefs.edit().putString(KEY_URL, url).apply()
  }

  override suspend fun clearServerUrl() {
    prefs.edit().remove(KEY_URL).apply()
  }

  private companion object {
    const val NAME = "horologia_server_prefs"
    const val KEY_URL = "server_url"
  }
}
