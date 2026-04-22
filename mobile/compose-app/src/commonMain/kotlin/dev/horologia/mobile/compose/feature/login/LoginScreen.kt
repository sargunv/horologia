package dev.horologia.mobile.compose.feature.login

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.IconButton
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.semantics.LiveRegionMode
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.liveRegion
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardCapitalization
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import dev.horologia.mobile.compose.LocalAppContainer
import dev.horologia.mobile.core.feature.login.LoginUiState
import dev.horologia.mobile.core.feature.login.LoginViewModel
import dev.horologia.mobile.core.feature.login.ProbeState

/**
 * ServerPicker + Launching + Finishing composite. Branches Compact (< 600 dp) vs. Expanded (>= 600
 * dp) per R2 using [BoxWithConstraints] and the received max-width, instead of `WindowSizeClass` —
 * CMP 1.9.0's window-size-class artifact isn't on the classpath; the measured width serves the same
 * role.
 *
 * Parent nav-host should call [onComplete] on the `Complete` transition to pop this destination and
 * navigate to Profile.
 */
@Composable
fun LoginScreen(
  onComplete: () -> Unit,
  initialServerUrl: String? = null,
  initialBanner: String? = null,
) {
  val viewModel: LoginViewModel =
    viewModel(factory = LocalAppContainer.current.loginViewModelFactory)
  val state by viewModel.uiState.collectAsStateWithLifecycle()

  LaunchedEffect(initialServerUrl) {
    if (!initialServerUrl.isNullOrEmpty()) viewModel.seedInitialUrl(url = initialServerUrl)
  }
  LaunchedEffect(initialBanner) {
    if (!initialBanner.isNullOrEmpty()) viewModel.showBanner(message = initialBanner)
  }
  LaunchedEffect(state) { if (state is LoginUiState.Complete) onComplete() }

  BoxWithConstraints(modifier = Modifier.fillMaxSize()) {
    val isExpanded = maxWidth >= 600.dp
    if (isExpanded) {
      ExpandedLoginLayout(
        state = state,
        onUrlChanged = viewModel::onUrlChanged,
        onSubmit = viewModel::onSubmit,
        onDismissBanner = viewModel::dismissBanner,
        onCancelSignIn = viewModel::cancelSignIn,
      )
    } else {
      CompactLoginLayout(
        state = state,
        onUrlChanged = viewModel::onUrlChanged,
        onSubmit = viewModel::onSubmit,
        onDismissBanner = viewModel::dismissBanner,
        onCancelSignIn = viewModel::cancelSignIn,
      )
    }
  }
}

@Composable
private fun CompactLoginLayout(
  state: LoginUiState,
  onUrlChanged: (String) -> Unit,
  onSubmit: () -> Unit,
  onDismissBanner: () -> Unit,
  onCancelSignIn: () -> Unit,
) {
  Column(
    modifier = Modifier.fillMaxSize().padding(24.dp),
    verticalArrangement = Arrangement.Center,
    horizontalAlignment = Alignment.Start,
  ) {
    LoginHeadline()
    Spacer(Modifier.height(24.dp))
    LoginBody(
      state = state,
      onUrlChanged = onUrlChanged,
      onSubmit = onSubmit,
      onDismissBanner = onDismissBanner,
      onCancelSignIn = onCancelSignIn,
    )
  }
}

@Composable
private fun ExpandedLoginLayout(
  state: LoginUiState,
  onUrlChanged: (String) -> Unit,
  onSubmit: () -> Unit,
  onDismissBanner: () -> Unit,
  onCancelSignIn: () -> Unit,
) {
  Column(
    modifier = Modifier.fillMaxSize().padding(32.dp),
    verticalArrangement = Arrangement.Center,
    horizontalAlignment = Alignment.CenterHorizontally,
  ) {
    Card(
      modifier = Modifier.widthIn(max = 480.dp).fillMaxWidth(),
      shape = RoundedCornerShape(28.dp),
    ) {
      Column(modifier = Modifier.padding(32.dp)) {
        LoginHeadline()
        Spacer(Modifier.height(24.dp))
        LoginBody(
          state = state,
          onUrlChanged = onUrlChanged,
          onSubmit = onSubmit,
          onDismissBanner = onDismissBanner,
          onCancelSignIn = onCancelSignIn,
        )
      }
    }
  }
}

@Composable
private fun LoginHeadline() {
  Text(
    "Connect to Horologia",
    style = MaterialTheme.typography.headlineMedium,
    fontWeight = FontWeight.Bold,
  )
  Spacer(Modifier.height(8.dp))
  Text(
    "Paste your server URL to sign in.",
    style = MaterialTheme.typography.bodyMedium,
    color = MaterialTheme.colorScheme.onSurfaceVariant,
  )
}

