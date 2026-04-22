import HorologiaCore
import SwiftUI

@MainActor
struct ProfileView: View {
  @Environment(\.appContainer) private var container
  @StateObject private var storeOwner = IosViewModelStoreOwner()
  @State private var uiState: ProfileUiState = ProfileUiStateLoading.shared

  /// `IosViewModelStoreOwner.viewModel(_:factory:)` resolves through the underlying
  /// `ViewModelStore`, which caches by model class — every call with the same type
  /// returns the same instance for the lifetime of `storeOwner`. That's what lets us
  /// compute this fresh on each `body` evaluation without spawning new ViewModels.
  private var viewModel: ProfileViewModel {
    storeOwner.viewModel(
      ProfileViewModel.self,
      factory: container!.profileViewModelFactory
    )
  }

  var body: some View {
    VStack(alignment: .leading, spacing: 16) {
      Text("Horologia")
        .font(.largeTitle.weight(.bold))

      switch onEnum(of: uiState) {
      case .loading:
        ProgressView()
          .progressViewStyle(.circular)
          .padding(.top, 8)
          .accessibilityLabel("Loading profile")

      case .success(let success):
        Text("Signed in as \(success.displayName)")
          .font(.title3)
          .accessibilityAddTraits(.updatesFrequently)

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

      NavigationLink("View spaces", value: Route.spaces)
        .buttonStyle(.borderedProminent)
        .padding(.top, 8)
    }
    .padding(24)
    .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
    .task {
      for await newState in viewModel.uiState {
        uiState = newState
        switch onEnum(of: newState) {
        case .success(let success):
          AccessibilityNotification.Announcement("Signed in as \(success.displayName)").post()
        case .error(let error):
          AccessibilityNotification.Announcement(error.message).post()
        case .loading:
          break
        }
      }
    }
  }
}
