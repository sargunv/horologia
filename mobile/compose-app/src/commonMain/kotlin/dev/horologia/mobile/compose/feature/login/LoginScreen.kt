package dev.horologia.mobile.compose.feature.login

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
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
      )
    } else {
      CompactLoginLayout(
        state = state,
        onUrlChanged = viewModel::onUrlChanged,
        onSubmit = viewModel::onSubmit,
      )
    }
  }
}

@Composable
private fun CompactLoginLayout(
  state: LoginUiState,
  onUrlChanged: (String) -> Unit,
  onSubmit: () -> Unit,
) {
  Column(
    modifier = Modifier.fillMaxSize().padding(24.dp),
    verticalArrangement = Arrangement.Center,
    horizontalAlignment = Alignment.Start,
  ) {
    LoginHeadline()
    Spacer(Modifier.height(24.dp))
    LoginBody(state = state, onUrlChanged = onUrlChanged, onSubmit = onSubmit)
  }
}

@Composable
private fun ExpandedLoginLayout(
  state: LoginUiState,
  onUrlChanged: (String) -> Unit,
  onSubmit: () -> Unit,
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
        LoginBody(state = state, onUrlChanged = onUrlChanged, onSubmit = onSubmit)
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
private fun LoginBody(state: LoginUiState, onUrlChanged: (String) -> Unit, onSubmit: () -> Unit) {
  when (state) {
    is LoginUiState.ServerPicker ->
      ServerPickerBody(state = state, onUrlChanged = onUrlChanged, onSubmit = onSubmit)
    is LoginUiState.LaunchingBrowser ->
      StatusBody(title = "Opening secure sign-in…", detail = state.input)
    is LoginUiState.Finishing -> StatusBody(title = "Finishing sign-in…", detail = state.input)
    is LoginUiState.Complete -> StatusBody(title = "Signed in.", detail = null)
  }
}

@Composable
private fun ServerPickerBody(
  state: LoginUiState.ServerPicker,
  onUrlChanged: (String) -> Unit,
  onSubmit: () -> Unit,
) {
  val banner = state.banner
  if (banner != null) {
    Text(
      banner,
      style = MaterialTheme.typography.bodyMedium,
      color = MaterialTheme.colorScheme.error,
      modifier = Modifier.padding(bottom = 12.dp),
    )
  }
  OutlinedTextField(
    value = state.input,
    onValueChange = onUrlChanged,
    label = { Text("Server URL") },
    placeholder = { Text("tasks.example.com") },
    singleLine = true,
    isError =
      state.probe is ProbeState.InvalidUnreachable || state.probe is ProbeState.InvalidWrongServer,
    supportingText = { ProbeSupportingText(probe = state.probe) },
    modifier = Modifier.fillMaxWidth(),
  )
  if (state.probe is ProbeState.Probing) {
    LinearProgressIndicator(modifier = Modifier.fillMaxWidth().padding(top = 12.dp))
  }
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
      ProbeState.Valid -> "Horologia server detected."
      is ProbeState.InvalidUnreachable -> "Can't reach ${probe.host}."
      ProbeState.InvalidWrongServer -> "Not a Horologia server."
    }
  if (message != null) Text(message)
}

@Composable
private fun StatusBody(title: String, detail: String?) {
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
  LinearProgressIndicator(modifier = Modifier.fillMaxWidth())
}
