import HorologiaCore
import SwiftUI

@main
struct HorologiaApp: App {
  // TODO: Replace dev-mode bearer token with real auth flow:
  //   POST /app/auth/login → cookie session → POST /api/auth/tokens (bearer token).
  @StateObject private var storeOwner = IosViewModelStoreOwner()

  private let appContainer: AppContainer

  init() {
    let infoDictionary = Bundle.main.infoDictionary
    let baseUrl =
      (infoDictionary?["HorologiaBaseUrl"] as? String).flatMap { $0.isEmpty ? nil : $0 }
      ?? "http://localhost:8080/api/"
    let devToken =
      (infoDictionary?["HorologiaDevToken"] as? String).flatMap { $0.isEmpty ? nil : $0 }

    self.appContainer = AppContainer(
      baseUrl: baseUrl,
      getToken: { devToken }
    )
  }

  var body: some Scene {
    WindowGroup {
      ContentView(
        storeOwner: storeOwner,
        profileViewModelFactory: appContainer.profileViewModelFactory
      )
    }
  }
}
