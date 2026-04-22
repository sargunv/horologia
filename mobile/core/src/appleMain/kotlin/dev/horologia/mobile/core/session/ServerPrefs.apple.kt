package dev.horologia.mobile.core.session

import platform.Foundation.NSUserDefaults

class IosServerPrefs : ServerPrefs {
  private val defaults: NSUserDefaults = NSUserDefaults.standardUserDefaults

  override suspend fun loadServerUrl(): String? = defaults.stringForKey(KEY_URL)

  override suspend fun saveServerUrl(url: String) {
    defaults.setObject(value = url, forKey = KEY_URL)
  }

  override suspend fun clearServerUrl() {
    defaults.removeObjectForKey(KEY_URL)
  }

  private companion object {
    const val KEY_URL = "dev.horologia.mobile.server_url"
  }
}
