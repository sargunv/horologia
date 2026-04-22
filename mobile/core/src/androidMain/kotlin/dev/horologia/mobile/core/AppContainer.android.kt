package dev.horologia.mobile.core

import android.content.Context
import androidx.lifecycle.ViewModelProvider
import dev.horologia.mobile.core.feature.login.BootRouter
import dev.horologia.mobile.core.platform.BrowserLauncher
import dev.horologia.mobile.core.session.ServerPrefs
import dev.horologia.mobile.core.session.SessionHolder
import dev.horologia.mobile.core.session.SessionStore

/**
 * Android actual: takes a `Context` because EncryptedSharedPreferences and SharedPreferences both
 * need one. The [browserLauncher] is the :core-side shim; the real Custom-Tabs launcher is
 * installed from `:compose-app/MainActivity`.
 */
actual class AppContainer(context: Context) {
  private val core =
    AppContainerCore(
      sessionStore = SessionStore(context = context),
      serverPrefs = ServerPrefs(context = context),
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
