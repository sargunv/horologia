import HorologiaCore
import SwiftUI

@main
@MainActor
struct HorologiaApp: App {
  @State private var container: AppContainer
  @State private var startRoot: StartRoot
  @State private var initialServerUrl: String?
  @State private var initialBanner: String?

  init() {
    let container = AppContainer()

    // Swift-side OAuthLauncher must be installed before any screen calls the VM.
    BrowserLauncherCompanion.shared.install(launcher: OAuthLauncher())

    // Run the bootRouter on a dispatch queue synchronously to decide the first-frame destination.
    let destination = Self.blockingDecideBoot(using: container)
    let baseUrl = Self.baseUrlFor(destination: destination)
    ApiBootstrapKt.configureHorologiaApi(
      baseUrl: baseUrl,
      getToken: { container.sessionHolder.currentAccessToken() }
    )

    let (root, url, banner) = Self.interpret(destination: destination)

    self._container = State(initialValue: container)
    self._startRoot = State(initialValue: root)
    self._initialServerUrl = State(initialValue: url)
    self._initialBanner = State(initialValue: banner)
  }

  var body: some Scene {
    WindowGroup {
      Group {
        switch startRoot {
        case .login:
          NavigationStack {
            LoginView(
              initialServerUrl: initialServerUrl,
              initialBanner: initialBanner,
              onComplete: { startRoot = .profile }
            )
          }
        case .profile:
          NavigationStack {
            ProfileView()
              .navigationDestination(for: Route.self) { route in
                switch route {
                case .spaces:
                  SpacesView()
                }
              }
          }
        }
      }
      .environment(\.appContainer, container)
    }
  }

  private enum StartRoot {
    case login
    case profile
  }

  private static func blockingDecideBoot(using container: AppContainer) -> BootDestination {
    // Use a DispatchSemaphore to bridge the suspend function to sync init.
    var result: BootDestination = BootDestinationUnconfigured.shared
    let semaphore = DispatchSemaphore(value: 0)
    Task {
      result = try! await container.bootRouter.decideBootDestination()
      semaphore.signal()
    }
    _ = semaphore.wait(timeout: .now() + .seconds(3))
    return result
  }

  private static func baseUrlFor(destination: BootDestination) -> String {
    switch onEnum(of: destination) {
    case .signedIn(let signed): return ensureApiPath(signed.savedUrl)
    case .serverOnly(let server): return ensureApiPath(server.savedUrl)
    case .signedOutAfterRefresh(let out): return ensureApiPath(out.savedUrl)
    case .unconfigured: return "https://horologia.invalid/api/"
    }
  }

  private static func ensureApiPath(_ raw: String) -> String {
    var trimmed = raw
    while trimmed.hasSuffix("/") { trimmed.removeLast() }
    if trimmed.hasSuffix("/api") || trimmed.contains("/api/") {
      return trimmed + "/"
    }
    return trimmed + "/api/"
  }

  private static func interpret(destination: BootDestination) -> (StartRoot, String?, String?) {
    switch onEnum(of: destination) {
    case .unconfigured: return (.login, nil, nil)
    case .serverOnly(let server): return (.login, server.savedUrl, nil)
    case .signedIn(let signed): return (.profile, signed.savedUrl, nil)
    case .signedOutAfterRefresh(let out): return (.login, out.savedUrl, "Signed out.")
    }
  }
}
