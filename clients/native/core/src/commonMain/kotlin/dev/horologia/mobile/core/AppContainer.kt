package dev.horologia.mobile.core

import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import dev.horologia.mobile.core.feature.login.BootRouter
import dev.horologia.mobile.core.feature.login.LiveLoginGateway
import dev.horologia.mobile.core.feature.login.LoginGateway
import dev.horologia.mobile.core.feature.login.LoginViewModel
import dev.horologia.mobile.core.feature.profile.LiveProfileGateway
import dev.horologia.mobile.core.feature.profile.ProfileGateway
import dev.horologia.mobile.core.feature.profile.ProfileViewModel
import dev.horologia.mobile.core.feature.spaces.LiveSpacesGateway
import dev.horologia.mobile.core.feature.spaces.SpacesGateway
import dev.horologia.mobile.core.feature.spaces.SpacesViewModel
import dev.horologia.mobile.core.platform.BrowserLauncher
import dev.horologia.mobile.core.session.ServerPrefs
import dev.horologia.mobile.core.session.SessionHolder
import dev.horologia.mobile.core.session.SessionStore

/**
 * Holds the gateways and ViewModel factories that back each screen.
 *
 * Each platform entry point constructs the concrete [SessionStore] / [ServerPrefs] /
 * [BrowserLauncher] and hands them in here — the factory graph itself stays in `commonMain`.
 *
 * Construction order is: `configureHorologiaApi(baseUrl) { sessionHolder.currentAccessToken() }`
 * followed by `AppContainer(...)`. Because `configureHorologiaApi` is safe to re-call (see its doc
 * comment), the login flow re-invokes it once the user picks a server.
 */
class AppContainer(
  val sessionStore: SessionStore,
  val serverPrefs: ServerPrefs,
  val browserLauncher: BrowserLauncher,
) {
  val sessionHolder: SessionHolder = SessionHolder(store = sessionStore)

  private val profileGateway: ProfileGateway = LiveProfileGateway()
  private val spacesGateway: SpacesGateway = LiveSpacesGateway()
  private val loginGateway: LoginGateway = LiveLoginGateway()

  val bootRouter: BootRouter =
    BootRouter(
      serverPrefs = serverPrefs,
      sessionHolder = sessionHolder,
      loginGateway = loginGateway,
    )

  val profileViewModelFactory: ViewModelProvider.Factory = viewModelFactory {
    initializer { ProfileViewModel(profileGateway) }
  }

  val spacesViewModelFactory: ViewModelProvider.Factory = viewModelFactory {
    initializer { SpacesViewModel(spacesGateway) }
  }

  val loginViewModelFactory: ViewModelProvider.Factory = viewModelFactory {
    initializer {
      LoginViewModel(
        gateway = loginGateway,
        browser = browserLauncher,
        serverPrefs = serverPrefs,
        sessionHolder = sessionHolder,
        reconfigureApi = { baseUrl ->
          configureHorologiaApi(
            baseUrl = ensureApiPath(baseUrl = baseUrl),
            getToken = { sessionHolder.currentAccessToken() },
          )
        },
      )
    }
  }
}
