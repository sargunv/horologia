package dev.horologia.mobile.core.feature.login

import dev.horologia.mobile.core.platform.FakeBrowserLauncher
import dev.horologia.mobile.core.platform.FakeServerPrefs
import dev.horologia.mobile.core.platform.FakeSessionStore
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
    val browser = FakeBrowserLauncher(launch = { "horologia://oauth?state=STATE&code=CODE" })
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
    val browser = FakeBrowserLauncher(launch = { "horologia://oauth?state=HOSTILE&code=CODE" })
    val vm = buildVm(gateway = gateway, browser = browser)

    vm.onUrlChanged("tasks.example.com")
    testScheduler.advanceUntilIdle()
    vm.onSubmit()
    testScheduler.advanceUntilIdle()

    val state = vm.uiState.value
    assertTrue(state is LoginUiState.ServerPicker, "got $state")
    assertEquals("Sign-in failed. Try again.", state.banner)
  }

  @Test
  fun finishing_dwellsUntilMinimum() = runTest {
    // Seed a harness that gates the token exchange on a deferred, so we can observe the
    // Finishing state mid-dwell without the drain running through to Complete.
    val release = CompletableDeferred<Unit>()
    val gateway =
      FakeLoginGateway(
        probeReturn = { ProbeResult.Ok },
        exchangeReturn = {
          release.await()
          TokenResult.Ok(
            accessToken = "AT",
            refreshToken = "RT",
            accessTokenExpiresAtMillis = 10_000L,
          )
        },
      )
    val browser = FakeBrowserLauncher()
    var now = 0L
    val vm =
      buildVm(
        gateway = gateway,
        browser = browser,
        nowMillis = { now },
        finishingMinDwellMillis = 300L,
      )
    browser.launch = { authorizeUrl ->
      val state = authorizeUrl.substringAfter("state=").substringBefore('&')
      "horologia://oauth?state=$state&code=CODE"
    }
    vm.onUrlChanged("tasks.example.com")
    testScheduler.advanceUntilIdle()
    vm.onSubmit()
    testScheduler.advanceUntilIdle()
    // The VM parked inside gateway.exchangeCode; advance `now` to simulate the exchange
    // taking 50 ms, then release — VM will see 50 ms elapsed and need to dwell 250 ms.
    now = 50L
    release.complete(Unit)
    testScheduler.advanceTimeBy(200L)
    testScheduler.runCurrent()
    assertTrue(
      vm.uiState.value is LoginUiState.Finishing,
      "expected Finishing, got ${vm.uiState.value}",
    )
  }

  @Test
  fun finishing_completesExactlyAtDwellBoundary() = runTest {
    val (vm, _, _) = buildDwellHarness(dwellMillis = 300L, exchangeCost = 50L)

    vm.onUrlChanged("tasks.example.com")
    testScheduler.advanceUntilIdle()
    vm.onSubmit()
    testScheduler.advanceUntilIdle()
    // The VM's `delay(remaining)` is driven by the test-scheduler's virtual time; after
    // draining it we should land on Complete.
    assertEquals(LoginUiState.Complete, vm.uiState.value)
  }

  private fun buildDwellHarness(
    dwellMillis: Long,
    exchangeCost: Long,
  ): Triple<LoginViewModel, FakeBrowserLauncher, () -> Long> {
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
    val browser = FakeBrowserLauncher()
    var now = 1_000L
    val vm =
      buildVm(
        gateway = gateway,
        browser = browser,
        nowMillis = { now },
        finishingMinDwellMillis = dwellMillis,
      )
    browser.launch = { authorizeUrl ->
      val state = authorizeUrl.substringAfter("state=").substringBefore('&')
      now += exchangeCost
      "horologia://oauth?state=$state&code=CODE"
    }
    return Triple(vm, browser, { now })
  }

  @Test
  fun concurrentOnSubmitCancelsInFlight() = runTest {
    val gate = CompletableDeferred<TokenResult>()
    val gateway =
      FakeLoginGateway(probeReturn = { ProbeResult.Ok }, exchangeReturn = { gate.await() })
    val browser = FakeBrowserLauncher()
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

  @Test
  fun tokenExchangeAuthFailureShowsBanner() = runTest {
    val gateway =
      FakeLoginGateway(
        probeReturn = { ProbeResult.Ok },
        exchangeReturn = { TokenResult.AuthFailure },
      )
    val browser = echoingBrowser()
    val vm = buildVm(gateway = gateway, browser = browser)
    vm.onUrlChanged("tasks.example.com")
    testScheduler.advanceUntilIdle()
    vm.onSubmit()
    testScheduler.advanceUntilIdle()
    val state = vm.uiState.value
    assertTrue(state is LoginUiState.ServerPicker, "got $state")
    assertEquals("Sign-in failed.", state.banner)
  }

  @Test
  fun tokenExchangeRetryableSurfacesMessage() = runTest {
    val gateway =
      FakeLoginGateway(
        probeReturn = { ProbeResult.Ok },
        exchangeReturn = { TokenResult.Retryable("Network error. Try again.") },
      )
    val browser = echoingBrowser()
    val vm = buildVm(gateway = gateway, browser = browser)
    vm.onUrlChanged("tasks.example.com")
    testScheduler.advanceUntilIdle()
    vm.onSubmit()
    testScheduler.advanceUntilIdle()
    val state = vm.uiState.value
    assertTrue(state is LoginUiState.ServerPicker, "got $state")
    assertEquals("Network error. Try again.", state.banner)
  }

  @Test
  fun tokenExchangePermanentSurfacesMessage() = runTest {
    val gateway =
      FakeLoginGateway(
        probeReturn = { ProbeResult.Ok },
        exchangeReturn = { TokenResult.Permanent("Unexpected response. Try again.") },
      )
    val browser = echoingBrowser()
    val vm = buildVm(gateway = gateway, browser = browser)
    vm.onUrlChanged("tasks.example.com")
    testScheduler.advanceUntilIdle()
    vm.onSubmit()
    testScheduler.advanceUntilIdle()
    val state = vm.uiState.value
    assertTrue(state is LoginUiState.ServerPicker, "got $state")
    assertEquals("Unexpected response. Try again.", state.banner)
  }

  private fun echoingBrowser(): FakeBrowserLauncher =
    FakeBrowserLauncher(
      launch = { url ->
        val s = url.substringAfter("state=").substringBefore('&')
        "horologia://oauth?state=$s&code=CODE"
      }
    )

  @Test
  fun callbackUrlUnparseableResetsToPicker() = runTest {
    val gateway = FakeLoginGateway(probeReturn = { ProbeResult.Ok })
    val browser = FakeBrowserLauncher(launch = { "@@@ not a url" })
    val vm = buildVm(gateway = gateway, browser = browser)
    vm.onUrlChanged("tasks.example.com")
    testScheduler.advanceUntilIdle()
    vm.onSubmit()
    testScheduler.advanceUntilIdle()
    val state = vm.uiState.value
    assertTrue(state is LoginUiState.ServerPicker, "got $state")
    assertEquals("Sign-in failed. Try again.", state.banner)
  }

  @Test
  fun callbackErrorAccessDeniedShowsCancelledBanner() = runTest {
    val gateway = FakeLoginGateway(probeReturn = { ProbeResult.Ok })
    // Echo state so the state match passes, then add error=access_denied.
    val browser =
      FakeBrowserLauncher(
        launch = { url ->
          val s = url.substringAfter("state=").substringBefore('&')
          "horologia://oauth?state=$s&error=access_denied"
        }
      )
    val vm = buildVm(gateway = gateway, browser = browser)
    vm.onUrlChanged("tasks.example.com")
    testScheduler.advanceUntilIdle()
    vm.onSubmit()
    testScheduler.advanceUntilIdle()
    val state = vm.uiState.value
    assertTrue(state is LoginUiState.ServerPicker, "got $state")
    assertEquals("Sign-in cancelled.", state.banner)
  }

  @Test
  fun callbackErrorOtherShowsFailedBanner() = runTest {
    val gateway = FakeLoginGateway(probeReturn = { ProbeResult.Ok })
    val browser =
      FakeBrowserLauncher(
        launch = { url ->
          val s = url.substringAfter("state=").substringBefore('&')
          "horologia://oauth?state=$s&error=server_error"
        }
      )
    val vm = buildVm(gateway = gateway, browser = browser)
    vm.onUrlChanged("tasks.example.com")
    testScheduler.advanceUntilIdle()
    vm.onSubmit()
    testScheduler.advanceUntilIdle()
    val state = vm.uiState.value
    assertTrue(state is LoginUiState.ServerPicker, "got $state")
    assertEquals("Sign-in failed.", state.banner)
  }

  @Test
  fun callbackMissingCodeShowsFailedBanner() = runTest {
    val gateway = FakeLoginGateway(probeReturn = { ProbeResult.Ok })
    val browser =
      FakeBrowserLauncher(
        launch = { url ->
          val s = url.substringAfter("state=").substringBefore('&')
          "horologia://oauth?state=$s"
        }
      )
    val vm = buildVm(gateway = gateway, browser = browser)
    vm.onUrlChanged("tasks.example.com")
    testScheduler.advanceUntilIdle()
    vm.onSubmit()
    testScheduler.advanceUntilIdle()
    val state = vm.uiState.value
    assertTrue(state is LoginUiState.ServerPicker, "got $state")
    assertEquals("Sign-in failed.", state.banner)
  }

  @Test
  fun browserCancelledResetsToPickerWithCancelledBanner() = runTest {
    val gateway = FakeLoginGateway(probeReturn = { ProbeResult.Ok })
    val browser =
      FakeBrowserLauncher(
        launch = { throw dev.horologia.mobile.core.platform.BrowserCancelledException() }
      )
    val vm = buildVm(gateway = gateway, browser = browser)
    vm.onUrlChanged("tasks.example.com")
    testScheduler.advanceUntilIdle()
    vm.onSubmit()
    testScheduler.advanceUntilIdle()
    val state = vm.uiState.value
    assertTrue(state is LoginUiState.ServerPicker, "got $state")
    assertEquals("OAuth sign-in cancelled by user", state.banner)
  }

  @Test
  fun seedInitialUrlPrefillsAndProbes() = runTest {
    val gateway = FakeLoginGateway(probeReturn = { ProbeResult.Ok })
    val vm = buildVm(gateway = gateway)
    vm.seedInitialUrl(url = "tasks.example.com")
    // Seeded probe bypasses debounce; drain the launched job.
    testScheduler.advanceUntilIdle()
    val state = vm.uiState.value
    assertTrue(state is LoginUiState.ServerPicker, "got $state")
    assertEquals("tasks.example.com", state.input)
    assertEquals(ProbeState.Valid, state.probe)
    assertEquals(1, gateway.probeCalls)
  }

  @Test
  fun seedInitialUrlNoopWhenInputAlreadyPresent() = runTest {
    val gateway = FakeLoginGateway(probeReturn = { ProbeResult.Ok })
    val vm = buildVm(gateway = gateway)
    vm.onUrlChanged("first.example.com")
    testScheduler.advanceUntilIdle()
    vm.seedInitialUrl(url = "second.example.com")
    testScheduler.advanceUntilIdle()
    val state = vm.uiState.value
    assertTrue(state is LoginUiState.ServerPicker, "got $state")
    assertEquals("first.example.com", state.input)
  }

  @Test
  fun seedInitialUrlNoopWhenNotServerPicker() = runTest {
    // Drive the VM into LaunchingBrowser via onSubmit, then try to seed.
    val gateway = FakeLoginGateway(probeReturn = { ProbeResult.Ok })
    val browser = FakeBrowserLauncher(launch = { CompletableDeferred<String>().await() })
    val vm = buildVm(gateway = gateway, browser = browser)
    vm.onUrlChanged("tasks.example.com")
    testScheduler.advanceUntilIdle()
    vm.onSubmit()
    testScheduler.advanceTimeBy(100L)
    testScheduler.runCurrent()
    assertTrue(vm.uiState.value is LoginUiState.LaunchingBrowser)
    vm.seedInitialUrl(url = "other.example.com")
    testScheduler.runCurrent()
    assertTrue(vm.uiState.value is LoginUiState.LaunchingBrowser)
  }

  @Test
  fun showBannerAttachesToCurrentServerPicker() = runTest {
    val vm = buildVm()
    vm.onUrlChanged("tasks.example.com")
    testScheduler.advanceUntilIdle()
    vm.showBanner(message = "Signed out.")
    val state = vm.uiState.value
    assertTrue(state is LoginUiState.ServerPicker, "got $state")
    assertEquals("Signed out.", state.banner)
  }

  @Test
  fun dismissBannerClearsBanner() = runTest {
    val vm = buildVm()
    vm.showBanner(message = "Signed out.")
    vm.dismissBanner()
    val state = vm.uiState.value
    assertTrue(state is LoginUiState.ServerPicker, "got $state")
    assertEquals(null, state.banner)
  }

  private fun buildVm(
    gateway: LoginGateway = FakeLoginGateway(probeReturn = { ProbeResult.Ok }),
    browser: FakeBrowserLauncher = FakeBrowserLauncher(),
    nowMillis: () -> Long = { 0L },
    debounceMillis: Long = 600L,
    finishingMinDwellMillis: Long = 300L,
  ): LoginViewModel =
    LoginViewModel(
      gateway = gateway,
      browser = browser,
      serverPrefs = FakeServerPrefs(),
      sessionHolder = SessionHolder(store = FakeSessionStore()),
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
