package dev.horologia.mobile.core.feature.login

import dev.horologia.mobile.core.platform.FakeServerPrefs
import dev.horologia.mobile.core.platform.FakeSessionStore
import dev.horologia.mobile.core.session.SessionHolder
import dev.horologia.mobile.core.session.StoredSession
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull
import kotlin.test.assertTrue
import kotlinx.coroutines.test.runTest

class BootRouterTest {
  @Test
  fun noSavedUrlReturnsUnconfigured() = runTest {
    val router =
      BootRouter(
        serverPrefs = FakeServerPrefs(),
        sessionHolder = holder(),
        loginGateway = NeverCalledGateway(),
        nowMillis = { 0L },
      )
    assertEquals(BootDestination.Unconfigured, router.decideBootDestination())
  }

  @Test
  fun savedUrlWithoutTokensReturnsServerOnly() = runTest {
    val router =
      BootRouter(
        serverPrefs = FakeServerPrefs(url = "https://tasks.example.com"),
        sessionHolder = holder(),
        loginGateway = NeverCalledGateway(),
        nowMillis = { 0L },
      )
    val result = router.decideBootDestination()
    assertTrue(result is BootDestination.ServerOnly, "got $result")
    assertEquals("https://tasks.example.com", result.savedUrl)
  }

  @Test
  fun savedUrlAndTokensReturnsSignedIn() = runTest {
    val store = FakeSessionStore()
    store.entries["tasks.example.com"] =
      StoredSession(
        accessToken = "AT",
        refreshToken = "RT",
        accessTokenExpiresAtMillis = 10_000_000L,
      )
    val gateway = CountingGateway()
    val router =
      BootRouter(
        serverPrefs = FakeServerPrefs(url = "https://tasks.example.com"),
        sessionHolder = SessionHolder(store = store),
        loginGateway = gateway,
        nowMillis = { 0L },
      )
    val result = router.decideBootDestination()
    assertTrue(result is BootDestination.SignedIn, "got $result")
    assertEquals("tasks.example.com", result.host)
    assertEquals(0, gateway.refreshCalls)
  }

  @Test
  fun invalidSavedUrlFallsBackToServerOnly() = runTest {
    val router =
      BootRouter(
        serverPrefs = FakeServerPrefs(url = "not a url"),
        sessionHolder = holder(),
        loginGateway = NeverCalledGateway(),
        nowMillis = { 0L },
      )
    val result = router.decideBootDestination()
    assertTrue(result is BootDestination.ServerOnly, "got $result")
  }

  @Test
  fun refreshedSessionWithinExpiryWindowReturnsSignedIn() = runTest {
    val store = FakeSessionStore()
    val now = 1_000_000L
    // Within the 60s leeway — expires 10s from now.
    store.entries["tasks.example.com"] =
      StoredSession(
        accessToken = "AT-old",
        refreshToken = "RT-old",
        accessTokenExpiresAtMillis = now + 10_000L,
      )
    val holder = SessionHolder(store = store)
    val gateway =
      CountingGateway(
        refreshReturn = {
          TokenResult.Ok(
            accessToken = "AT-new",
            refreshToken = "RT-new",
            accessTokenExpiresAtMillis = now + 3_600_000L,
          )
        }
      )
    val router =
      BootRouter(
        serverPrefs = FakeServerPrefs(url = "https://tasks.example.com"),
        sessionHolder = holder,
        loginGateway = gateway,
        nowMillis = { now },
      )
    val result = router.decideBootDestination()
    assertTrue(result is BootDestination.SignedIn, "got $result")
    assertEquals(1, gateway.refreshCalls)
    val stored = store.entries["tasks.example.com"]!!
    assertEquals("AT-new", stored.accessToken)
    assertEquals("RT-new", stored.refreshToken)
  }

  @Test
  fun refreshKeepsOldRefreshTokenWhenServerOmitsIt() = runTest {
    val store = FakeSessionStore()
    val now = 1_000_000L
    store.entries["tasks.example.com"] =
      StoredSession(
        accessToken = "AT-old",
        refreshToken = "RT-old",
        accessTokenExpiresAtMillis = now - 1L,
      )
    val gateway =
      CountingGateway(
        refreshReturn = {
          TokenResult.Ok(
            accessToken = "AT-new",
            refreshToken = null,
            accessTokenExpiresAtMillis = now + 3_600_000L,
          )
        }
      )
    val router =
      BootRouter(
        serverPrefs = FakeServerPrefs(url = "https://tasks.example.com"),
        sessionHolder = SessionHolder(store = store),
        loginGateway = gateway,
        nowMillis = { now },
      )
    val result = router.decideBootDestination()
    assertTrue(result is BootDestination.SignedIn, "got $result")
    assertEquals("RT-old", store.entries["tasks.example.com"]!!.refreshToken)
  }

  @Test
  fun refreshAuthFailureReturnsSignedOutAfterRefresh() = runTest {
    val store = FakeSessionStore()
    val now = 1_000_000L
    store.entries["tasks.example.com"] =
      StoredSession(accessToken = "AT", refreshToken = "RT", accessTokenExpiresAtMillis = now - 1L)
    val gateway = CountingGateway(refreshReturn = { TokenResult.AuthFailure })
    val router =
      BootRouter(
        serverPrefs = FakeServerPrefs(url = "https://tasks.example.com"),
        sessionHolder = SessionHolder(store = store),
        loginGateway = gateway,
        nowMillis = { now },
      )
    val result = router.decideBootDestination()
    assertTrue(result is BootDestination.SignedOutAfterRefresh, "got $result")
    assertEquals("https://tasks.example.com", result.savedUrl)
    assertNull(store.entries["tasks.example.com"])
  }

