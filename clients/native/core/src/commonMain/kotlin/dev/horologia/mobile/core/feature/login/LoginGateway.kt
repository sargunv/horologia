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

/**
 * RFC 8414 OAuth 2.0 Authorization Server Metadata. Served at
 * `/.well-known/oauth-authorization-server` by Horologia's server. We probe this endpoint (not
 * `/app/auth/config`) because:
 *
 * - Routes under `/app/` are same-origin-gated to the first-party web client; a headless HTTP
 *   client without an `Origin` header gets 403.
 * - `/.well-known/oauth-authorization-server` is the standards-mandated discovery endpoint for any
 *   OAuth 2.0 client. It's mounted at the server root, unauthenticated, and rejects nothing.
 * - A successful decode (issuer + authorization_endpoint + token_endpoint + `S256` in
 *   `code_challenge_methods_supported`) confirms "this is a Horologia server AND supports the PKCE
 *   mode we need" — stronger than `AuthConfig`, which only told us about password/OIDC UX.
 */
@Serializable
private data class OAuthServerMetadataShape(
  @SerialName("issuer") val issuer: String,
  @SerialName("authorization_endpoint") val authorizationEndpoint: String,
  @SerialName("token_endpoint") val tokenEndpoint: String,
  @SerialName("code_challenge_methods_supported")
  val codeChallengeMethodsSupported: List<String> = emptyList(),
)

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
        val probeUrl =
          appendPath(baseUrl = baseUrl, path = "/.well-known/oauth-authorization-server")
        val response: HttpResponse = client.get(probeUrl)
        if (response.status != HttpStatusCode.OK) {
          return@withTimeout ProbeResult.WrongServer
        }
        try {
          val metadata = response.body<OAuthServerMetadataShape>()
          // Require the exact PKCE mode we'll use. A server that advertises authorization_code
          // but not S256 is not a Horologia server we can talk to.
          if ("S256" !in metadata.codeChallengeMethodsSupported) {
            ProbeResult.WrongServer
          } else {
            ProbeResult.Ok
          }
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
          response.status.value in 400..499 -> TokenResult.Permanent("Sign-in failed. Try again.")
          else -> TokenResult.Retryable("Server error. Try again.")
        }
      }
    } catch (_: TimeoutCancellationException) {
      TokenResult.Retryable("The server took too long to respond. Try again.")
    } catch (e: CancellationException) {
      throw e
    } catch (t: Throwable) {
      // Map exception types to fixed user-facing strings. Upstream Ktor exception
      // messages can leak TLS / URL / certificate detail, so we never forward them
      // to the UI banner — full details are logged here for developer debugging.
      println("LoginGateway.postToken failed: $t")
      classifyTokenException(t)
    } finally {
      client.close()
    }
  }
}

/**
 * Build `<baseUrl>/<path>` by structurally appending to the URL's encoded path, preserving any
 * existing query string / fragment. Uses [URLBuilder] rather than naive string concatenation so
 * base URLs carrying a `?q=…` suffix don't end up with the path wedged mid-query.
 */
internal fun appendPath(baseUrl: String, path: String): String {
  val parsed =
    try {
      Url(baseUrl)
    } catch (_: IllegalArgumentException) {
      // Fall back to the dumb concatenation when the base isn't a full URL
      // (e.g. "tasks.example.com" pre-normalize): preserves existing callsite shape.
      val trimmedBase = baseUrl.trimEnd('/')
      val trimmedPath = path.trimStart('/')
      return "$trimmedBase/$trimmedPath"
    }
  val builder = URLBuilder(parsed)
  val existingSegments = parsed.segments.filter { it.isNotEmpty() }
  val newSegments = path.split('/').filter { it.isNotEmpty() }
  builder.pathSegments = existingSegments + newSegments
  return builder.buildString()
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

/**
 * Map a Throwable from `postToken` to a banner-ready [TokenResult]. Raw messages never reach the
 * UI; type-name sniffing keeps the classifier commonMain-compatible without pulling in
 * `javax.net.ssl` (JVM-only) or similar platform-specific exception APIs.
 */
internal fun classifyTokenException(t: Throwable): TokenResult {
  if (t is JsonConvertException) {
    return TokenResult.Permanent("The server sent an unexpected response. Try again.")
  }
  val typeName = t::class.simpleName.orEmpty()
  val message = t.message.orEmpty()
  return when {
    typeName.contains("UnknownHostException") -> TokenResult.Permanent("Can't reach the server.")
    typeName.contains("SSL") || typeName.contains("Tls") || typeName.contains("Certificate") ->
      TokenResult.Permanent("The server's certificate isn't trusted.")
    typeName.contains("IOException") ||
      typeName.contains("Connect") ||
      typeName.contains("Socket") ||
      message.contains("connection", ignoreCase = true) ->
      TokenResult.Retryable("Network error. Try again.")
    else -> TokenResult.Retryable("Sign-in failed. Try again.")
  }
}
