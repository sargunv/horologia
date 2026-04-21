import HorologiaCore
import SwiftUI

@main
@MainActor
struct HorologiaApp: App {
  // TODO: Replace dev-mode bearer token with real auth flow:
  //   POST /app/auth/login → cookie session → POST /api/auth/tokens (bearer token).
  @StateObject private var storeOwner = IosViewModelStoreOwner()

  private let appContainer: AppContainer
  private let profileViewModel: ProfileViewModel

  init() {
    let infoDictionary = Bundle.main.infoDictionary
    let baseUrl =
      (infoDictionary?["HorologiaBaseUrl"] as? String).flatMap { $0.isEmpty ? nil : $0 }
      ?? "http://localhost:8080/api/"
    let devToken =
      (infoDictionary?["HorologiaDevToken"] as? String).flatMap { $0.isEmpty ? nil : $0 }

    let container = AppContainer(
      baseUrl: baseUrl,
      getToken: { devToken }
    )
    self.appContainer = container

    let owner = IosViewModelStoreOwner()
    self._storeOwner = StateObject(wrappedValue: owner)
    self.profileViewModel = owner.viewModel(
      ProfileViewModel.self,
      factory: container.profileViewModelFactory
    )
  }

  var body: some Scene {
    WindowGroup {
      ContentView(viewModel: profileViewModel)
    }
  }
}
