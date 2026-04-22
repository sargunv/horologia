import HorologiaCore
import SwiftUI

/// Server-picker + browser-handoff + finishing composite. Mirrors
/// `LoginScreen.kt` on Compose side. Compact (`.compact`) layout fills the
/// viewport; Regular uses a centered glass card. iOS 26's `.glassEffect()` and
/// `GlassEffectContainer` supply the Liquid Glass treatment on the card and
/// primary button (R22).
@MainActor
struct LoginView: View {
  @Environment(\.appContainer) private var container
  @Environment(\.horizontalSizeClass) private var horizontalSizeClass
  @StateObject private var storeOwner = IosViewModelStoreOwner()
  @State private var uiState: LoginUiState = LoginUiStateServerPicker(
    input: "",
    probe: ProbeStateEmpty.shared,
    banner: nil
  )
  @State private var lastAnnouncedBanner: String?
  @FocusState private var urlFieldFocused: Bool

  let initialServerUrl: String?
  let initialBanner: String?
  let onComplete: () -> Void

  init(
    initialServerUrl: String? = nil,
    initialBanner: String? = nil,
    onComplete: @escaping () -> Void
  ) {
    self.initialServerUrl = initialServerUrl
    self.initialBanner = initialBanner
    self.onComplete = onComplete
  }

  private var viewModel: LoginViewModel {
    storeOwner.viewModel(
      LoginViewModel.self,
      factory: container!.loginViewModelFactory
    )
  }

  var body: some View {
    Group {
      if horizontalSizeClass == .regular {
        expandedBody
      } else {
        compactBody
      }
    }
    .onAppear { urlFieldFocused = true }
    .task {
      if let url = initialServerUrl, !url.isEmpty {
        viewModel.seedInitialUrl(url: url)
      }
      if let banner = initialBanner, !banner.isEmpty {
        viewModel.showBanner(message: banner)
      }
      for await newState in viewModel.uiState {
        uiState = newState
        announceBannerIfNeeded(state: newState)
        if newState is LoginUiStateComplete {
          onComplete()
        }
      }
    }
  }

  private func announceBannerIfNeeded(state: LoginUiState) {
    let currentBanner: String?
    if let picker = state as? LoginUiStateServerPicker {
      currentBanner = picker.banner
    } else {
      currentBanner = nil
    }
    if currentBanner != lastAnnouncedBanner {
      lastAnnouncedBanner = currentBanner
      if let message = currentBanner {
        AccessibilityNotification.Announcement(message).post()
      }
    }
  }

  private var compactBody: some View {
    VStack(alignment: .leading, spacing: 16) {
      headline
      content
      Spacer()
    }
    .padding(24)
    .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
  }

  private var expandedBody: some View {
    VStack {
      Spacer()
      GlassEffectContainer {
        VStack(alignment: .leading, spacing: 16) {
          headline
          content
        }
        .padding(32)
        .frame(maxWidth: 480)
        .glassEffect(in: .rect(cornerRadius: 28))
      }
      Spacer()
    }
    .frame(maxWidth: .infinity, maxHeight: .infinity)
    .padding(32)
  }

  private var headline: some View {
    VStack(alignment: .leading, spacing: 8) {
      Text("Connect to Horologia")
        .font(.largeTitle.weight(.bold))
      Text("Paste your server URL to sign in.")
        .font(.body)
        .foregroundStyle(.secondary)
    }
  }

  @ViewBuilder
  private var content: some View {
    switch onEnum(of: uiState) {
    case .serverPicker(let picker):
      serverPickerBody(picker: picker)
    case .launchingBrowser(let state):
      statusBody(
        title: "Opening sign-in…",
        detail: state.input,
        onCancel: { viewModel.cancelSignIn() }
      )
    case .finishing(let state):
      statusBody(
        title: "Finishing sign-in…",
        detail: state.input,
        onCancel: { viewModel.cancelSignIn() }
      )
    case .complete:
      statusBody(title: "Signed in.", detail: nil, onCancel: nil)
    }
  }

  private func serverPickerBody(picker: LoginUiStateServerPicker) -> some View {
    VStack(alignment: .leading, spacing: 12) {
      if let banner = picker.banner {
        HStack(alignment: .top, spacing: 8) {
          Image(systemName: "exclamationmark.triangle.fill")
            .foregroundStyle(Color.red)
            .accessibilityHidden(true)
          Text(banner)
            .font(.body)
            .foregroundStyle(Color.red)
            .accessibilityAddTraits(.isStaticText)
          Spacer(minLength: 8)
          Button {
            viewModel.dismissBanner()
          } label: {
            Image(systemName: "xmark")
          }
          .accessibilityLabel("Dismiss banner")
        }
      }
      TextField(
        "",
        text: Binding(
          get: { picker.input },
          set: { viewModel.onUrlChanged(input: $0) }
        ),
        prompt: Text("tasks.example.com")
      )
      .accessibilityLabel("Server URL")
      .textInputAutocapitalization(.never)
      .autocorrectionDisabled(true)
      .keyboardType(.URL)
      .submitLabel(.go)
      .onSubmit {
        if picker.probe is ProbeStateValid { viewModel.onSubmit() }
      }
      .focused($urlFieldFocused)
      .textFieldStyle(.roundedBorder)

      probeSupportingText(picker.probe)

      Button(action: { viewModel.onSubmit() }) {
        Text("Continue")
          .frame(maxWidth: .infinity)
      }
      .buttonStyle(.borderedProminent)
      .disabled(!(picker.probe is ProbeStateValid))
      .glassEffect(in: .capsule)
    }
  }

  @ViewBuilder
  private func probeSupportingText(_ probe: ProbeState) -> some View {
    switch onEnum(of: probe) {
    case .empty, .typing:
      EmptyView()
    case .probing:
      Text("Checking server…")
        .font(.caption)
        .foregroundStyle(.secondary)
    case .valid:
      Text("Horologia server detected.")
        .font(.caption)
        .foregroundStyle(.secondary)
    case .invalidUnreachable(let unreachable):
      HStack(spacing: 4) {
        Image(systemName: "exclamationmark.triangle.fill").accessibilityHidden(true)
        Text("Can't reach \(unreachable.host).")
      }
      .font(.caption)
      .foregroundStyle(Color.red)
    case .invalidWrongServer:
      HStack(spacing: 4) {
        Image(systemName: "exclamationmark.triangle.fill").accessibilityHidden(true)
        Text("Not a Horologia server.")
      }
      .font(.caption)
      .foregroundStyle(Color.red)
    }
  }

  private func statusBody(
    title: String,
    detail: String?,
    onCancel: (() -> Void)?
  ) -> some View {
    VStack(alignment: .leading, spacing: 12) {
      Text(title)
        .font(.title3)
      if let detail = detail {
        Text(detail)
          .font(.body)
          .foregroundStyle(.secondary)
      }
      ProgressView()
        .progressViewStyle(.linear)
        .accessibilityLabel(title)
      if let onCancel = onCancel {
        // ASWebAuthenticationSession already signals dismissal via canceledLogin, so this is
        // a redundant escape hatch on iOS — useful when the sheet is stuck or the user changed
        // their mind mid-flow.
        Button("Cancel", action: onCancel).buttonStyle(.bordered)
      }
    }
  }
}
