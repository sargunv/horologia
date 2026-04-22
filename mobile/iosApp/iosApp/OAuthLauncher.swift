import AuthenticationServices
import HorologiaCore
import SwiftUI

/// Swift-side implementation of the Kotlin `IosBrowserLauncherBridge` contract.
/// Uses `ASWebAuthenticationSession` for the outbound + inbound browser trip.
///
/// Throws `NSError` with domain `IosBrowserLauncherBridgeCompanion.shared.ErrorDomain`
/// and code `CancelledCode` / `FailedCode` to signal outcomes back to Kotlin — which
/// re-throws typed `BrowserCancelledException` / `BrowserFailedException` via
/// `IosBrowserLauncher`. See `IosBrowserLauncherBridge.kt` for the contract.
@MainActor
final class OAuthLauncher: NSObject, IosBrowserLauncherBridge,
  ASWebAuthenticationPresentationContextProviding
{
  private var currentSession: ASWebAuthenticationSession?

  // SKIE translates Kotlin `suspend fun launchAndAwait(...)` to an `async throws` Swift
  // method named `__launchAndAwait` on the generated protocol; the user-facing
  // `launchAndAwait` is exposed via an extension that trampolines to this one.
  func __launchAndAwait(authorizeUrl: String, redirectUri: String) async throws -> String {
    try await withTaskCancellationHandler(operation: {
      try await withCheckedThrowingContinuation {
        [weak self] (continuation: CheckedContinuation<String, Error>) in
        guard let self = self else {
          continuation.resume(throwing: OAuthLauncher.failedError("Sign-in unavailable."))
          return
        }
        guard let authURL = URL(string: authorizeUrl),
              let components = URLComponents(string: redirectUri),
              let scheme = components.scheme
        else {
          continuation.resume(throwing: OAuthLauncher.failedError("Invalid sign-in URL."))
          return
        }

        let session = ASWebAuthenticationSession(url: authURL, callbackURLScheme: scheme) {
          [weak self] callbackURL, error in
          defer {
            DispatchQueue.main.async { [weak self] in self?.currentSession = nil }
          }
          if let error = error {
            let nsError = error as NSError
            if nsError.domain == ASWebAuthenticationSessionErrorDomain {
              switch nsError.code {
              case ASWebAuthenticationSessionError.canceledLogin.rawValue:
                continuation.resume(throwing: OAuthLauncher.cancelledError())
                return
              case ASWebAuthenticationSessionError.presentationContextNotProvided.rawValue,
                   ASWebAuthenticationSessionError.presentationContextInvalid.rawValue:
                continuation.resume(throwing: OAuthLauncher.failedError("Couldn't present sign-in."))
                return
              default:
                break
              }
            }
            continuation.resume(
              throwing: OAuthLauncher.failedError(error.localizedDescription))
            return
          }
          if let callbackURL = callbackURL {
            continuation.resume(returning: callbackURL.absoluteString)
          } else {
            continuation.resume(throwing: OAuthLauncher.cancelledError())
          }
        }
        session.presentationContextProvider = self
        session.prefersEphemeralWebBrowserSession = true

        self.currentSession = session
        if !session.start() {
          self.currentSession = nil
          continuation.resume(throwing: OAuthLauncher.failedError("Couldn't present sign-in."))
        }
      }
    }, onCancel: { [weak self] in
      // onCancel is @Sendable; hop to the main actor via GCD to cancel the session since
      // `currentSession` is @MainActor-isolated and the cancellation handler isn't.
      DispatchQueue.main.async {
        self?.currentSession?.cancel()
        self?.currentSession = nil
      }
    })
  }

  func presentationAnchor(for session: ASWebAuthenticationSession) -> ASPresentationAnchor {
    // Pick a foreground-active window scene's key window. Falling back to an empty
    // ASPresentationAnchor() triggers `.presentationContextInvalid`, so prefer a
    // clear fatalError-equivalent over silent breakage — in practice, SwiftUI
    // always has at least one active scene by the time the user taps Continue.
    let foregroundScenes = UIApplication.shared.connectedScenes
      .filter { $0.activationState == .foregroundActive }
    for scene in foregroundScenes {
      if let windowScene = scene as? UIWindowScene {
        if let keyWindow = windowScene.windows.first(where: { $0.isKeyWindow }) {
          return keyWindow
        }
        if let anyWindow = windowScene.windows.first {
          return anyWindow
        }
      }
    }
    return ASPresentationAnchor()
  }

  private static var errorDomain: String {
    IosBrowserLauncherBridgeCompanion.shared.ErrorDomain
  }

  private static func cancelledError() -> NSError {
    NSError(
      domain: errorDomain,
      code: Int(IosBrowserLauncherBridgeCompanion.shared.CancelledCode),
      userInfo: [NSLocalizedDescriptionKey: "Sign-in cancelled."]
    )
  }

  private static func failedError(_ message: String) -> NSError {
    NSError(
      domain: errorDomain,
      code: Int(IosBrowserLauncherBridgeCompanion.shared.FailedCode),
      userInfo: [NSLocalizedDescriptionKey: message]
    )
  }
}
