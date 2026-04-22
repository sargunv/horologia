package dev.horologia.mobile.core

import com.kroegerama.openapi.kmp.gen.companion.AuthItem
import dev.horologia.mobile.generated.Api
import dev.horologia.mobile.generated.Auth
import io.ktor.http.Url

/**
 * Configure the generated Horologia API singleton. Safe to call multiple times: Ktor 3.4.2's
 * `DefaultRequest.Plugin.install` rebuilds the request builder per outbound call, so mutating
 * `Api.baseUrl` and re-installing the auth provider takes effect on the next request. The mobile
 * login flow uses that property — ServerPicker calls `configureHorologiaApi` whenever the user
 * picks a server, and cold-launch routing calls it again during bootstrap.
 *
 * [getToken] is evaluated on every outbound request, so installing it once against a
 * [dev.horologia.mobile.core.session.SessionHolder] is sufficient — the holder reflects the current
 * session, and a sign-out that clears it automatically drops bearers from subsequent requests.
 *
 * @throws IllegalArgumentException if [baseUrl] is not a valid URL.
 */
fun configureHorologiaApi(baseUrl: String, getToken: () -> String?) {
  Api.baseUrl =
    try {
      Url(baseUrl)
    } catch (e: IllegalArgumentException) {
      throw IllegalArgumentException("Invalid Horologia base URL: '$baseUrl'", e)
    }
  Api.setAuthProvider(Auth.BearerAuth { getToken()?.let { token -> AuthItem.Bearer(token) } })
}

/**
 * Maps a Horologia server root URL to the Ktor client base URL it needs (host -> host/api/). Called
 * by platform entry points (Android/desktop/iOS) and `LoginViewModel.reconfigureApi`, which means
 * the invariant lives in exactly one place. If [baseUrl] already ends in `/api` or contains
 * `/api/`, only a trailing slash is appended.
 */
fun ensureApiPath(baseUrl: String): String {
  val trimmed = baseUrl.trimEnd('/')
  return if (trimmed.endsWith("/api") || trimmed.contains("/api/")) "$trimmed/" else "$trimmed/api/"
}
