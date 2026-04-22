import HorologiaCore
import SwiftUI

@main
@MainActor
struct HorologiaApp: App {
  @State private var container: AppContainer
  @State private var resolvedRoot: ResolvedRoot?

  init() {
    let container = AppContainer()

    // Swift-side OAuthLauncher must be installed before any screen calls the VM.
    BrowserLauncherCompanion.shared.install(launcher: OAuthLauncher())

    // Configure the Api singleton with a placeholder so any call before routing
    // resolves doesn't NPE on a missing base URL. `RootView` re-configures with
    // the resolved URL inside `.task { }`.
    ApiBootstrapKt.configureHorologiaApi(
      baseUrl: "https://horologia.invalid/api/",
      getToken: { container.sessionHolder.currentAccessToken() }
    )

    self._container = State(initialValue: container)
    self._resolvedRoot = State(initialValue: nil)
  }

  var body: some Scene {
    WindowGroup {
      RootView(container: container, resolvedRoot: $resolvedRoot)
        .environment(\.appContainer, container)
    }
  }
}

private struct ResolvedRoot: Equatable {
  var start: StartRoot
  var initialServerUrl: String?
  var initialBanner: String?
}

enum StartRoot: Equatable {
  case login
  case profile
}

@MainActor
private struct RootView: View {
  let container: AppContainer
  @Binding var resolvedRoot: ResolvedRoot?

  var body: some View {
    Group {
      if let resolved = resolvedRoot {
        switch resolved.start {
        case .login:
          NavigationStack {
            LoginView(
              initialServerUrl: resolved.initialServerUrl,
              initialBanner: resolved.initialBanner,
              onComplete: { resolvedRoot?.start = .profile }
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
      } else {
        splash
      }
    }
    .task {
      if resolvedRoot != nil { return }
      do {
        let destination = try await container.bootRouter.decideBootDestination()
        let baseUrl = Self.baseUrlFor(destination: destination)
        ApiBootstrapKt.configureHorologiaApi(
          baseUrl: baseUrl,
          getToken: { container.sessionHolder.currentAccessToken() }
        )
        let (root, url, banner) = Self.interpret(destination: destination)
        resolvedRoot = ResolvedRoot(start: root, initialServerUrl: url, initialBanner: banner)
      } catch {
        // Decode failure: treat as Unconfigured and surface to devs via print().
        print("HorologiaApp: decideBootDestination failed: \(error); defaulting to Unconfigured")
        resolvedRoot = ResolvedRoot(start: .login, initialServerUrl: nil, initialBanner: nil)
      }
    }
  }

  private var splash: some View {
    VStack {
      ProgressView()
        .accessibilityLabel("Loading")
    }
    .frame(maxWidth: .infinity, maxHeight: .infinity)
  }

  private static func baseUrlFor(destination: BootDestination) -> String {
    switch onEnum(of: destination) {
    case .signedIn(let signed): return ensureApiPath(signed.savedUrl)
    case .serverOnly(let server): return ensureApiPath(server.savedUrl)
    case .signedOutAfterRefresh(let out): return ensureApiPath(out.savedUrl)
    case .unconfigured: return "https://horologia.invalid/api/"
    }
  }

  /// Mirrors `ApiBootstrap.ensureApiPath` in `:core`. Kept in Swift because calling
  /// the Kotlin top-level helper through the generated framework binding adds an
  /// awkward `ApiBootstrapKt.ensureApiPath(...)` call-site; the invariant is tiny
  /// enough that a local mirror is cheaper than the indirection. If either side
  /// changes, update the other.
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
