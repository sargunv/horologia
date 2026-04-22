package dev.horologia.mobile.compose

import androidx.compose.material3.Surface
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import dev.horologia.mobile.compose.feature.login.LoginScreen
import dev.horologia.mobile.compose.nav.LoginRoute
import dev.horologia.mobile.compose.nav.ProfileRoute
import dev.horologia.mobile.compose.nav.SpacesRoute
import dev.horologia.mobile.compose.theme.HorologiaTheme
import dev.horologia.mobile.core.AppContainer

/**
 * Composition root for the Compose app. Installs [LocalAppContainer] once and hosts the [NavHost]
 * that wires destinations to the screens. Screens receive narrow `on<Action>` lambdas, never
 * [androidx.navigation.NavController] itself — all navigation edits stay localized here.
 *
 * [startDestination] is decided by the platform entry point ahead of first frame using
 * `appContainer.bootRouter.decideBootDestination()`. That lets cold-launch routing land the user
 * directly on Profile without a flicker when tokens are present.
 */
@Composable
fun HorologiaApp(
  appContainer: AppContainer,
  startDestination: Any = LoginRoute,
  initialServerUrl: String? = null,
  initialBanner: String? = null,
) {
  CompositionLocalProvider(LocalAppContainer provides appContainer) {
    HorologiaTheme {
      Surface {
        val navController = rememberNavController()
        NavHost(navController = navController, startDestination = startDestination) {
          composable<LoginRoute> {
            LoginScreen(
              onComplete = {
                navController.navigate(ProfileRoute) { popUpTo(LoginRoute) { inclusive = true } }
              },
              initialServerUrl = initialServerUrl,
              initialBanner = initialBanner,
            )
          }
          composable<ProfileRoute> {
            ProfileScreen(onOpenSpaces = { navController.navigate(SpacesRoute) })
          }
          composable<SpacesRoute> { SpacesScreen(onBack = { navController.popBackStack() }) }
        }
      }
    }
  }
}
