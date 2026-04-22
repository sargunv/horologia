import AuthenticationServices
import HorologiaCore
import SwiftUI

/// Swift-side implementation of the Kotlin `IosBrowserLauncherBridge` contract.
/// Uses `ASWebAuthenticationSession` for the outbound + inbound browser trip;
/// the session's completion handler fires with either the callback URL
/// (on success) or an NSError (on user cancellation / browser failure).
///
/// Handed to the Kotlin `IosBrowserLauncher` at construction so the Kotlin login
/// VM can call `launchAndAwait(authorizeUrl:)` without knowing anything about
/// `ASWebAuthenticationSession`.
@MainActor
final class OAuthLauncher: NSObject, IosBrowserLauncherBridge,
  ASWebAuthenticationPresentationContextProviding
{
  private var currentSession: ASWebAuthenticationSession?

  func launchAndAwait(authorizeUrl: String, redirectUri: String) async throws -> String {
    try await withTaskCancellationHandler {
      try await withCheckedThrowingContinuation {
        [weak self] (continuation: CheckedContinuation<String, Error>) in
        guard let self = self else {
          continuation.resume(throwing: BrowserFailedException(message: "Sign-in unavailable."))
          return
        }
        guard let authURL = URL(string: authorizeUrl),
              let components = URLComponents(string: redirectUri),
              let scheme = components.scheme
        else {
          continuation.resume(throwing: BrowserFailedException(message: "Invalid sign-in URL."))
          return
        }

        let session = ASWebAuthenticationSession(url: authURL, callbackURLScheme: scheme) {
          [weak self] callbackURL, error in
          defer { Task { @MainActor [weak self] in self?.currentSession = nil } }
          if let error = error {
            let nsError = error as NSError
            if nsError.domain == ASWebAuthenticationSessionErrorDomain {
              switch nsError.code {
              case ASWebAuthenticationSessionError.canceledLogin.rawValue:
                continuation.resume(throwing: BrowserCancelledException(message: "OAuth sign-in cancelled by user"))
                return
              case ASWebAuthenticationSessionError.presentationContextNotProvided.rawValue,
                   ASWebAuthenticationSessionError.presentationContextInvalid.rawValue:
                continuation.resume(throwing: BrowserFailedException(message: "Couldn't present sign-in."))
                return
              default:
                break
              }
            }
            continuation.resume(throwing: error)
            return
          }
          if let callbackURL = callbackURL {
            continuation.resume(returning: callbackURL.absoluteString)
          } else {
            continuation.resume(throwing: BrowserCancelledException(message: "OAuth sign-in cancelled by user"))
          }
        }
        session.presentationContextProvider = self
        session.prefersEphemeralWebBrowserSession = true

        self.currentSession = session
        if !session.start() {
          self.currentSession = nil
          continuation.resume(throwing: BrowserFailedException(message: "Couldn't present sign-in."))
        }
      }
    } onCancel: {
      Task { @MainActor [weak self] in
        self?.currentSession?.cancel()
        self?.currentSession = nil
      }
    }
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
    // No foreground scene available — return an empty anchor, which ASWeb...
    // will reject with `.presentationContextInvalid`, surfaced as a
    // BrowserFailedException above.
    return ASPresentationAnchor()
  }
}
