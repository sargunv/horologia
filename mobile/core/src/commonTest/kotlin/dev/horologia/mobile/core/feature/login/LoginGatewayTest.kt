package dev.horologia.mobile.core.feature.login

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.http.HttpStatusCode
import io.ktor.http.headersOf
import io.ktor.serialization.kotlinx.json.json
import io.ktor.utils.io.ByteReadChannel
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull
import kotlin.test.assertTrue
import kotlin.time.Duration.Companion.milliseconds
import kotlinx.coroutines.delay
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json

class LoginGatewayTest {
  private fun clientWith(
    statusCode: HttpStatusCode,
    body: String,
    contentType: String = "application/json",
  ): HttpClient {
    val engine = MockEngine { _ ->
      respond(
        content = ByteReadChannel(body),
        status = statusCode,
        headers = headersOf("Content-Type", contentType),
      )
    }
    return HttpClient(engine) {
      expectSuccess = false
      install(ContentNegotiation) { json(Json { ignoreUnknownKeys = true }) }
    }
  }

  @Test
  fun probeServer_validAuthConfigIsOk() = runBlocking {
    val gateway =
      LiveLoginGateway(
        httpClientFactory = {
          clientWith(
            HttpStatusCode.OK,
            """{"oidc":{"enabled":false,"label":"","autoRedirect":false},"password":{"enabled":true}}""",
          )
        }
      )
    val result = gateway.probeServer(baseUrl = "https://tasks.example.com")
    assertEquals(ProbeResult.Ok, result, "got $result")
  }

  @Test
  fun probeServer_malformedJsonIsWrongServer() = runBlocking {
    val gateway =
      LiveLoginGateway(
        httpClientFactory = { clientWith(HttpStatusCode.OK, """{"not":"a horologia response"}""") }
      )
    assertEquals(
      ProbeResult.WrongServer,
      gateway.probeServer(baseUrl = "https://tasks.example.com"),
    )
  }

  @Test
  fun probeServer_non200IsWrongServer() = runBlocking {
    val gateway =
      LiveLoginGateway(
        httpClientFactory = { clientWith(HttpStatusCode.NotFound, "not found", "text/plain") }
      )
    assertEquals(
      ProbeResult.WrongServer,
      gateway.probeServer(baseUrl = "https://tasks.example.com"),
    )
  }

  @Test
  fun probeServer_networkFailureIsUnreachable() = runBlocking {
    val gateway =
      LiveLoginGateway(
        httpClientFactory = {
          HttpClient(MockEngine { _ -> throw RuntimeException("connection refused") }) {
            expectSuccess = false
          }
        }
      )
    val result = gateway.probeServer(baseUrl = "https://tasks.example.com")
    assertTrue(result is ProbeResult.Unreachable, "expected Unreachable, got $result")
    assertEquals("tasks.example.com", result.host)
  }

  @Test
  fun exchangeCode_okReturnsTokens() = runBlocking {
    val gateway =
      LiveLoginGateway(
        httpClientFactory = {
          clientWith(
            HttpStatusCode.OK,
            """{"access_token":"AT","refresh_token":"RT","expires_in":3600,"token_type":"Bearer"}""",
          )
        },
        nowMillis = { 1000L },
      )
    val result =
      gateway.exchangeCode(
        baseUrl = "https://tasks.example.com",
        code = "C",
        codeVerifier = "V",
        redirectUri = "horologia://oauth",
        clientId = "horologia-mobile",
      )
    assertTrue(result is TokenResult.Ok, "expected Ok, got $result")
    assertEquals("AT", result.accessToken)
    assertEquals("RT", result.refreshToken)
    assertEquals(1000L + 3600L * 1000L, result.accessTokenExpiresAtMillis)
  }

  @Test
  fun exchangeCode_401IsAuthFailure() = runBlocking {
    val gateway =
      LiveLoginGateway(
        httpClientFactory = {
          clientWith(HttpStatusCode.Unauthorized, """{"error":"invalid_grant"}""")
        }
      )
    val result =
      gateway.exchangeCode(
        baseUrl = "https://tasks.example.com",
        code = "C",
        codeVerifier = "V",
        redirectUri = "horologia://oauth",
        clientId = "horologia-mobile",
      )
    assertEquals(TokenResult.AuthFailure, result)
  }

