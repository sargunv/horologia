package dev.horologia.mobile.core.feature.login

import io.ktor.client.HttpClient
import io.ktor.client.call.body
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.client.request.forms.submitForm
import io.ktor.client.request.get
import io.ktor.client.statement.HttpResponse
import io.ktor.http.HttpStatusCode
import io.ktor.http.Parameters
import io.ktor.http.URLBuilder
import io.ktor.http.Url
import io.ktor.serialization.JsonConvertException
import io.ktor.serialization.kotlinx.json.json
import kotlin.time.Clock
import kotlin.time.Duration
import kotlin.time.Duration.Companion.seconds
import kotlin.time.ExperimentalTime
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.TimeoutCancellationException
import kotlinx.coroutines.withTimeout
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json

/**
 * Everything the login flow needs to hit across the wire. Three methods, one short-lived Ktor
 * client constructed per call (we never leak a long-lived client because the target server URL
 * changes during the ServerPicker flow).
 */
internal interface LoginGateway {
  suspend fun probeServer(baseUrl: String): ProbeResult

  suspend fun exchangeCode(
    baseUrl: String,
    code: String,
    codeVerifier: String,
    redirectUri: String,
    clientId: String,
  ): TokenResult

  suspend fun refreshAccessToken(
    baseUrl: String,
    refreshToken: String,
    clientId: String,
  ): TokenResult
}

internal sealed interface ProbeResult {
  data object Ok : ProbeResult

  data object WrongServer : ProbeResult

  data class Unreachable(val host: String) : ProbeResult
}

internal sealed interface TokenResult {
  data class Ok(
    val accessToken: String,
    val refreshToken: String?,
    val accessTokenExpiresAtMillis: Long,
  ) : TokenResult

  data object AuthFailure : TokenResult

  data class Retryable(val message: String) : TokenResult

  data class Permanent(val message: String) : TokenResult
}

/** Shape expected from `GET /app/auth/config`; mirrors the generated `AuthConfig`. */
@Serializable
private data class AuthConfigShape(
  @SerialName("oidc") val oidc: OidcSection,
  @SerialName("password") val password: PasswordSection,
)

@Serializable
private data class OidcSection(
  @SerialName("enabled") val enabled: Boolean,
  @SerialName("label") val label: String,
  @SerialName("autoRedirect") val autoRedirect: Boolean,
)

@Serializable private data class PasswordSection(@SerialName("enabled") val enabled: Boolean)

@Serializable
private data class TokenResponseShape(
  @SerialName("access_token") val accessToken: String,
  @SerialName("refresh_token") val refreshToken: String? = null,
  @SerialName("expires_in") val expiresIn: Long? = null,
  @SerialName("token_type") val tokenType: String? = null,
)

@OptIn(ExperimentalTime::class)
internal class LiveLoginGateway(
  private val httpClientFactory: () -> HttpClient = ::defaultHttpClient,
  private val nowMillis: () -> Long = { Clock.System.now().toEpochMilliseconds() },
  private val probeTimeout: Duration = 4.seconds,
  private val tokenTimeout: Duration = 15.seconds,
) : LoginGateway {
  override suspend fun probeServer(baseUrl: String): ProbeResult {
    val host = runCatching { Url(baseUrl).host }.getOrNull() ?: baseUrl
    val client = httpClientFactory()
    return try {
      withTimeout(probeTimeout) {
        val probeUrl = appendPath(baseUrl = baseUrl, path = "/app/auth/config")
        val response: HttpResponse = client.get(probeUrl)
        if (response.status != HttpStatusCode.OK) {
          return@withTimeout ProbeResult.WrongServer
        }
        try {
          response.body<AuthConfigShape>()
          ProbeResult.Ok
        } catch (_: JsonConvertException) {
          ProbeResult.WrongServer
        }
      }
    } catch (_: TimeoutCancellationException) {
      ProbeResult.Unreachable(host = host)
    } catch (e: CancellationException) {
      throw e
    } catch (_: Throwable) {
      ProbeResult.Unreachable(host = host)
    } finally {
      client.close()
    }
  }

  override suspend fun exchangeCode(
    baseUrl: String,
    code: String,
    codeVerifier: String,
    redirectUri: String,
    clientId: String,
  ): TokenResult =
    postToken(
      baseUrl = baseUrl,
      formParams =
        Parameters.build {
          append("grant_type", "authorization_code")
          append("code", code)
          append("code_verifier", codeVerifier)
          append("redirect_uri", redirectUri)
          append("client_id", clientId)
        },
    )

  override suspend fun refreshAccessToken(
    baseUrl: String,
    refreshToken: String,
    clientId: String,
  ): TokenResult =
    postToken(
      baseUrl = baseUrl,
      formParams =
        Parameters.build {
          append("grant_type", "refresh_token")
          append("refresh_token", refreshToken)
          append("client_id", clientId)
        },
    )

  private suspend fun postToken(baseUrl: String, formParams: Parameters): TokenResult {
    val client = httpClientFactory()
    return try {
      withTimeout(tokenTimeout) {
        val tokenUrl = appendPath(baseUrl = baseUrl, path = "/oauth/token")
        val response: HttpResponse = client.submitForm(url = tokenUrl, formParameters = formParams)
        when {
          response.status == HttpStatusCode.OK -> {
            val shape: TokenResponseShape = response.body()
            val expiresAt =
              shape.expiresIn?.let { nowMillis() + it * 1000L } ?: (nowMillis() + 3600L * 1000L)
            TokenResult.Ok(
              accessToken = shape.accessToken,
              refreshToken = shape.refreshToken,
              accessTokenExpiresAtMillis = expiresAt,
            )
          }
          response.status == HttpStatusCode.Unauthorized ||
            response.status == HttpStatusCode.Forbidden -> TokenResult.AuthFailure
          response.status.value in 400..499 ->
            TokenResult.Permanent("Sign-in failed (${response.status.value}).")
          else -> TokenResult.Retryable("Server error (${response.status.value}).")
        }
      }
    } catch (_: TimeoutCancellationException) {
      TokenResult.Retryable("Request timed out after $tokenTimeout.")
    } catch (e: CancellationException) {
      throw e
    } catch (t: Throwable) {
      TokenResult.Retryable(t.message ?: "Network error during token exchange.")
    } finally {
      client.close()
    }
  }
}

/** Build `<baseUrl>/<path>` without doubling up slashes or clobbering a trailing path segment. */
internal fun appendPath(baseUrl: String, path: String): String {
  val trimmedBase = baseUrl.trimEnd('/')
  val trimmedPath = path.trimStart('/')
  return "$trimmedBase/$trimmedPath"
}

/**
 * Replacement URL builder when callers need an `io.ktor.http.Url` instance. Currently unused
 * outside tests; kept here because the compose login screen will want it when rendering the
 * authorize URL in the "opening browser" copy.
 */
internal fun authorizeUrl(
  baseUrl: String,
  clientId: String,
  redirectUri: String,
  state: String,
  codeChallenge: String,
  scope: String? = null,
): String {
  val builder = URLBuilder(appendPath(baseUrl = baseUrl, path = "/oauth/authorize"))
  builder.parameters.apply {
    append("response_type", "code")
    append("client_id", clientId)
    append("redirect_uri", redirectUri)
    append("code_challenge", codeChallenge)
    append("code_challenge_method", "S256")
    append("state", state)
    if (scope != null) append("scope", scope)
  }
  return builder.buildString()
}

internal fun defaultHttpClient(): HttpClient = HttpClient {
  expectSuccess = false
  install(ContentNegotiation) { json(Json { ignoreUnknownKeys = true }) }
}
