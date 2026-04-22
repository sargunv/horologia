package dev.horologia.mobile.core.feature.login

import dev.horologia.mobile.core.session.ServerPrefs
import dev.horologia.mobile.core.session.SessionHolder
import dev.horologia.mobile.core.session.StoredSession
import io.ktor.http.Url
import kotlin.time.Clock
import kotlin.time.ExperimentalTime

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
 * Decide where to land on first-frame. One disk read + an optional single refresh-token round-trip
 * when the stored access token is expired or within [REFRESH_LEEWAY_MILLIS] of expiring. A
 * transient failure during that refresh does NOT block cold launch — we return
 * [BootDestination.SignedIn] with the near-expired token and let the first API call surface the 401
 * through the existing classifier.
 */
@OptIn(ExperimentalTime::class)
class BootRouter
internal constructor(
  private val serverPrefs: ServerPrefs,
  private val sessionHolder: SessionHolder,
  private val loginGateway: LoginGateway,
  private val nowMillis: () -> Long = { Clock.System.now().toEpochMilliseconds() },
) {
  suspend fun decideBootDestination(): BootDestination {
    val savedUrl = serverPrefs.loadServerUrl() ?: return BootDestination.Unconfigured
    val host =
      runCatching { Url(savedUrl).host }.getOrNull()?.takeIf { it.isNotEmpty() }
        ?: return BootDestination.ServerOnly(savedUrl = savedUrl)
    val stored =
      sessionHolder.load(host = host) ?: return BootDestination.ServerOnly(savedUrl = savedUrl)

    val needsRefresh = nowMillis() >= stored.accessTokenExpiresAtMillis - REFRESH_LEEWAY_MILLIS
    if (!needsRefresh) {
      return BootDestination.SignedIn(savedUrl = savedUrl, host = host)
    }

    val refresh = stored.refreshToken
    if (refresh == null) {
      sessionHolder.clear(host = host)
      return BootDestination.SignedOutAfterRefresh(savedUrl = savedUrl)
    }

    // The token endpoint is rooted at the server host (`/oauth/token`), not under `/api/`, so
    // we pass the unprefixed saved URL the same way `LoginViewModel` does during sign-in.
    val result =
      loginGateway.refreshAccessToken(
        baseUrl = savedUrl,
        refreshToken = refresh,
        clientId = REFRESH_CLIENT_ID,
      )
    return when (result) {
      is TokenResult.Ok -> {
        sessionHolder.install(
          host = host,
          session =
            StoredSession(
              accessToken = result.accessToken,
              refreshToken = result.refreshToken ?: stored.refreshToken,
              accessTokenExpiresAtMillis = result.accessTokenExpiresAtMillis,
            ),
        )
        BootDestination.SignedIn(savedUrl = savedUrl, host = host)
      }
      is TokenResult.AuthFailure,
      is TokenResult.Permanent -> {
        sessionHolder.clear(host = host)
        BootDestination.SignedOutAfterRefresh(savedUrl = savedUrl)
      }
      is TokenResult.Retryable -> BootDestination.SignedIn(savedUrl = savedUrl, host = host)
    }
  }

  internal companion object {
    internal const val REFRESH_LEEWAY_MILLIS: Long = 60_000L
    internal const val REFRESH_CLIENT_ID: String = "horologia-mobile"
  }
}
