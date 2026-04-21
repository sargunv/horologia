package dev.horologia.mobile.compose

import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import dev.horologia.mobile.compose.nav.ProfileRoute
import dev.horologia.mobile.compose.nav.SpacesRoute
import dev.horologia.mobile.core.AppContainer

/**
 * Composition root for the Compose app. Installs [LocalAppContainer] once and hosts the [NavHost]
 * that wires destinations to the screens. Screens receive narrow `on<Action>` lambdas, never
 * [androidx.navigation.NavController] itself — all navigation edits stay localized here.
 */
@Composable
fun HorologiaApp(appContainer: AppContainer) {
  CompositionLocalProvider(LocalAppContainer provides appContainer) {
    MaterialTheme {
      Surface {
        val navController = rememberNavController()
        NavHost(navController = navController, startDestination = ProfileRoute) {
          composable<ProfileRoute> {
            ProfileScreen(onOpenSpaces = { navController.navigate(SpacesRoute) })
          }
          composable<SpacesRoute> { SpacesScreen(onBack = { navController.popBackStack() }) }
        }
      }
    }
  }
}
