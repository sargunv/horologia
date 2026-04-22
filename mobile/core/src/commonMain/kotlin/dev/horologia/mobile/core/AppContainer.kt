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
 * Android's constructor takes a `Context` (see the platform actual) because
 * `EncryptedSharedPreferences` and the Custom-Tabs launcher both need one; desktop and iOS actuals
 * are parameterless. All three share the same post-construct property layout so that `commonMain`
 * consumers — `HorologiaApp`, `MainActivity` — can program against one API.
 *
 * Construction order is: `configureHorologiaApi(baseUrl) { sessionHolder.currentAccessToken() }`
 * followed by `AppContainer(...)`. Because `configureHorologiaApi` is safe to re-call (see its doc
 * comment), the login flow re-invokes it once the user picks a server.
 */
expect class AppContainer {
  val sessionStore: SessionStore
  val serverPrefs: ServerPrefs
  val sessionHolder: SessionHolder
  val browserLauncher: BrowserLauncher
  val bootRouter: BootRouter
  val profileViewModelFactory: ViewModelProvider.Factory
  val spacesViewModelFactory: ViewModelProvider.Factory
  val loginViewModelFactory: ViewModelProvider.Factory
}

/**
 * Shared wiring for the factories + gateways. Platforms build their own [SessionStore] /
 * [ServerPrefs] / [BrowserLauncher] and pass them in here. This indirection keeps the factory graph
 * in commonMain instead of duplicating it across every actual.
 */
internal class AppContainerCore(
  val sessionStore: SessionStore,
  val serverPrefs: ServerPrefs,
  val browserLauncher: BrowserLauncher,
) {
  val sessionHolder: SessionHolder = SessionHolder(store = sessionStore)
  val bootRouter: BootRouter = BootRouter(serverPrefs = serverPrefs, sessionHolder = sessionHolder)

  private val profileGateway: ProfileGateway = LiveProfileGateway()
  private val spacesGateway: SpacesGateway = LiveSpacesGateway()
  private val loginGateway: LoginGateway = LiveLoginGateway()

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
        browserLauncher = browserLauncher,
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