@Composable
private fun LoginBody(
  state: LoginUiState,
  onUrlChanged: (String) -> Unit,
  onSubmit: () -> Unit,
  onDismissBanner: () -> Unit,
  onCancelSignIn: () -> Unit,
) {
  when (state) {
    is LoginUiState.ServerPicker ->
      ServerPickerBody(
        state = state,
        onUrlChanged = onUrlChanged,
        onSubmit = onSubmit,
        onDismissBanner = onDismissBanner,
      )
    is LoginUiState.LaunchingBrowser ->
      StatusBody(title = "Opening sign-in…", detail = state.input, onCancel = onCancelSignIn)
    is LoginUiState.Finishing ->
      StatusBody(title = "Finishing sign-in…", detail = state.input, onCancel = onCancelSignIn)
    is LoginUiState.Complete -> StatusBody(title = "Signed in.", detail = null, onCancel = null)
  }
}

@Composable
private fun ServerPickerBody(
  state: LoginUiState.ServerPicker,
  onUrlChanged: (String) -> Unit,
  onSubmit: () -> Unit,
  onDismissBanner: () -> Unit,
) {
  val banner = state.banner
  if (banner != null) {
    Row(
      modifier = Modifier.fillMaxWidth().padding(bottom = 12.dp),
      verticalAlignment = Alignment.CenterVertically,
    ) {
      Text(
        banner,
        style = MaterialTheme.typography.bodyMedium,
        color = MaterialTheme.colorScheme.error,
        modifier = Modifier.weight(1f).semantics { liveRegion = LiveRegionMode.Assertive },
      )
      IconButton(
        onClick = onDismissBanner,
        modifier = Modifier.semantics { contentDescription = "Dismiss" },
      ) {
        Text("×", style = MaterialTheme.typography.titleLarge)
      }
    }
  }
  val focusRequester = remember { FocusRequester() }
  LaunchedEffect(Unit) { focusRequester.requestFocus() }
  OutlinedTextField(
    value = state.input,
    onValueChange = onUrlChanged,
    label = { Text("Server URL") },
    placeholder = { Text("tasks.example.com") },
    singleLine = true,
    isError =
      state.probe is ProbeState.InvalidUnreachable || state.probe is ProbeState.InvalidWrongServer,
    supportingText = { ProbeSupportingText(probe = state.probe) },
    keyboardOptions =
      KeyboardOptions(
        keyboardType = KeyboardType.Uri,
        autoCorrectEnabled = false,
        capitalization = KeyboardCapitalization.None,
        imeAction = ImeAction.Go,
      ),
    keyboardActions = KeyboardActions(onGo = { if (state.probe is ProbeState.Valid) onSubmit() }),
    modifier = Modifier.fillMaxWidth().focusRequester(focusRequester),
  )
  Spacer(Modifier.height(16.dp))
  Button(
    onClick = onSubmit,
    enabled = state.probe is ProbeState.Valid,
    modifier = Modifier.fillMaxWidth(),
  ) {
    Text("Continue")
  }
}

@Composable
private fun ProbeSupportingText(probe: ProbeState) {
  val message =
    when (probe) {
      ProbeState.Empty -> null
      ProbeState.Typing -> null
      ProbeState.Probing -> "Checking server…"
      is ProbeState.Valid -> "Horologia server detected."
      is ProbeState.InvalidUnreachable -> "Can't reach ${probe.host}."
      ProbeState.InvalidWrongServer -> "Not a Horologia server."
    }
  if (message != null) Text(message)
}

@Composable
private fun StatusBody(title: String, detail: String?, onCancel: (() -> Unit)?) {
  Text(title, style = MaterialTheme.typography.titleMedium)
  if (detail != null) {
    Spacer(Modifier.height(8.dp))
    Text(
      detail,
      style = MaterialTheme.typography.bodyMedium,
      color = MaterialTheme.colorScheme.onSurfaceVariant,
    )
  }
  Spacer(Modifier.height(16.dp))
  LinearProgressIndicator(
    modifier = Modifier.fillMaxWidth().semantics { contentDescription = title }
  )
  if (onCancel != null) {
    Spacer(Modifier.height(12.dp))
    // Desktop has no reliable "user closed the browser tab" signal, so this is the primary
    // escape hatch there. On Android/iOS this backstops the native cancel paths (Custom Tabs
    // back gesture, ASWebAuthenticationSession dismissal).
    TextButton(onClick = onCancel) { Text("Cancel") }
  }
}

@Suppress("unused")
@Composable
private fun CheckingServerProgress() {
  CircularProgressIndicator(
    modifier = Modifier.semantics { contentDescription = "Checking server" }
  )
}
