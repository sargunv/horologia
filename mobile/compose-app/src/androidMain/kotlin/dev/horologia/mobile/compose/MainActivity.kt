package dev.horologia.mobile.compose

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import dev.horologia.mobile.compose.platform.AndroidBrowserLauncherImpl
import dev.horologia.mobile.core.AppContainer
import dev.horologia.mobile.core.configureHorologiaApi
import dev.horologia.mobile.core.platform.BrowserLauncher

class MainActivity : ComponentActivity() {
  private val appContainer by lazy { AppContainer(context = applicationContext) }

  override fun onCreate(savedInstanceState: Bundle?) {
    super.onCreate(savedInstanceState)

    BrowserLauncher.install(launcher = AndroidBrowserLauncherImpl(context = applicationContext))

    // Configure the Api singleton with a placeholder so any call before the boot router
    // resolves doesn't NPE on a missing base URL. `HorologiaApp` re-configures with the
    // resolved URL inside a `LaunchedEffect`.
    configureHorologiaApi(
      baseUrl = FALLBACK_BASE_URL,
      getToken = { appContainer.sessionHolder.currentAccessToken() },
    )

    setContent { HorologiaApp(appContainer = appContainer) }
  }

  private companion object {
    const val FALLBACK_BASE_URL = "https://horologia.invalid/api/"
  }
}
