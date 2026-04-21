import HorologiaCore
import SwiftUI

/// SwiftUI-friendly `ViewModelStoreOwner` that manages the `ViewModelStore`
/// lifecycle the way AndroidX's Compose ViewModel helpers do on Android.
final class IosViewModelStoreOwner: ObservableObject, ViewModelStoreOwner {
  let viewModelStore: ViewModelStore = ViewModelStore()

  func viewModel<T: ViewModel>(
    modelClass: AnyClass,
    factory: any ViewModelProviderFactory,
    key: String? = nil,
    extras: CreationExtras? = nil
  ) -> T {
    do {
      let resolved = try viewModelStore.resolveViewModel(
        modelClass: modelClass,
        factory: factory,
        key: key,
        extras: extras
      )
      guard let viewModel = resolved as? T else {
        fatalError(
          "Resolved ViewModel has unexpected type \(type(of: resolved)); expected \(T.self)"
        )
      }
      return viewModel
    } catch {
      fatalError("Failed to resolve ViewModel of type \(T.self): \(error)")
    }
  }

  deinit {
    viewModelStore.clear()
  }
}
