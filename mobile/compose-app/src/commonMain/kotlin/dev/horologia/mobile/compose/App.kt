package dev.horologia.mobile.compose

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import dev.horologia.mobile.core.screen.profile.ProfileUiState
import dev.horologia.mobile.core.screen.profile.ProfileViewModel

@Composable
fun ProfileScreen(viewModel: ProfileViewModel) {
  val state by viewModel.uiState.collectAsState()

  MaterialTheme {
    Surface {
      Column(
        modifier = Modifier.fillMaxSize().padding(24.dp),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.Start,
      ) {
        Text(
          "Horologia",
          style = MaterialTheme.typography.displaySmall,
          fontWeight = FontWeight.Bold,
        )

        when (val current = state) {
          is ProfileUiState.Loading ->
            CircularProgressIndicator(modifier = Modifier.padding(top = 24.dp))

          is ProfileUiState.Success ->
            Text(
              text = "Signed in as ${current.displayName}",
              style = MaterialTheme.typography.titleMedium,
              modifier = Modifier.padding(top = 24.dp),
            )

          is ProfileUiState.Error -> {
            Text(
              text = current.message,
              style = MaterialTheme.typography.bodyLarge,
              color = MaterialTheme.colorScheme.error,
              modifier = Modifier.padding(top = 24.dp),
            )
            if (current.retryable) {
              Button(onClick = viewModel::refresh, modifier = Modifier.padding(top = 12.dp)) {
                Text("Retry")
              }
            }
          }
        }
      }
    }
  }
}
