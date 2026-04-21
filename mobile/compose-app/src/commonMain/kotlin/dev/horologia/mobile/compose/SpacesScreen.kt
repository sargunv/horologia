package dev.horologia.mobile.compose

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
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
import dev.horologia.mobile.core.feature.spaces.SpacesUiState
import dev.horologia.mobile.core.feature.spaces.SpacesViewModel

@Composable
fun SpacesScreen(onBack: () -> Unit) {
  val viewModel: SpacesViewModel =
    viewModel(factory = LocalAppContainer.current.spacesViewModelFactory)
  val state by viewModel.uiState.collectAsStateWithLifecycle()

  Column(modifier = Modifier.fillMaxSize().padding(24.dp), verticalArrangement = Arrangement.Top) {
    Row(verticalAlignment = Alignment.CenterVertically) {
      TextButton(
        onClick = onBack,
        // Wrap the glyph+label so screen readers announce "Back" instead of reading
        // the arrow character literally ("left arrow, Back").
        modifier = Modifier.semantics(mergeDescendants = true) { contentDescription = "Back" },
      ) {
        Text("← Back")
      }
      Spacer(Modifier.width(8.dp))
      Text("Spaces", style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.Bold)
    }

    when (val current = state) {
      is SpacesUiState.Loading ->
        CircularProgressIndicator(
          modifier =
            Modifier.padding(top = 24.dp).semantics { contentDescription = "Loading spaces" }
        )

      is SpacesUiState.Success ->
        LazyColumn(modifier = Modifier.fillMaxSize().padding(top = 16.dp)) {
          items(current.items, key = { it.slug }) { item ->
            Text(
              text = item.name,
              style = MaterialTheme.typography.bodyLarge,
              modifier = Modifier.padding(vertical = 12.dp),
            )
          }
        }

      is SpacesUiState.Error -> {
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
  }
}
