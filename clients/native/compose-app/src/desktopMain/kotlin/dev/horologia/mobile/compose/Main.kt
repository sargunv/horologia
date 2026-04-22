package dev.horologia.mobile.compose

import androidx.compose.ui.window.Window
import androidx.compose.ui.window.application
import dev.horologia.mobile.core.AppContainer
import dev.horologia.mobile.core.configureHorologiaApi
import dev.horologia.mobile.core.platform.DesktopBrowserLauncher
import dev.horologia.mobile.core.session.DesktopServerPrefs
import dev.horologia.mobile.core.session.DesktopSessionStore

fun main() {
  val appContainer =
    AppContainer(
      sessionStore = DesktopSessionStore(),
      serverPrefs = DesktopServerPrefs(),
      browserLauncher = DesktopBrowserLauncher(),
    )

  // Configure the Api singleton with a placeholder so any call before the boot router
  // resolves doesn't NPE on a missing base URL. `HorologiaApp` re-configures with the
  // resolved URL inside a `LaunchedEffect`.
  configureHorologiaApi(
    baseUrl = FALLBACK_BASE_URL,
    getToken = { appContainer.sessionHolder.currentAccessToken() },
  )

  application {
    Window(onCloseRequest = ::exitApplication, title = "Horologia") {
      HorologiaApp(appContainer = appContainer)
    }
  }
}

private const val FALLBACK_BASE_URL = "https://horologia.invalid/api/"