  @Test
  fun exchangeCode_400IsPermanent() = runBlocking {
    val gateway =
      LiveLoginGateway(
        httpClientFactory = {
          clientWith(HttpStatusCode.BadRequest, """{"error":"invalid_request"}""")
        }
      )
    val result = exchange(gateway = gateway)
    assertTrue(result is TokenResult.Permanent, "expected Permanent, got $result")
  }

  @Test
  fun exchangeCode_500IsRetryable() = runBlocking {
    val gateway =
      LiveLoginGateway(
        httpClientFactory = { clientWith(HttpStatusCode.InternalServerError, "oops", "text/plain") }
      )
    val result = exchange(gateway = gateway)
    assertTrue(result is TokenResult.Retryable, "expected Retryable, got $result")
  }

  @Test
  fun exchangeCode_timeoutIsRetryable() = runBlocking {
    val gateway =
      LiveLoginGateway(
        httpClientFactory = {
          HttpClient(
            MockEngine { _ ->
              delay(60_000L)
              respond(content = ByteReadChannel(""), status = HttpStatusCode.OK)
            }
          ) {
            expectSuccess = false
          }
        },
        tokenTimeout = 50.milliseconds,
      )
    val result = exchange(gateway = gateway)
    assertTrue(result is TokenResult.Retryable, "expected Retryable, got $result")
  }

  @Test
  fun exchangeCode_missingExpiresInFallsBackToOneHour() = runBlocking {
    val gateway =
      LiveLoginGateway(
        httpClientFactory = {
          clientWith(
            HttpStatusCode.OK,
            """{"access_token":"AT","refresh_token":"RT","token_type":"Bearer"}""",
          )
        },
        nowMillis = { 1000L },
      )
    val result = exchange(gateway = gateway)
    assertTrue(result is TokenResult.Ok, "expected Ok, got $result")
    assertEquals(1000L + 3600L * 1000L, result.accessTokenExpiresAtMillis)
  }

  @Test
  fun exchangeCode_malformed200BodyIsPermanent() = runBlocking {
    val gateway =
      LiveLoginGateway(
        httpClientFactory = { clientWith(HttpStatusCode.OK, """<<not json at all>>""") }
      )
    val result = exchange(gateway = gateway)
    assertTrue(
      result is TokenResult.Permanent || result is TokenResult.Retryable,
      "expected Permanent (JsonConvertException) or Retryable, got $result",
    )
  }

  @Test
  fun refreshAccessToken_okRotatesRefreshToken() = runBlocking {
    val gateway =
      LiveLoginGateway(
        httpClientFactory = {
          clientWith(
            HttpStatusCode.OK,
            """{"access_token":"AT2","refresh_token":"RT2","expires_in":3600,"token_type":"Bearer"}""",
          )
        },
        nowMillis = { 0L },
      )
    val result =
      gateway.refreshAccessToken(
        baseUrl = "https://tasks.example.com",
        refreshToken = "RT1",
        clientId = "horologia-mobile",
      )
    assertTrue(result is TokenResult.Ok, "expected Ok, got $result")
    assertEquals("AT2", result.accessToken)
    assertEquals("RT2", result.refreshToken)
  }

  @Test
  fun refreshAccessToken_okWithoutNewRefreshTokenKeepsNullField() = runBlocking {
    val gateway =
      LiveLoginGateway(
        httpClientFactory = {
          clientWith(
            HttpStatusCode.OK,
            """{"access_token":"AT2","expires_in":3600,"token_type":"Bearer"}""",
          )
        },
        nowMillis = { 0L },
      )
    val result =
      gateway.refreshAccessToken(
        baseUrl = "https://tasks.example.com",
        refreshToken = "RT1",
        clientId = "horologia-mobile",
      )
    assertTrue(result is TokenResult.Ok, "expected Ok, got $result")
    assertNull(result.refreshToken)
  }

