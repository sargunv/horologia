package dev.horologia.mobile.core

import androidx.lifecycle.ViewModelProvider
import dev.horologia.mobile.core.feature.login.BootRouter
import dev.horologia.mobile.core.platform.BrowserLauncher
import dev.horologia.mobile.core.session.ServerPrefs
import dev.horologia.mobile.core.session.SessionHolder
import dev.horologia.mobile.core.session.SessionStore

actual class AppContainer {
  private val core =
    AppContainerCore(
      sessionStore = SessionStore(),
      serverPrefs = ServerPrefs(),
      browserLauncher = BrowserLauncher(),
    )

  actual val sessionStore: SessionStore
    get() = core.sessionStore

  actual val serverPrefs: ServerPrefs
    get() = core.serverPrefs

  actual val sessionHolder: SessionHolder
    get() = core.sessionHolder

  actual val browserLauncher: BrowserLauncher
    get() = core.browserLauncher

  actual val bootRouter: BootRouter
    get() = core.bootRouter

  actual val profileViewModelFactory: ViewModelProvider.Factory
    get() = core.profileViewModelFactory

  actual val spacesViewModelFactory: ViewModelProvider.Factory
    get() = core.spacesViewModelFactory

  actual val loginViewModelFactory: ViewModelProvider.Factory
    get() = core.loginViewModelFactory
}
