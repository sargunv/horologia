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
import kotlin.test.assertTrue
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
}
