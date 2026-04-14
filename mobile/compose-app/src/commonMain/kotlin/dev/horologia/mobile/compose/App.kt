package dev.horologia.mobile.compose

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp

@Composable
fun HorologiaApp() {
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
        Text(
          text = "Compose bootstrap running on ${platformName()}",
          style = MaterialTheme.typography.titleMedium,
          modifier = Modifier.padding(top = 12.dp),
        )
        Text(
          text = "Shared KMP core is wired in. Platform apps stay thin on top.",
          style = MaterialTheme.typography.bodyLarge,
          modifier = Modifier.padding(top = 8.dp),
        )
      }
    }
  }
}
