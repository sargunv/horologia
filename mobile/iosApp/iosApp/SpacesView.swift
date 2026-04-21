import HorologiaCore
import SwiftUI

@MainActor
struct SpacesView: View {
  @Environment(\.appContainer) private var container
  @StateObject private var storeOwner = IosViewModelStoreOwner()
  @State private var uiState: SpacesUiState = SpacesUiStateLoading.shared

  /// Same caching guarantee as `ProfileView.viewModel` — the `ViewModelStore`
  /// dedupes by model class, so repeated `body` invocations hit the cache.
  private var viewModel: SpacesViewModel {
    storeOwner.viewModel(
      SpacesViewModel.self,
      factory: container!.spacesViewModelFactory
    )
  }

  var body: some View {
    VStack(alignment: .leading, spacing: 16) {
      switch onEnum(of: uiState) {
      case .loading:
        ProgressView()
          .progressViewStyle(.circular)
          .padding(.top, 8)
          .accessibilityLabel("Loading spaces")

      case .success(let success):
        List(success.items, id: \.slug) { item in
          Text(item.name)
            .font(.body)
        }
        .listStyle(.plain)

      case .error(let error):
        Text(error.message)
          .font(.body)
          .foregroundStyle(Color.red)
          .accessibilityAddTraits(.updatesFrequently)
        if error.retryable {
          Button("Retry") {
            viewModel.refresh()
          }
          .buttonStyle(.borderedProminent)
        }
      }
    }
    .padding(24)
    .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
    .navigationTitle("Spaces")
    .navigationBarTitleDisplayMode(.inline)
    .task {
      for await newState in viewModel.uiState {
        uiState = newState
        switch onEnum(of: newState) {
        case .success(let success):
          let count = success.items.count
          AccessibilityNotification.Announcement(
            count == 1 ? "Loaded 1 space" : "Loaded \(count) spaces"
          ).post()
        case .error(let error):
          AccessibilityNotification.Announcement(error.message).post()
        case .loading:
          break
        }
      }
    }
  }
}
