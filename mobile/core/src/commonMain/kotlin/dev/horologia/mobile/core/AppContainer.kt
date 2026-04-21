package dev.horologia.mobile.core

import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import dev.horologia.mobile.core.feature.profile.LiveProfileGateway
import dev.horologia.mobile.core.feature.profile.ProfileGateway
import dev.horologia.mobile.core.feature.profile.ProfileViewModel
import dev.horologia.mobile.core.feature.spaces.LiveSpacesGateway
import dev.horologia.mobile.core.feature.spaces.SpacesGateway
import dev.horologia.mobile.core.feature.spaces.SpacesViewModel

/**
 * Holds the gateways and ViewModel factories that back each screen. Requires
 * [configureHorologiaApi] to have been called first, so the generated `Api` singleton is wired with
 * a base URL and an auth provider.
 *
 * New features add their gateway + factory here as additional properties.
 */
class AppContainer {
  private val profileGateway: ProfileGateway = LiveProfileGateway()
  private val spacesGateway: SpacesGateway = LiveSpacesGateway()

  val profileViewModelFactory: ViewModelProvider.Factory = viewModelFactory {
    initializer { ProfileViewModel(profileGateway) }
  }

  val spacesViewModelFactory: ViewModelProvider.Factory = viewModelFactory {
    initializer { SpacesViewModel(spacesGateway) }
  }
}
