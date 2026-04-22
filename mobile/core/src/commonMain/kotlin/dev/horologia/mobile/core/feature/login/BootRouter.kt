package dev.horologia.mobile.core.feature.login

import dev.horologia.mobile.core.session.ServerPrefs
import dev.horologia.mobile.core.session.ServerPrefsAdapter
import dev.horologia.mobile.core.session.ServerPrefsReader
import dev.horologia.mobile.core.session.SessionHolder
import io.ktor.http.Url

/**
 * Where the app should land on cold launch, per design spec § H.
 *
 * - [Unconfigured] — no saved server, no tokens. Render empty ServerPicker.
 * - [ServerOnly] — server saved but no valid session; ServerPicker with URL pre-filled so the user
 *   can try again.
 * - [SignedIn] — hydrate tokens into [SessionHolder] and jump straight to Profile; the background
 *   silent-refresh is handled by the caller (see R14 optimistic render).
 * - [SignedOutAfterRefresh] — tokens existed but refresh failed. ServerPicker with the URL
 *   pre-filled and a "Signed out." banner.
 */
sealed interface BootDestination {
  data object Unconfigured : BootDestination

  data class ServerOnly(val savedUrl: String) : BootDestination

  data class SignedIn(val savedUrl: String, val host: String) : BootDestination

  data class SignedOutAfterRefresh(val savedUrl: String) : BootDestination
}

/**
 * Decide where to land on first-frame. Synchronous-fast: the only I/O is the two local-disk reads
 * (server prefs + keychain). Silent refresh is explicitly deferred to the caller — the architect
 * phase chose "optimistic render + background refresh" so the first-frame time never waits on
 * network.
 */
class BootRouter
internal constructor(
  private val serverPrefs: ServerPrefsReader,
  private val sessionHolder: SessionHolder,
) {
  constructor(
    serverPrefs: ServerPrefs,
    sessionHolder: SessionHolder,
  ) : this(serverPrefs = ServerPrefsAdapter(prefs = serverPrefs), sessionHolder = sessionHolder)

  suspend fun decideBootDestination(): BootDestination {
    val savedUrl = serverPrefs.loadServerUrl() ?: return BootDestination.Unconfigured
    val host =
      runCatching { Url(savedUrl).host }.getOrNull()?.takeIf { it.isNotEmpty() }
        ?: return BootDestination.ServerOnly(savedUrl = savedUrl)
    sessionHolder.load(host = host) ?: return BootDestination.ServerOnly(savedUrl = savedUrl)
    return BootDestination.SignedIn(savedUrl = savedUrl, host = host)
  }
}
