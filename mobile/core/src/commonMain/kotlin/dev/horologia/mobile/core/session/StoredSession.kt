package dev.horologia.mobile.core.session

import kotlinx.serialization.Serializable

/**
 * Tokens returned by `POST /oauth/token`, keyed per server host in [SessionStore].
 *
 * [accessTokenExpiresAtMillis] is the wall-clock instant at which the access token expires — the
 * caller stamps it at token-exchange time using the server's `expires_in`. Kept in UTC millis to
 * stay expect/actual-friendly (no `Instant` dependency on the commonMain surface of Kotlin 2.3.x).
 */
@Serializable
data class StoredSession(
  val accessToken: String,
  val refreshToken: String?,
  val accessTokenExpiresAtMillis: Long,
)
