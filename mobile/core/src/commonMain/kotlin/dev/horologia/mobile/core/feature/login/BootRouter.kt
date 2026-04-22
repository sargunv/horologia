package dev.horologia.mobile.core.feature.login

import dev.horologia.mobile.core.platform.platformLog
import dev.horologia.mobile.core.session.ServerPrefs
import dev.horologia.mobile.core.session.SessionHolder
import dev.horologia.mobile.core.session.StoredSession
import io.ktor.http.Url
import kotlin.time.Clock
import kotlin.time.ExperimentalTime
import kotlinx.coroutines.CancellationException

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
  suspend fun decideBootDestination(): BootDestination =
    try {
      decideBootDestinationInner()
    } catch (ce: CancellationException) {
      throw ce
    } catch (t: Throwable) {
      // On Kotlin/Native the SKIE-generated Swift bridge for a suspend fn turns any escaping
      // non-CancellationException into a process abort via the default uncaught-exception hook
      // (the Swift `catch` never sees it). Swallow to Unconfigured — cold launch lands on the
      // server picker where the user can retry, instead of the app crashing on open.
      platformLog(
        "BootRouter",
        "decideBootDestination failed: ${t::class.simpleName}: ${t.message}; defaulting to Unconfigured",
      )
      BootDestination.Unconfigured
    }

  private suspend fun decideBootDestinationInner(): BootDestination {
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
        // If the server rotated the refresh token, the old one is consumed server-side the
        // moment Ok arrives. A failed persist here (Keychain locked, disk full, etc.) would
        // strand the user with a valid-in-memory but unsaveable session; next cold launch
        // would retry refresh with the already-consumed old token and hard-fail. Swallow the
        // write error and fall through to optimistic SignedIn with the OLD in-memory session;
        // the first API call's 401 will re-trigger a full sign-out through the normal path.
        try {
          sessionHolder.install(
            host = host,
            session =
              StoredSession(
                accessToken = result.accessToken,
                refreshToken = result.refreshToken ?: stored.refreshToken,
                accessTokenExpiresAtMillis = result.accessTokenExpiresAtMillis,
              ),
          )
        } catch (ce: CancellationException) {
          throw ce
        } catch (t: Throwable) {
          platformLog(
            "BootRouter",
            "sessionHolder.install after refresh failed: ${t::class.simpleName}: ${t.message}",
          )
          // Persist failed; session stays in memory with pre-refresh tokens.
        }
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
