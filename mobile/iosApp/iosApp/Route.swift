import Foundation

/// Routes for the SwiftUI `NavigationStack`. Each case corresponds to a destination reachable
/// from the Profile root. The Compose side (`dev.horologia.mobile.compose.nav`) has its own
/// route types for the same destinations, but the two lists aren't literally symmetric — the
/// Profile root isn't a `Route` case here because SwiftUI treats it as the stack's root view,
/// while Compose models every destination (including the start) uniformly.
enum Route: Hashable {
  case spaces
}
