package dev.horologia.mobile.core

import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import com.kroegerama.openapi.kmp.gen.companion.AuthItem
import dev.horologia.mobile.core.screen.profile.LiveProfileGateway
import dev.horologia.mobile.core.screen.profile.ProfileGateway
import dev.horologia.mobile.core.screen.profile.ProfileViewModel
import dev.horologia.mobile.generated.Api
import dev.horologia.mobile.generated.Auth
import io.ktor.http.Url

// SINGLETON — the generated `Api` is a process-wide object. `AppContainer` must be
// constructed once at app start; constructing a second instance mutates shared state.
class AppContainer(baseUrl: String, getToken: () -> String?) {
  init {
    Api.baseUrl = Url(baseUrl)
    Api.setAuthProvider(Auth.BearerAuth { getToken()?.let { token -> AuthItem.Bearer(token) } })
  }

  private val profileGateway: ProfileGateway = LiveProfileGateway()

  val profileViewModelFactory: ViewModelProvider.Factory = viewModelFactory {
    initializer { ProfileViewModel(profileGateway) }
  }
}
