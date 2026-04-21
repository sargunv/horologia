package dev.horologia.mobile.compose

import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.remember
import androidx.compose.ui.window.Window
import androidx.compose.ui.window.application
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.ViewModelStore
import dev.horologia.mobile.core.AppContainer
import dev.horologia.mobile.core.screen.profile.ProfileViewModel

fun main() {
  // TODO: Replace dev-mode bearer token with real auth flow:
  //   POST /app/auth/login → cookie session → POST /api/auth/tokens (bearer token).
  val baseUrl = System.getenv("HOROLOGIA_BASE_URL") ?: "http://localhost:8080/api/"
  val devToken = System.getenv("HOROLOGIA_DEV_TOKEN")
  val appContainer = AppContainer(baseUrl = baseUrl, getToken = { devToken })

  application {
    Window(onCloseRequest = ::exitApplication, title = "Horologia") {
      val viewModelStore = remember { ViewModelStore() }
      DisposableEffect(viewModelStore) { onDispose { viewModelStore.clear() } }
      val viewModel =
        remember(viewModelStore) {
          ViewModelProvider.create(viewModelStore, appContainer.profileViewModelFactory)[
              ProfileViewModel::class]
        }
      ProfileScreen(viewModel)
    }
  }
}
