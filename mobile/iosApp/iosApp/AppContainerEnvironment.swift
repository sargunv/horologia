import HorologiaCore
import SwiftUI

/// SwiftUI environment value for the Kotlin `AppContainer`. `AppContainer` is constructed
/// once at app start and never swapped, so it needs no `ObservableObject` wrapper —
/// an environment value is the idiomatic shape for this kind of read-only dependency.
///
/// The value is `Optional` so the key can satisfy `EnvironmentKey`'s `Sendable` requirement
/// under Swift 6 strict concurrency (a `nil` default is trivially sendable, while a
/// `@MainActor`-isolated `AppContainer` default value is not). At read sites, views
/// force-unwrap; the app always injects a real container at the root.
extension EnvironmentValues {
  @Entry var appContainer: AppContainer? = nil
}
