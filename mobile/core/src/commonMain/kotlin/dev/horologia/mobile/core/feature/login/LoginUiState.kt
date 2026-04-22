package dev.horologia.mobile.core.feature.login

/**
 * Discriminated union covering every screen shape the login flow can render.
 *
 * - [ServerPicker] — URL field + probe state + optional banner for returning-from-failure copy.
 * - [LaunchingBrowser] — the `LoginViewModel` has built the authorize URL and handed control to
 *   [dev.horologia.mobile.core.platform.BrowserLauncher]. Screens render an "Opening sign-in"
 *   scrim; no user input is useful here except cancel.
 * - [Finishing] — token exchanged, tokens persisted, minimum dwell enforced per § E of the spec.
 * - [Complete] — terminal; the host should navigate to Profile.
 *
 * All state transitions are driven by [LoginViewModel]; screens are pure functions of state.
 */
sealed interface LoginUiState {
  data class ServerPicker(val input: String, val probe: ProbeState, val banner: String? = null) :
    LoginUiState

  data class LaunchingBrowser(val input: String) : LoginUiState

  data class Finishing(val input: String) : LoginUiState

  data object Complete : LoginUiState
}

/**
 * The six field-state buckets from design spec § C, enumerated so the UI can render supporting
 * text + Continue-enablement with zero guesswork.
 *
 * [InvalidUnreachable] carries the host so copy can say "Can't reach tasks.example.com."
 * [InvalidWrongServer] has no payload — copy is always "Not a Horologia server."
 */
sealed interface ProbeState {
  data object Empty : ProbeState

  data object Typing : ProbeState

  data object Probing : ProbeState

  /**
   * The probe succeeded. [resolvedUrl] is the URL that actually answered — may differ from the raw
   * input when the scheme was auto-detected (e.g. user typed `localhost:8080`, probe fell back to
   * `http://localhost:8080`). `onSubmit` uses this exact URL for the OAuth trip.
   */
  data class Valid(val resolvedUrl: String) : ProbeState

  data class InvalidUnreachable(val host: String) : ProbeState

  data object InvalidWrongServer : ProbeState
}
