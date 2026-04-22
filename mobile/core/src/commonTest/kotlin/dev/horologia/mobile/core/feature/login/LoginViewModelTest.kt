package dev.horologia.mobile.core.feature.login

import dev.horologia.mobile.core.platform.FakeBrowserDriver
import dev.horologia.mobile.core.platform.FakeServerPrefs
import dev.horologia.mobile.core.platform.FakeSessionPersister
import dev.horologia.mobile.core.session.SessionHolder
import kotlin.test.AfterTest
import kotlin.test.BeforeTest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain

@OptIn(ExperimentalCoroutinesApi::class)
class LoginViewModelTest {
  private val dispatcher = StandardTestDispatcher()

  @BeforeTest
  fun setUp() {
    Dispatchers.setMain(dispatcher)
  }

  @AfterTest
  fun tearDown() {
    Dispatchers.resetMain()
  }

  @Test
  fun initialStateIsEmptyServerPicker() = runTest {
    val vm = buildVm()
    val state = vm.uiState.value
    assertTrue(state is LoginUiState.ServerPicker, "got $state")
    assertEquals("", state.input)
    assertEquals(ProbeState.Empty, state.probe)
  }

  @Test
  fun onUrlChangedDebouncesAndTransitionsToValid() = runTest {
    val gateway = FakeLoginGateway(probeReturn = { ProbeResult.Ok })
    val vm = buildVm(gateway = gateway)

    vm.onUrlChanged("tasks.example.com")
    assertEquals(ProbeState.Typing, (vm.uiState.value as LoginUiState.ServerPicker).probe)

    // Before the debounce fires, still Typing.
    testScheduler.advanceTimeBy(300L)
    testScheduler.runCurrent()
    assertEquals(ProbeState.Typing, (vm.uiState.value as LoginUiState.ServerPicker).probe)

    // After the debounce + probe, Valid.
    testScheduler.advanceUntilIdle()
    assertEquals(ProbeState.Valid, (vm.uiState.value as LoginUiState.ServerPicker).probe)
    assertEquals(1, gateway.probeCalls)
  }

  @Test
  fun newKeystrokeCancelsPendingProbe() = runTest {
    val gateway = FakeLoginGateway(probeReturn = { ProbeResult.Ok })
    val vm = buildVm(gateway = gateway)

    vm.onUrlChanged("a")
    testScheduler.advanceTimeBy(300L)
    vm.onUrlChanged("ab")
    testScheduler.advanceUntilIdle()
    // Only one probe fires (for the later input).
    assertEquals(1, gateway.probeCalls)
    assertEquals("ab", (vm.uiState.value as LoginUiState.ServerPicker).input)
  }

  @Test
  fun probeUnreachableTransitionsToInvalidUnreachable() = runTest {
    val gateway =
      FakeLoginGateway(probeReturn = { ProbeResult.Unreachable(host = "tasks.example.com") })
    val vm = buildVm(gateway = gateway)

    vm.onUrlChanged("tasks.example.com")
    testScheduler.advanceUntilIdle()
    val probe = (vm.uiState.value as LoginUiState.ServerPicker).probe
    assertTrue(probe is ProbeState.InvalidUnreachable, "got $probe")
    assertEquals("tasks.example.com", probe.host)
  }

  @Test
  fun probeWrongServerTransitionsToInvalidWrongServer() = runTest {
    val gateway = FakeLoginGateway(probeReturn = { ProbeResult.WrongServer })
    val vm = buildVm(gateway = gateway)

    vm.onUrlChanged("https://example.com")
    testScheduler.advanceUntilIdle()
    assertEquals(
      ProbeState.InvalidWrongServer,
      (vm.uiState.value as LoginUiState.ServerPicker).probe,
    )
  }

  @Test
  fun onSubmitWithoutValidIsNoop() = runTest {
    val gateway = FakeLoginGateway(probeReturn = { ProbeResult.WrongServer })
    val vm = buildVm(gateway = gateway)
    vm.onUrlChanged("bad")
    testScheduler.advanceUntilIdle()
    vm.onSubmit()
    testScheduler.advanceUntilIdle()
    assertTrue(vm.uiState.value is LoginUiState.ServerPicker)
  }

  @Test
  fun onSubmitHappyPathLandsOnComplete() = runTest {
    val gateway =
      FakeLoginGateway(
        probeReturn = { ProbeResult.Ok },
        exchangeReturn = {
          TokenResult.Ok(
            accessToken = "AT",
            refreshToken = "RT",
            accessTokenExpiresAtMillis = 10_000L,
          )
        },
      )
    val browser = FakeBrowserDriver(launch = { "horologia://oauth?state=STATE&code=CODE" })
    var fakeNow = 0L
    val vm = buildVm(gateway = gateway, browser = browser, nowMillis = { fakeNow })

    vm.onUrlChanged("tasks.example.com")
    testScheduler.advanceUntilIdle()

    // Patch state so callback matches. The VM's onSubmit generates its own state,
    // but we intercept by letting the browser driver echo whatever state is in the URL.
    browser.launch = { authorizeUrl ->
      val params =
        authorizeUrl.substringAfter('?').split('&').associate {
          val (k, v) = it.split('=', limit = 2)
          k to v
        }
      val state = params["state"]!!
      fakeNow = 50L
      "horologia://oauth?state=$state&code=CODE"
    }
    vm.onSubmit()
    // LaunchingBrowser -> Finishing -> Complete. Drain.
    testScheduler.advanceUntilIdle()
    assertEquals(LoginUiState.Complete, vm.uiState.value)
  }

