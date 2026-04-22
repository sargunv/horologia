package dev.horologia.mobile.compose

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import dev.horologia.mobile.compose.nav.LoginRoute
import dev.horologia.mobile.compose.nav.ProfileRoute
import dev.horologia.mobile.compose.platform.AndroidBrowserLauncherImpl
import dev.horologia.mobile.core.AppContainer
import dev.horologia.mobile.core.configureHorologiaApi
import dev.horologia.mobile.core.feature.login.BootDestination
import dev.horologia.mobile.core.platform.BrowserLauncher
import kotlinx.coroutines.runBlocking

class MainActivity : ComponentActivity() {
  private val appContainer by lazy { AppContainer(context = applicationContext) }

  override fun onCreate(savedInstanceState: Bundle?) {
    super.onCreate(savedInstanceState)

    BrowserLauncher.install(launcher = AndroidBrowserLauncherImpl(context = applicationContext))

    val destination = runBlocking { appContainer.bootRouter.decideBootDestination() }
    configureHorologiaApi(
      baseUrl = baseUrlFor(destination = destination),
      getToken = { appContainer.sessionHolder.currentAccessToken() },
    )

    val start: Any
    val initialUrl: String?
    val initialBanner: String?
    when (destination) {
      is BootDestination.Unconfigured -> {
        start = LoginRoute
        initialUrl = null
        initialBanner = null
      }
      is BootDestination.ServerOnly -> {
        start = LoginRoute
        initialUrl = destination.savedUrl
        initialBanner = null
      }
      is BootDestination.SignedIn -> {
        start = ProfileRoute
        initialUrl = destination.savedUrl
        initialBanner = null
      }
      is BootDestination.SignedOutAfterRefresh -> {
        start = LoginRoute
        initialUrl = destination.savedUrl
        initialBanner = "Signed out."
      }
    }

    setContent {
      HorologiaApp(
        appContainer = appContainer,
        startDestination = start,
        initialServerUrl = initialUrl,
        initialBanner = initialBanner,
      )
    }
  }

  private fun baseUrlFor(destination: BootDestination): String =
    when (destination) {
      is BootDestination.SignedIn -> destination.savedUrl.ensureApiPath()
      is BootDestination.ServerOnly -> destination.savedUrl.ensureApiPath()
      is BootDestination.SignedOutAfterRefresh -> destination.savedUrl.ensureApiPath()
      is BootDestination.Unconfigured -> FALLBACK_BASE_URL
    }

  private fun String.ensureApiPath(): String {
    val trimmed = trimEnd('/')
    return if (trimmed.endsWith("/api") || trimmed.contains("/api/")) "$trimmed/"
    else "$trimmed/api/"
  }

  private companion object {
    // Placeholder base URL for before the user picks a server. The LoginViewModel
    // rewrites Api.baseUrl via configureHorologiaApi once a probe succeeds.
    const val FALLBACK_BASE_URL = "https://horologia.invalid/api/"
  }
}
