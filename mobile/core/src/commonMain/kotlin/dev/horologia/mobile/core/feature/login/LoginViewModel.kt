package dev.horologia.mobile.core.feature.login

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dev.horologia.mobile.core.platform.BrowserCancelledException
import dev.horologia.mobile.core.platform.BrowserDriver
import dev.horologia.mobile.core.platform.BrowserLauncher
import dev.horologia.mobile.core.platform.BrowserLauncherDriver
import dev.horologia.mobile.core.session.ServerPrefs
import dev.horologia.mobile.core.session.ServerPrefsAdapter
import dev.horologia.mobile.core.session.ServerPrefsReader
import dev.horologia.mobile.core.session.SessionHolder
import dev.horologia.mobile.core.session.StoredSession
import io.ktor.http.Url
import kotlin.time.Clock
import kotlin.time.ExperimentalTime
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

/**
 * Whole-flow owner for the login surface: server-picker debounced probe, PKCE + state generation,
 * browser handoff, token exchange, min-dwell finishing state. Mirrors `feature/profile`'s single-VM
 * template byte-for-byte on structure (StateFlow<UiState>, cancel-safe Job fields, internal
 * constructor).
 *
 * PKCE verifier and OAuth state are VM-private fields, cleared in every terminal path (success,
 * cancellation, state mismatch, exception) — per R6 they must never hit disk.
 */
