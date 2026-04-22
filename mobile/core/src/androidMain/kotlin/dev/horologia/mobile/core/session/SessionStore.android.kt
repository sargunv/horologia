package dev.horologia.mobile.core.session

import android.content.Context
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

/**
 * EncryptedSharedPreferences-backed session store. One prefs file holds a JSON blob per host; hosts
 * are keyed directly so swapping servers doesn't evict the other's tokens.
 *
 * `androidx.security:security-crypto` 1.1.0-alpha06 is the latest published build as of 2026-04;
 * the artifact is flagged deprecated but still ships and is the pragmatic choice for this task. A
 * direct-Keystore replacement is tracked under "Deferred Items" in workpad.md.
 */
actual class SessionStore(context: Context) {
  private val prefs = run {
    val appContext = context.applicationContext
    val masterKey =
      MasterKey.Builder(appContext, MasterKey.DEFAULT_MASTER_KEY_ALIAS)
        .setKeyScheme(MasterKey.KeyScheme.AES256_GCM)
        .build()
    EncryptedSharedPreferences.create(
      appContext,
      NAME,
      masterKey,
      EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
      EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM,
    )
  }

  actual suspend fun read(host: String): StoredSession? {
    val raw = prefs.getString(host, null) ?: return null
    return try {
      json.decodeFromString<StoredSession>(raw)
    } catch (_: Throwable) {
      null
    }
  }

  actual suspend fun write(host: String, session: StoredSession) {
    prefs.edit().putString(host, json.encodeToString(session)).apply()
  }

  actual suspend fun clear(host: String) {
    prefs.edit().remove(host).apply()
  }

  private companion object {
    const val NAME = "horologia_session_store"
    val json = Json { ignoreUnknownKeys = true }
  }
}
