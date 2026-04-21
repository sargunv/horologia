import HorologiaCore
import SwiftUI

@main
@MainActor
struct HorologiaApp: App {
  // TODO: Replace dev-mode bearer token with real auth flow:
  //   POST /app/auth/login → cookie session → POST /api/auth/tokens (bearer token).
  private let container: AppContainer

  init() {
    let infoDictionary = Bundle.main.infoDictionary
    let baseUrl =
      (infoDictionary?["HorologiaBaseUrl"] as? String).flatMap { $0.isEmpty ? nil : $0 }
      ?? "http://localhost:8080/api/"
    let devToken =
      (infoDictionary?["HorologiaDevToken"] as? String).flatMap { $0.isEmpty ? nil : $0 }

    ApiBootstrapKt.configureHorologiaApi(baseUrl: baseUrl, getToken: { devToken })
    self.container = AppContainer()
  }

  var body: some Scene {
    WindowGroup {
      NavigationStack {
        ProfileView()
          .navigationDestination(for: Route.self) { route in
            switch route {
            case .spaces:
              SpacesView()
            }
          }
      }
      .environment(\.appContainer, container)
    }
  }
}