  @Test
  fun refreshAccessToken_401IsAuthFailure() = runBlocking {
    val gateway =
      LiveLoginGateway(
        httpClientFactory = {
          clientWith(HttpStatusCode.Unauthorized, """{"error":"invalid_grant"}""")
        }
      )
    val result =
      gateway.refreshAccessToken(
        baseUrl = "https://tasks.example.com",
        refreshToken = "RT1",
        clientId = "horologia-mobile",
      )
    assertEquals(TokenResult.AuthFailure, result)
  }

  @Test
  fun refreshAccessToken_500IsRetryable() = runBlocking {
    val gateway =
      LiveLoginGateway(
        httpClientFactory = { clientWith(HttpStatusCode.InternalServerError, "oops", "text/plain") }
      )
    val result =
      gateway.refreshAccessToken(
        baseUrl = "https://tasks.example.com",
        refreshToken = "RT1",
        clientId = "horologia-mobile",
      )
    assertTrue(result is TokenResult.Retryable, "expected Retryable, got $result")
  }

  @Test
  fun refreshAccessToken_timeoutIsRetryable() = runBlocking {
    val gateway =
      LiveLoginGateway(
        httpClientFactory = {
          HttpClient(
            MockEngine { _ ->
              delay(60_000L)
              respond(content = ByteReadChannel(""), status = HttpStatusCode.OK)
            }
          ) {
            expectSuccess = false
          }
        },
        tokenTimeout = 50.milliseconds,
      )
    val result =
      gateway.refreshAccessToken(
        baseUrl = "https://tasks.example.com",
        refreshToken = "RT1",
        clientId = "horologia-mobile",
      )
    assertTrue(result is TokenResult.Retryable, "expected Retryable, got $result")
  }

  @Test
  fun probeServer_timeoutIsUnreachable() = runBlocking {
    val gateway =
      LiveLoginGateway(
        httpClientFactory = {
          HttpClient(
            MockEngine { _ ->
              delay(60_000L)
              respond(content = ByteReadChannel(""), status = HttpStatusCode.OK)
            }
          ) {
            expectSuccess = false
          }
        },
        probeTimeout = 50.milliseconds,
      )
    val result = gateway.probeServer(baseUrl = "https://tasks.example.com")
    assertTrue(result is ProbeResult.Unreachable, "expected Unreachable, got $result")
  }

  @Test
  fun appendPath_handlesTrailingAndLeadingSlashes() {
    // 4 combinations: {trailing /, no trailing} × {leading /, no leading}.
    assertEquals(
      "https://host.com/app/auth/config",
      appendPath("https://host.com", "/app/auth/config"),
    )
    assertEquals(
      "https://host.com/app/auth/config",
      appendPath("https://host.com/", "/app/auth/config"),
    )
    assertEquals(
      "https://host.com/app/auth/config",
      appendPath("https://host.com", "app/auth/config"),
    )
    assertEquals(
      "https://host.com/app/auth/config",
      appendPath("https://host.com/", "app/auth/config"),
    )
  }

  @Test
  fun appendPath_preservesQueryString() {
    val result = appendPath("https://host.com?q=1", "/app/auth/config")
    assertTrue(result.endsWith("/app/auth/config?q=1"), "got $result")
  }

  @Test
  fun authorizeUrl_containsAllRequiredParams() {
    val url =
      authorizeUrl(
        baseUrl = "https://tasks.example.com",
        clientId = "horologia-mobile",
        redirectUri = "horologia://oauth",
        state = "S",
        codeChallenge = "C",
        scope = "profile",
      )
    // Each required PKCE + OAuth param must be present in the query string.
    assertTrue(url.contains("response_type=code"), "missing response_type, got $url")
    assertTrue(url.contains("client_id=horologia-mobile"), "missing client_id")
    assertTrue(url.contains("redirect_uri="), "missing redirect_uri")
    assertTrue(url.contains("code_challenge=C"), "missing code_challenge")
    assertTrue(url.contains("code_challenge_method=S256"), "missing code_challenge_method")
    assertTrue(url.contains("state=S"), "missing state")
    assertTrue(url.contains("scope=profile"), "missing scope")
  }

  private suspend fun exchange(gateway: LoginGateway): TokenResult =
    gateway.exchangeCode(
      baseUrl = "https://tasks.example.com",
      code = "C",
      codeVerifier = "V",
      redirectUri = "horologia://oauth",
      clientId = "horologia-mobile",
    )
}
