package dev.horologia.mobile.compose

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Surface
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import dev.horologia.mobile.compose.feature.login.LoginScreen
import dev.horologia.mobile.compose.nav.LoginRoute
import dev.horologia.mobile.compose.nav.ProfileRoute
import dev.horologia.mobile.compose.nav.SpacesRoute
import dev.horologia.mobile.compose.theme.HorologiaTheme
import dev.horologia.mobile.core.AppContainer
import dev.horologia.mobile.core.configureHorologiaApi
import dev.horologia.mobile.core.ensureApiPath
import dev.horologia.mobile.core.feature.login.BootDestination

/**
 * Composition root for the Compose app. Installs [LocalAppContainer] once and hosts the [NavHost]
 * that wires destinations to the screens. Screens receive narrow `on<Action>` lambdas, never
 * [androidx.navigation.NavController] itself — all navigation edits stay localized here.
 *
 * Cold-launch routing happens in a [LaunchedEffect] on first frame — a neutral splash renders until
 * the boot router resolves, then the NavHost mounts at the decided start destination. This keeps
 * the main thread free during `EncryptedSharedPreferences` / Keychain init (no ANR risk), and
 * matches the iOS shape where SwiftUI renders a `.task { }`-driven splash.
 */
@Composable
fun HorologiaApp(appContainer: AppContainer) {
  var resolved by remember { mutableStateOf<Triple<Any, String?, String?>?>(null) }

  LaunchedEffect(appContainer) {
    val destination = appContainer.bootRouter.decideBootDestination()
    configureHorologiaApi(
      baseUrl = baseUrlFor(destination = destination),
      getToken = { appContainer.sessionHolder.currentAccessToken() },
    )
    resolved = interpret(destination = destination)
  }

  CompositionLocalProvider(LocalAppContainer provides appContainer) {
    HorologiaTheme {
      Surface {
        val current = resolved
        if (current == null) {
          Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
            CircularProgressIndicator(
              modifier = Modifier.semantics { contentDescription = "Loading" }
            )
          }
        } else {
          val (start, initialUrl, initialBanner) = current
          val navController = rememberNavController()
          NavHost(navController = navController, startDestination = start) {
            composable<LoginRoute> {
              LoginScreen(
                onComplete = {
                  navController.navigate(ProfileRoute) { popUpTo(LoginRoute) { inclusive = true } }
                },
                initialServerUrl = initialUrl,
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
}

private fun baseUrlFor(destination: BootDestination): String =
  when (destination) {
    is BootDestination.SignedIn -> ensureApiPath(baseUrl = destination.savedUrl)
    is BootDestination.ServerOnly -> ensureApiPath(baseUrl = destination.savedUrl)
    is BootDestination.SignedOutAfterRefresh -> ensureApiPath(baseUrl = destination.savedUrl)
    is BootDestination.Unconfigured -> "https://horologia.invalid/api/"
  }

private fun interpret(destination: BootDestination): Triple<Any, String?, String?> =
  when (destination) {
    is BootDestination.Unconfigured -> Triple(LoginRoute, null, null)
    is BootDestination.ServerOnly -> Triple(LoginRoute, destination.savedUrl, null)
    is BootDestination.SignedIn -> Triple(ProfileRoute, destination.savedUrl, null)
    is BootDestination.SignedOutAfterRefresh ->
      Triple(LoginRoute, destination.savedUrl, "Signed out.")
  }