@OptIn(ExperimentalTime::class)
class LoginViewModel
internal constructor(
  private val gateway: LoginGateway,
  private val browser: BrowserDriver,
  private val serverPrefs: ServerPrefsReader,
  private val sessionHolder: SessionHolder,
  private val clientId: String = DEFAULT_CLIENT_ID,
  private val debounceMillis: Long = DEFAULT_DEBOUNCE_MILLIS,
  private val finishingMinDwellMillis: Long = DEFAULT_FINISHING_MIN_DWELL_MILLIS,
  private val nowMillis: () -> Long = { Clock.System.now().toEpochMilliseconds() },
) : ViewModel() {
  internal constructor(
    gateway: LoginGateway,
    browserLauncher: BrowserLauncher,
    serverPrefs: ServerPrefs,
    sessionHolder: SessionHolder,
    clientId: String = DEFAULT_CLIENT_ID,
    debounceMillis: Long = DEFAULT_DEBOUNCE_MILLIS,
    finishingMinDwellMillis: Long = DEFAULT_FINISHING_MIN_DWELL_MILLIS,
  ) : this(
    gateway = gateway,
    browser = BrowserLauncherDriver(launcher = browserLauncher),
    serverPrefs = ServerPrefsAdapter(prefs = serverPrefs),
    sessionHolder = sessionHolder,
    clientId = clientId,
    debounceMillis = debounceMillis,
    finishingMinDwellMillis = finishingMinDwellMillis,
  )

  private val _uiState =
    MutableStateFlow<LoginUiState>(LoginUiState.ServerPicker(input = "", probe = ProbeState.Empty))
  val uiState: StateFlow<LoginUiState> = _uiState.asStateFlow()

  private var probeJob: Job? = null
  private var flowJob: Job? = null

  // Cleared at every terminal transition.
  private var pendingVerifier: String? = null
  private var pendingState: String? = null
  private var pendingNormalizedUrl: String? = null

  /** Called by the cold-launch router when a saved server URL should be pre-filled. */
  fun seedInitialUrl(url: String) {
    val current = _uiState.value as? LoginUiState.ServerPicker ?: return
    if (current.input.isNotEmpty()) return
    _uiState.value = current.copy(input = url, probe = ProbeState.Typing)
    debounceAndProbe(url = url)
  }

  /** Called after a cold-launch refresh failure. */
  fun showBanner(message: String) {
    val current = _uiState.value as? LoginUiState.ServerPicker ?: return
    _uiState.value = current.copy(banner = message)
  }

  fun onUrlChanged(input: String) {
    val current = _uiState.value
    if (current !is LoginUiState.ServerPicker) return
    if (input.isEmpty()) {
      probeJob?.cancel()
      _uiState.value = current.copy(input = input, probe = ProbeState.Empty, banner = null)
      return
    }
    _uiState.value = current.copy(input = input, probe = ProbeState.Typing, banner = null)
    debounceAndProbe(url = input)
  }

  private fun debounceAndProbe(url: String) {
    probeJob?.cancel()
    probeJob = viewModelScope.launch {
      delay(debounceMillis)
      runProbe(url = url)
    }
  }

  private suspend fun runProbe(url: String) {
    val current = _uiState.value as? LoginUiState.ServerPicker ?: return
    if (current.input != url) return
    val normalized = UrlNormalizer.normalize(url)
    if (normalized == null) {
      _uiState.value = current.copy(probe = ProbeState.InvalidWrongServer)
      return
    }
    _uiState.value = current.copy(probe = ProbeState.Probing)
    val result = gateway.probeServer(baseUrl = normalized)
    val live = _uiState.value as? LoginUiState.ServerPicker ?: return
    if (live.input != url) return
    _uiState.value =
      live.copy(
        probe =
          when (result) {
            ProbeResult.Ok -> ProbeState.Valid
            ProbeResult.WrongServer -> ProbeState.InvalidWrongServer
            is ProbeResult.Unreachable -> ProbeState.InvalidUnreachable(host = result.host)
          }
      )
  }

  /**
   * Fires when the user taps Continue in `Valid` state. Builds PKCE values, persists the server URL
   * (so a subsequent cold launch can recover it), launches the browser, waits for callback,
   * exchanges tokens, persists them, enforces the 300 ms dwell, then lands on
   * [LoginUiState.Complete].
   */
  fun onSubmit() {
    val current = _uiState.value
    if (current !is LoginUiState.ServerPicker) return
    if (current.probe !is ProbeState.Valid) return
    val normalized = UrlNormalizer.normalize(current.input) ?: return

    flowJob?.cancel()
    flowJob = viewModelScope.launch {
      try {
        serverPrefs.saveServerUrl(url = normalized)
        pendingNormalizedUrl = normalized

        val verifier = Pkce.generateCodeVerifier()
        val state = Pkce.generateState()
        val challenge = Pkce.codeChallengeS256(verifier = verifier)
        pendingVerifier = verifier
        pendingState = state

        val redirectUri = browser.redirectUri()
        val authUrl =
          authorizeUrl(
            baseUrl = normalized,
            clientId = clientId,
            redirectUri = redirectUri,
            state = state,
            codeChallenge = challenge,
          )

        _uiState.value = LoginUiState.LaunchingBrowser(input = current.input)
        val callbackUrl =
          try {
            browser.launchAndAwait(authorizeUrl = authUrl)
          } catch (_: BrowserCancelledException) {
            resetToPicker(current = current, banner = "Sign-in cancelled.")
            return@launch
          }

        val parsed = runCatching { Url(callbackUrl) }.getOrNull()
        if (parsed == null) {
          resetToPicker(current = current, banner = "Sign-in couldn't be verified.")
          return@launch
        }
        val params = parsed.parameters
        val returnedState = params["state"]
        if (returnedState != state) {
          resetToPicker(current = current, banner = "Sign-in couldn't be verified.")
          return@launch
        }
        val errorParam = params["error"]
        if (errorParam != null) {
          val banner =
            if (errorParam == "access_denied") "Sign-in cancelled." else "Sign-in failed."
          resetToPicker(current = current, banner = banner)
          return@launch
        }
        val code = params["code"]
        if (code.isNullOrEmpty()) {
          resetToPicker(current = current, banner = "Sign-in failed.")
          return@launch
        }

        _uiState.value = LoginUiState.Finishing(input = current.input)
        val started = nowMillis()
        val tokenResult =
          gateway.exchangeCode(
            baseUrl = normalized,
            code = code,
            codeVerifier = verifier,
            redirectUri = redirectUri,
            clientId = clientId,
          )
        when (tokenResult) {
          is TokenResult.Ok -> {
            val host = runCatching { Url(normalized).host }.getOrNull() ?: normalized
            sessionHolder.install(
              host = host,
              session =
                StoredSession(
                  accessToken = tokenResult.accessToken,
                  refreshToken = tokenResult.refreshToken,
                  accessTokenExpiresAtMillis = tokenResult.accessTokenExpiresAtMillis,
                ),
            )
            val elapsed = nowMillis() - started
            val remaining = finishingMinDwellMillis - elapsed
            if (remaining > 0) delay(remaining)
            _uiState.value = LoginUiState.Complete
          }
          is TokenResult.AuthFailure -> resetToPicker(current = current, banner = "Sign-in failed.")
          is TokenResult.Retryable -> resetToPicker(current = current, banner = tokenResult.message)
          is TokenResult.Permanent -> resetToPicker(current = current, banner = tokenResult.message)
        }
      } catch (e: CancellationException) {
        throw e
      } catch (t: Throwable) {
        resetToPicker(current = current, banner = t.message ?: "Sign-in failed.")
      } finally {
        pendingVerifier = null
        pendingState = null
      }
    }
  }

  /** Callback from the OS-URL handler (iOS `.onOpenURL`, Android redirect activity). */
  fun onExternalCallback(callbackUrl: String) {
    // Currently the browser launcher handles its own callback wait; this hook
    // exists so platform layers that cannot complete the wait themselves
    // (e.g. if iOS ever drops `ASWebAuthenticationSession`) can forward.
    // On today's platforms the channel is internal to BrowserLauncher.
    // No-op by default. Kept in the public surface so platform code has
    // a single, documented entry point that won't change shape later.
    @Suppress("UNUSED_PARAMETER") val ignored = callbackUrl
  }

  fun dismissBanner() {
    val current = _uiState.value as? LoginUiState.ServerPicker ?: return
    _uiState.value = current.copy(banner = null)
  }

  private fun resetToPicker(current: LoginUiState.ServerPicker, banner: String) {
    _uiState.value =
      LoginUiState.ServerPicker(input = current.input, probe = current.probe, banner = banner)
  }

  companion object {
    const val DEFAULT_CLIENT_ID: String = "horologia-mobile"
    const val DEFAULT_DEBOUNCE_MILLIS: Long = 600L
    const val DEFAULT_FINISHING_MIN_DWELL_MILLIS: Long = 300L
  }
}
