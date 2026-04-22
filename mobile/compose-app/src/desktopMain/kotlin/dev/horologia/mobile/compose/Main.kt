package dev.horologia.mobile.compose

import androidx.compose.ui.window.Window
import androidx.compose.ui.window.application
import dev.horologia.mobile.compose.nav.LoginRoute
import dev.horologia.mobile.compose.nav.ProfileRoute
import dev.horologia.mobile.core.AppContainer
import dev.horologia.mobile.core.configureHorologiaApi
import dev.horologia.mobile.core.feature.login.BootDestination
import kotlinx.coroutines.runBlocking

fun main() {
  val appContainer = AppContainer()

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

  application {
    Window(onCloseRequest = ::exitApplication, title = "Horologia") {
      HorologiaApp(
        appContainer = appContainer,
        startDestination = start,
        initialServerUrl = initialUrl,
        initialBanner = initialBanner,
      )
    }
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
  return if (trimmed.endsWith("/api") || trimmed.contains("/api/")) "$trimmed/" else "$trimmed/api/"
}

// Placeholder base URL for pre-login Api singleton init. LoginViewModel rewrites it.
private const val FALLBACK_BASE_URL = "https://horologia.invalid/api/"
