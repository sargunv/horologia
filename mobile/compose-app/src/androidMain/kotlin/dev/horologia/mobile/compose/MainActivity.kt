package dev.horologia.mobile.compose

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.lifecycle.viewmodel.compose.viewModel
import dev.horologia.mobile.core.AppContainer
import dev.horologia.mobile.core.screen.profile.ProfileViewModel

class MainActivity : ComponentActivity() {
  // TODO: Replace dev-mode bearer token with real auth flow:
  //   POST /app/auth/login → cookie session → POST /api/auth/tokens (bearer token).
  private val appContainer by lazy {
    AppContainer(
      baseUrl = BuildConfig.HOROLOGIA_BASE_URL,
      getToken = { BuildConfig.HOROLOGIA_DEV_TOKEN.ifBlank { null } },
    )
  }

  override fun onCreate(savedInstanceState: Bundle?) {
    super.onCreate(savedInstanceState)
    setContent {
      val viewModel: ProfileViewModel = viewModel(factory = appContainer.profileViewModelFactory)
      ProfileScreen(viewModel)
    }
  }
}