  @Test
  fun refreshPermanentReturnsSignedOutAfterRefresh() = runTest {
    val store = FakeSessionStore()
    val now = 1_000_000L
    store.entries["tasks.example.com"] =
      StoredSession(accessToken = "AT", refreshToken = "RT", accessTokenExpiresAtMillis = now - 1L)
    val gateway = CountingGateway(refreshReturn = { TokenResult.Permanent("nope") })
    val router =
      BootRouter(
        serverPrefs = FakeServerPrefs(url = "https://tasks.example.com"),
        sessionHolder = SessionHolder(store = store),
        loginGateway = gateway,
        nowMillis = { now },
      )
    val result = router.decideBootDestination()
    assertTrue(result is BootDestination.SignedOutAfterRefresh, "got $result")
    assertNull(store.entries["tasks.example.com"])
  }

  @Test
  fun refreshRetryableReturnsSignedInOptimistic() = runTest {
    val store = FakeSessionStore()
    val now = 1_000_000L
    store.entries["tasks.example.com"] =
      StoredSession(
        accessToken = "AT-old",
        refreshToken = "RT-old",
        accessTokenExpiresAtMillis = now - 1L,
      )
    val gateway = CountingGateway(refreshReturn = { TokenResult.Retryable("transient") })
    val router =
      BootRouter(
        serverPrefs = FakeServerPrefs(url = "https://tasks.example.com"),
        sessionHolder = SessionHolder(store = store),
        loginGateway = gateway,
        nowMillis = { now },
      )
    val result = router.decideBootDestination()
    assertTrue(result is BootDestination.SignedIn, "got $result")
    // Stored session untouched — first real API call will 401 and surface through the classifier.
    val stored = store.entries["tasks.example.com"]!!
    assertEquals("AT-old", stored.accessToken)
    assertEquals("RT-old", stored.refreshToken)
  }

  @Test
  fun noRefreshTokenWithExpiredAccessReturnsSignedOutAfterRefresh() = runTest {
    val store = FakeSessionStore()
    val now = 1_000_000L
    store.entries["tasks.example.com"] =
      StoredSession(accessToken = "AT", refreshToken = null, accessTokenExpiresAtMillis = now - 1L)
    val gateway = CountingGateway()
    val router =
      BootRouter(
        serverPrefs = FakeServerPrefs(url = "https://tasks.example.com"),
        sessionHolder = SessionHolder(store = store),
        loginGateway = gateway,
        nowMillis = { now },
      )
    val result = router.decideBootDestination()
    assertTrue(result is BootDestination.SignedOutAfterRefresh, "got $result")
    assertEquals(0, gateway.refreshCalls)
    assertNull(store.entries["tasks.example.com"])
  }

  @Test
  fun freshAccessTokenSkipsRefreshCall() = runTest {
    val store = FakeSessionStore()
    val now = 1_000_000L
    // 10 minutes of headroom, well outside the 60s leeway.
    store.entries["tasks.example.com"] =
      StoredSession(
        accessToken = "AT",
        refreshToken = "RT",
        accessTokenExpiresAtMillis = now + 10 * 60_000L,
      )
    val gateway = CountingGateway()
    val router =
      BootRouter(
        serverPrefs = FakeServerPrefs(url = "https://tasks.example.com"),
        sessionHolder = SessionHolder(store = store),
        loginGateway = gateway,
        nowMillis = { now },
      )
    val result = router.decideBootDestination()
    assertTrue(result is BootDestination.SignedIn, "got $result")
    assertEquals(0, gateway.refreshCalls)
  }

  private fun holder(): SessionHolder = SessionHolder(store = FakeSessionStore())

  private class NeverCalledGateway : LoginGateway {
    override suspend fun probeServer(baseUrl: String): ProbeResult =
      error("probeServer should not be called")

    override suspend fun exchangeCode(
      baseUrl: String,
      code: String,
      codeVerifier: String,
      redirectUri: String,
      clientId: String,
    ): TokenResult = error("exchangeCode should not be called")

    override suspend fun refreshAccessToken(
      baseUrl: String,
      refreshToken: String,
      clientId: String,
    ): TokenResult = error("refreshAccessToken should not be called")
  }

  private class CountingGateway(
    val refreshReturn: suspend () -> TokenResult = { TokenResult.AuthFailure }
  ) : LoginGateway {
    var refreshCalls: Int = 0
      private set

    override suspend fun probeServer(baseUrl: String): ProbeResult =
      error("probeServer should not be called")

    override suspend fun exchangeCode(
      baseUrl: String,
      code: String,
      codeVerifier: String,
      redirectUri: String,
      clientId: String,
    ): TokenResult = error("exchangeCode should not be called")

    override suspend fun refreshAccessToken(
      baseUrl: String,
      refreshToken: String,
      clientId: String,
    ): TokenResult {
      refreshCalls += 1
      return refreshReturn()
    }
  }
}