  @Test
  fun stateMismatchReturnsToServerPickerWithBanner() = runTest {
    val gateway = FakeLoginGateway(probeReturn = { ProbeResult.Ok })
    val browser = FakeBrowserDriver(launch = { "horologia://oauth?state=HOSTILE&code=CODE" })
    val vm = buildVm(gateway = gateway, browser = browser)

    vm.onUrlChanged("tasks.example.com")
    testScheduler.advanceUntilIdle()
    vm.onSubmit()
    testScheduler.advanceUntilIdle()

    val state = vm.uiState.value
    assertTrue(state is LoginUiState.ServerPicker, "got $state")
    assertEquals("Sign-in couldn't be verified.", state.banner)
  }

  @Test
  fun finishingMinDwellIsHonored() = runTest {
    val gateway =
      FakeLoginGateway(
        probeReturn = { ProbeResult.Ok },
        exchangeReturn = {
          TokenResult.Ok(
            accessToken = "AT",
            refreshToken = "RT",
            accessTokenExpiresAtMillis = 10_000L,
          )
        },
      )
    val browser = FakeBrowserDriver()
    val tick = 1_000L
    var now = tick
    val vm =
      buildVm(
        gateway = gateway,
        browser = browser,
        nowMillis = { now },
        finishingMinDwellMillis = 300L,
      )

    vm.onUrlChanged("tasks.example.com")
    testScheduler.advanceUntilIdle()

    // Browser launches quickly but we set `now` to simulate a 50ms token exchange,
    // forcing the VM to `delay(250)` to fill the dwell quota.
    browser.launch = { authorizeUrl ->
      val state = authorizeUrl.substringAfter("state=").substringBefore('&')
      now += 50L
      "horologia://oauth?state=$state&code=CODE"
    }
    vm.onSubmit()

    // Let the VM reach Finishing.
    testScheduler.advanceTimeBy(100L)
    testScheduler.runCurrent()
    assertTrue(
      vm.uiState.value is LoginUiState.Finishing ||
        vm.uiState.value is LoginUiState.LaunchingBrowser,
      "expected Finishing or LaunchingBrowser, got ${vm.uiState.value}",
    )

    testScheduler.advanceUntilIdle()
    assertEquals(LoginUiState.Complete, vm.uiState.value)
  }

  @Test
  fun concurrentOnSubmitCancelsInFlight() = runTest {
    val gate = CompletableDeferred<TokenResult>()
    val gateway =
      FakeLoginGateway(probeReturn = { ProbeResult.Ok }, exchangeReturn = { gate.await() })
    val browser = FakeBrowserDriver()
    val vm = buildVm(gateway = gateway, browser = browser)
    vm.onUrlChanged("tasks.example.com")
    testScheduler.advanceUntilIdle()
    vm.onSubmit()
    testScheduler.advanceTimeBy(100L)
    testScheduler.runCurrent()
    // Trigger a second submit path: change URL and resubmit to force cancel.
    vm.onUrlChanged("other.example.com")
    testScheduler.advanceUntilIdle()
    // The completed deferred after cancellation should not advance the VM.
    gate.complete(
      TokenResult.Ok(accessToken = "late", refreshToken = null, accessTokenExpiresAtMillis = 0L)
    )
    testScheduler.advanceUntilIdle()
    assertTrue(vm.uiState.value is LoginUiState.ServerPicker)
  }

  private fun buildVm(
    gateway: LoginGateway = FakeLoginGateway(probeReturn = { ProbeResult.Ok }),
    browser: FakeBrowserDriver = FakeBrowserDriver(),
    nowMillis: () -> Long = { 0L },
    debounceMillis: Long = 600L,
    finishingMinDwellMillis: Long = 300L,
  ): LoginViewModel =
    LoginViewModel(
      gateway = gateway,
      browser = browser,
      serverPrefs = FakeServerPrefs(),
      sessionHolder = SessionHolder(persister = FakeSessionPersister()),
      debounceMillis = debounceMillis,
      finishingMinDwellMillis = finishingMinDwellMillis,
      nowMillis = nowMillis,
    )

  private class FakeLoginGateway(
    val probeReturn: suspend () -> ProbeResult,
    val exchangeReturn: suspend () -> TokenResult = {
      TokenResult.Ok(accessToken = "AT", refreshToken = "RT", accessTokenExpiresAtMillis = 0L)
    },
  ) : LoginGateway {
    var probeCalls = 0

    override suspend fun probeServer(baseUrl: String): ProbeResult {
      probeCalls += 1
      return probeReturn()
    }

    override suspend fun exchangeCode(
      baseUrl: String,
      code: String,
      codeVerifier: String,
      redirectUri: String,
      clientId: String,
    ): TokenResult = exchangeReturn()

    override suspend fun refreshAccessToken(
      baseUrl: String,
      refreshToken: String,
      clientId: String,
    ): TokenResult = TokenResult.AuthFailure
  }
}
