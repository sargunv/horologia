package dev.horologia.mobile.core

import com.kroegerama.openapi.kmp.gen.companion.AuthItem
import dev.horologia.mobile.generated.Api
import dev.horologia.mobile.generated.Auth
import io.ktor.http.Url

/**
 * Configure the generated Horologia API singleton. **Call exactly once** at app start, before
 * constructing an `AppContainer` — this mutates process-wide state on the generated `Api` object.
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
