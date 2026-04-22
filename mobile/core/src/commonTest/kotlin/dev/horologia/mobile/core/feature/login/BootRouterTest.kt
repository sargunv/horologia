package dev.horologia.mobile.core.feature.login

import dev.horologia.mobile.core.platform.FakeServerPrefs
import dev.horologia.mobile.core.platform.FakeSessionPersister
import dev.horologia.mobile.core.session.SessionHolder
import dev.horologia.mobile.core.session.StoredSession
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue
import kotlinx.coroutines.test.runTest

class BootRouterTest {
  @Test
  fun noSavedUrlReturnsUnconfigured() = runTest {
    val router = BootRouter(serverPrefs = FakeServerPrefs(), sessionHolder = holder())
    assertEquals(BootDestination.Unconfigured, router.decideBootDestination())
  }

  @Test
  fun savedUrlWithoutTokensReturnsServerOnly() = runTest {
    val router =
      BootRouter(
        serverPrefs = FakeServerPrefs(url = "https://tasks.example.com"),
        sessionHolder = holder(),
      )
    val result = router.decideBootDestination()
    assertTrue(result is BootDestination.ServerOnly, "got $result")
    assertEquals("https://tasks.example.com", result.savedUrl)
  }

  @Test
  fun savedUrlAndTokensReturnsSignedIn() = runTest {
    val persister = FakeSessionPersister()
    persister.entries["tasks.example.com"] =
      StoredSession(accessToken = "AT", refreshToken = "RT", accessTokenExpiresAtMillis = 0L)
    val router =
      BootRouter(
        serverPrefs = FakeServerPrefs(url = "https://tasks.example.com"),
        sessionHolder = SessionHolder(persister = persister),
      )
    val result = router.decideBootDestination()
    assertTrue(result is BootDestination.SignedIn, "got $result")
    assertEquals("tasks.example.com", result.host)
  }

  @Test
  fun invalidSavedUrlFallsBackToServerOnly() = runTest {
    val router =
      BootRouter(serverPrefs = FakeServerPrefs(url = "not a url"), sessionHolder = holder())
    val result = router.decideBootDestination()
    assertTrue(result is BootDestination.ServerOnly, "got $result")
  }

  private fun holder(): SessionHolder = SessionHolder(persister = FakeSessionPersister())
}
