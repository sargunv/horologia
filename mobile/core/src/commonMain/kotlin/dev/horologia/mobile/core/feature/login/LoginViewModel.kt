package dev.horologia.mobile.core.feature.login

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dev.horologia.mobile.core.platform.BrowserCancelledException
import dev.horologia.mobile.core.platform.BrowserFailedException
import dev.horologia.mobile.core.platform.BrowserLauncher
import dev.horologia.mobile.core.platform.platformLog
import dev.horologia.mobile.core.session.ServerPrefs
import dev.horologia.mobile.core.session.SessionHolder
import dev.horologia.mobile.core.session.StoredSession
import io.ktor.http.Url
import kotlin.time.Clock
import kotlin.time.ExperimentalTime
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineExceptionHandler
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

/**
 * Whole-flow owner for the login surface: server-picker debounced probe, PKCE + state generation,
 * browser handoff, token exchange, min-dwell finishing state. Mirrors `feature/profile`'s single-VM
 * template byte-for-byte on structure (StateFlow<UiState>, cancel-safe Job fields).
 *
 * PKCE verifier and OAuth state are VM-private fields, cleared in every terminal path (success,
 * cancellation, state mismatch, exception) — per R6 they must never hit disk.
 */
@OptIn(ExperimentalTime::class)
class LoginViewModel
internal constructor(
  private val gateway: LoginGateway,
  private val browser: BrowserLauncher,
  private val serverPrefs: ServerPrefs,
  private val sessionHolder: SessionHolder,
  private val clientId: String = DEFAULT_CLIENT_ID,
  private val scope: String = DEFAULT_SCOPE,
  private val debounceMillis: Long = DEFAULT_DEBOUNCE_MILLIS,
  private val finishingMinDwellMillis: Long = DEFAULT_FINISHING_MIN_DWELL_MILLIS,
  private val nowMillis: () -> Long = { Clock.System.now().toEpochMilliseconds() },
  private val reconfigureApi: (String) -> Unit = {},
) : ViewModel() {
  private val _uiState =
    MutableStateFlow<LoginUiState>(LoginUiState.ServerPicker(input = "", probe = ProbeState.Empty))
  val uiState: StateFlow<LoginUiState> = _uiState.asStateFlow()

  private var probeJob: Job? = null
  private var flowJob: Job? = null

  // Kotlin/Native's default CoroutineExceptionHandler aborts the process with SIGABRT on any
  // uncaught exception in a `launch` child (even under SupervisorJob). [safeLaunch] centralizes
  // the handler so a new `launch` callsite can't accidentally omit it and reintroduce the abort
  // hazard. The body-level try/catch in `onSubmit` is still the primary path; this handler is
  // the last-ditch safety net for throws that slip past a finally block or a StateFlow emission.
  private val errorHandler = CoroutineExceptionHandler { _, t ->
    if (t !is CancellationException) {
      platformLog("LoginViewModel", "uncaught ${t::class.simpleName}: ${t.message}")
    }
  }

  private fun safeLaunch(block: suspend CoroutineScope.() -> Unit): Job =
    viewModelScope.launch(errorHandler, block = block)

  // Cleared at every terminal transition.
  private var pendingVerifier: String? = null
  private var pendingState: String? = null
  private var pendingNormalizedUrl: String? = null

  /** Called by the cold-launch router when a saved server URL should be pre-filled. */
  fun seedInitialUrl(url: String) {
    val current = _uiState.value as? LoginUiState.ServerPicker ?: return
    if (current.input.isNotEmpty()) return
    _uiState.value = current.copy(input = url, probe = ProbeState.Probing)
    probeJob?.cancel()
    probeJob = safeLaunch { runProbe(url = url) }
  }

  /** Called after a cold-launch refresh failure. */
  fun showBanner(message: String) {
    val current = _uiState.value as? LoginUiState.ServerPicker ?: return
    _uiState.value = current.copy(banner = message)
  }

  /**
   * Abort the in-flight OAuth trip and return to the server picker. Called from the Cancel button
   * rendered on the [LoginUiState.LaunchingBrowser] and [LoginUiState.Finishing] screens — the
   * user's primary escape hatch when the browser hangs or they change their mind. On desktop
   * there's no "browser tab closed" signal, so this is the only cancel path.
   */
  fun cancelSignIn() {
    val current = _uiState.value
    if (current !is LoginUiState.LaunchingBrowser && current !is LoginUiState.Finishing) return
    flowJob?.cancel()
    flowJob = null
    pendingVerifier = null
    pendingState = null
    pendingNormalizedUrl = null
    val input =
      when (current) {
        is LoginUiState.LaunchingBrowser -> current.input
        is LoginUiState.Finishing -> current.input
      }
    val probe = if (input.isEmpty()) ProbeState.Empty else ProbeState.Typing
    _uiState.value =
      LoginUiState.ServerPicker(input = input, probe = probe, banner = "Sign-in cancelled.")
    // Re-probe so the Continue button becomes live again without the user touching the field.
    if (input.isNotEmpty()) debounceAndProbe(url = input)
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
    probeJob = safeLaunch {
      delay(debounceMillis)
      runProbe(url = url)
    }
  }

  private suspend fun runProbe(url: String) {
    val current = _uiState.value as? LoginUiState.ServerPicker ?: return
    if (current.input != url) return
    val candidates = UrlNormalizer.candidates(url)
    if (candidates.isEmpty()) {
      _uiState.value = current.copy(probe = ProbeState.InvalidWrongServer)
      return
    }
    _uiState.value = current.copy(probe = ProbeState.Probing)

    // Try candidates in order. `Ok` and `WrongServer` are definitive — we stop. `Unreachable`
    // means we fall through to the next candidate (the `http://` fallback for bare hosts). If
    // every candidate is unreachable, surface the first candidate's host in the error state so
    // the UI copy doesn't flicker.
    var lastUnreachable: ProbeResult.Unreachable? = null
    var finalState: ProbeState? = null
    for (candidate in candidates) {
      val result = gateway.probeServer(baseUrl = candidate)
      val live = _uiState.value as? LoginUiState.ServerPicker ?: return
      if (live.input != url) return
      when (result) {
        ProbeResult.Ok -> {
          finalState = ProbeState.Valid(resolvedUrl = candidate)
          break
        }
        ProbeResult.WrongServer -> {
          finalState = ProbeState.InvalidWrongServer
          break
        }
        is ProbeResult.Unreachable -> {
          if (lastUnreachable == null) lastUnreachable = result
        }
      }
    }
    val live = _uiState.value as? LoginUiState.ServerPicker ?: return
    if (live.input != url) return
    _uiState.value =
      live.copy(probe = finalState ?: ProbeState.InvalidUnreachable(host = lastUnreachable!!.host))
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
    val valid = current.probe as? ProbeState.Valid ?: return
    // Use the URL the probe actually reached — scheme auto-detection may have fallen back to
    // `http://` for a bare host. Re-normalizing the raw input would lose that resolution.
    val normalized = valid.resolvedUrl

    probeJob?.cancel()
    flowJob?.cancel()
    flowJob = safeLaunch {
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
            scope = scope,
          )

        _uiState.value = LoginUiState.LaunchingBrowser(input = current.input)
        val callbackUrl =
          try {
            browser.launchAndAwait(authorizeUrl = authUrl)
          } catch (e: BrowserCancelledException) {
            resetToPicker(current = current, banner = e.message ?: "Sign-in cancelled.")
            return@safeLaunch
          } catch (e: BrowserFailedException) {
            resetToPicker(current = current, banner = e.message ?: "Sign-in failed. Try again.")
            return@safeLaunch
          }

        val parsed = runCatching { Url(callbackUrl) }.getOrNull()
        if (parsed == null) {
          resetToPicker(current = current, banner = "Sign-in failed. Try again.")
          return@safeLaunch
        }
        val params = parsed.parameters
        val returnedState = params["state"]
        if (returnedState != state) {
          resetToPicker(current = current, banner = "Sign-in failed. Try again.")
          return@safeLaunch
        }
        val errorParam = params["error"]
        if (errorParam != null) {
          val banner =
            if (errorParam == "access_denied") "Sign-in cancelled." else "Sign-in failed."
          resetToPicker(current = current, banner = banner)
          return@safeLaunch
        }
        val code = params["code"]
        if (code.isNullOrEmpty()) {
          resetToPicker(current = current, banner = "Sign-in failed.")
          return@safeLaunch
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
            reconfigureApi(normalized)
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
        platformLog("LoginViewModel", "onSubmit threw ${t::class.simpleName}: ${t.message}")
        resetToPicker(current = current, banner = t.message ?: "Sign-in failed.")
      } finally {
        pendingVerifier = null
        pendingState = null
        pendingNormalizedUrl = null
      }
    }
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
    // Mirrors cli/internal/runtime/oauth.go DefaultOAuthScopes — the mobile client requests the
    // same full scope set so profile/spaces/tasks surfaces work without follow-up re-auth.
    const val DEFAULT_SCOPE: String =
      "activity:read admin:read admin:write profile:read spaces:read spaces:write " +
        "tags:read tags:write tasks:read tasks:write users:read users:write"
    const val DEFAULT_DEBOUNCE_MILLIS: Long = 600L
    const val DEFAULT_FINISHING_MIN_DWELL_MILLIS: Long = 300L
  }
}
