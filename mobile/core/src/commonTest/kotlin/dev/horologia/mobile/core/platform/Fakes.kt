package dev.horologia.mobile.core.platform

import dev.horologia.mobile.core.session.ServerPrefs
import dev.horologia.mobile.core.session.SessionStore
import dev.horologia.mobile.core.session.StoredSession

class FakeServerPrefs(var url: String? = null) : ServerPrefs {
  override suspend fun loadServerUrl(): String? = url

  override suspend fun saveServerUrl(url: String) {
    this.url = url
  }

  override suspend fun clearServerUrl() {
    url = null
  }
}

class FakeSessionStore : SessionStore {
  val entries = mutableMapOf<String, StoredSession>()

  override suspend fun read(host: String): StoredSession? = entries[host]

  override suspend fun write(host: String, session: StoredSession) {
    entries[host] = session
  }

  override suspend fun clear(host: String) {
    entries.remove(host)
  }
}

class FakeBrowserLauncher(
  val redirectUriValue: String = "horologia://oauth",
  var launch: (suspend (String) -> String) = { "$redirectUriValue?state=STATE&code=CODE" },
) : BrowserLauncher {
  var lastAuthorizeUrl: String? = null

  override fun redirectUri(): String = redirectUriValue

  override suspend fun launchAndAwait(authorizeUrl: String): String {
    lastAuthorizeUrl = authorizeUrl
    return launch(authorizeUrl)
  }
}
