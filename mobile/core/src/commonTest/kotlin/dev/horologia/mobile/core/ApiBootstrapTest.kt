package dev.horologia.mobile.core

import dev.horologia.mobile.generated.Api
import dev.horologia.mobile.generated.Auth
import io.ktor.http.Url
import kotlin.test.AfterTest
import kotlin.test.BeforeTest
import kotlin.test.Test
import kotlin.test.assertEquals

/**
 * Regression guard for the adversarial finding in `workpad.md`: Ktor 3.4.2's
 * `DefaultRequest.Plugin.install` reads the request builder per-call, so rebasing `Api.baseUrl`
 * through a second `configureHorologiaApi(...)` call should take effect on the next outbound
 * request. A future Ktor upgrade that reverts to install-time URL capture would break this test.
 *
 * The test doesn't round-trip an actual HTTP call (that would require wiring ContentNegotiation + a
 * MockEngine into the generated Api client, which isn't in this codebase's surface area today).
 * Instead it asserts the narrower invariant: `Api.baseUrl` reflects the most recent
 * `configureHorologiaApi` call, so the `apiConfig$lambda$1` per-request reader sees the fresh
 * value.
 */
class ApiBootstrapTest {
  private val defaultBaseUrl = Api.servers.first()

  @BeforeTest
  fun setUp() {
    Api.baseUrl = defaultBaseUrl
    Api.setAuthProvider(Auth.BearerAuth { null })
  }

  @AfterTest
  fun tearDown() {
    Api.baseUrl = defaultBaseUrl
    Api.setAuthProvider(Auth.BearerAuth { null })
  }

  @Test
  fun configureHorologiaApi_rebasesOnRecall() {
    configureHorologiaApi(baseUrl = "https://first.example.com/api/", getToken = { null })
    assertEquals(Url("https://first.example.com/api/"), Api.baseUrl)

    configureHorologiaApi(baseUrl = "https://second.example.com/api/", getToken = { null })
    assertEquals(Url("https://second.example.com/api/"), Api.baseUrl)
  }

  @Test
  fun ensureApiPath_appendsApiSegment() {
    assertEquals("https://tasks.example.com/api/", ensureApiPath("https://tasks.example.com"))
    assertEquals("https://tasks.example.com/api/", ensureApiPath("https://tasks.example.com/"))
  }

  @Test
  fun ensureApiPath_preservesExistingApiSegment() {
    assertEquals("https://tasks.example.com/api/", ensureApiPath("https://tasks.example.com/api"))
    assertEquals("https://tasks.example.com/api/", ensureApiPath("https://tasks.example.com/api/"))
    assertEquals(
      "https://tasks.example.com/api/v2/",
      ensureApiPath("https://tasks.example.com/api/v2"),
    )
  }
}
