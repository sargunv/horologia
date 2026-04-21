import HorologiaCore
import SwiftUI

@MainActor
struct ContentView: View {
  let viewModel: ProfileViewModel

  @State private var uiState: ProfileUiState = ProfileUiStateLoading.shared

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
          .foregroundStyle(Color(uiColor: .systemRed))
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
