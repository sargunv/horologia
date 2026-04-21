package dev.horologia.mobile.compose

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.LiveRegionMode
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.liveRegion
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import dev.horologia.mobile.core.feature.profile.ProfileUiState
import dev.horologia.mobile.core.feature.profile.ProfileViewModel

@Composable
fun ProfileScreen(onOpenSpaces: () -> Unit) {
  val viewModel: ProfileViewModel =
    viewModel(factory = LocalAppContainer.current.profileViewModelFactory)
  val state by viewModel.uiState.collectAsStateWithLifecycle()

  Column(
    modifier = Modifier.fillMaxSize().padding(24.dp),
    verticalArrangement = Arrangement.Center,
    horizontalAlignment = Alignment.Start,
  ) {
    Text("Horologia", style = MaterialTheme.typography.displaySmall, fontWeight = FontWeight.Bold)

    when (val current = state) {
      is ProfileUiState.Loading ->
        CircularProgressIndicator(
          modifier =
            Modifier.padding(top = 24.dp).semantics { contentDescription = "Loading profile" }
        )

      is ProfileUiState.Success ->
        Text(
          text = "Signed in as ${current.displayName}",
          style = MaterialTheme.typography.titleMedium,
          modifier = Modifier.padding(top = 24.dp).semantics { liveRegion = LiveRegionMode.Polite },
        )

      is ProfileUiState.Error -> {
        Text(
          text = current.message,
          style = MaterialTheme.typography.bodyLarge,
          color = MaterialTheme.colorScheme.error,
          modifier = Modifier.padding(top = 24.dp).semantics { liveRegion = LiveRegionMode.Polite },
        )
        if (current.retryable) {
          Button(onClick = viewModel::refresh, modifier = Modifier.padding(top = 12.dp)) {
            Text("Retry")
          }
        }
      }
    }

    Button(onClick = onOpenSpaces, modifier = Modifier.padding(top = 24.dp)) { Text("View spaces") }
  }
}
