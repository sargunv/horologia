package dev.horologia.mobile.compose

import androidx.compose.ui.window.Window
import androidx.compose.ui.window.application
import dev.horologia.mobile.core.AppContainer
import dev.horologia.mobile.core.configureHorologiaApi
import dev.horologia.mobile.core.feature.profile.ProfileViewModel

fun main() {
  // TODO: Replace dev-mode bearer token with real auth flow:
  //   POST /app/auth/login → cookie session → POST /api/auth/tokens (bearer token).
  val baseUrl = System.getenv("HOROLOGIA_BASE_URL") ?: "http://localhost:8080/api/"
  val devToken = System.getenv("HOROLOGIA_DEV_TOKEN")
  configureHorologiaApi(baseUrl = baseUrl, getToken = { devToken })
  val appContainer = AppContainer()

  application {
    Window(onCloseRequest = ::exitApplication, title = "Horologia") {
      val viewModel =
        rememberViewModel(ProfileViewModel::class, appContainer.profileViewModelFactory)
      ProfileScreen(viewModel)
    }
  }
}
