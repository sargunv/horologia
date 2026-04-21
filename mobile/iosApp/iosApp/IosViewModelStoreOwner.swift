import HorologiaCore
import SwiftUI

/// SwiftUI-friendly `ViewModelStoreOwner` that manages the `ViewModelStore`
/// lifecycle the way AndroidX's Compose ViewModel helpers do on Android.
final class IosViewModelStoreOwner: ObservableObject, ViewModelStoreOwner {
  let viewModelStore: ViewModelStore = ViewModelStore()

  func viewModel<T: ViewModel>(
    _ modelType: T.Type,
    factory: any ViewModelProviderFactory,
    key: String? = nil,
    extras: CreationExtras? = nil
  ) -> T {
    do {
      let resolved = try viewModelStore.resolveViewModel(
        modelClass: modelType,
        factory: factory,
        key: key,
        extras: extras
      )
      // Safe: `resolveViewModel` returns an instance of `modelType` when it succeeds;
      // a mismatch would indicate a Kotlin-side invariant violation.
      return resolved as! T
    } catch {
      fatalError("Failed to resolve ViewModel of type \(T.self): \(error)")
    }
  }

  deinit {
    viewModelStore.clear()
  }
}
