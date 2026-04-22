import AuthenticationServices
import HorologiaCore
import SwiftUI

/// Swift-side implementation of the Kotlin `IosBrowserLauncher` contract.
/// Uses `ASWebAuthenticationSession` for the outbound + inbound browser trip;
/// the session's completion handler fires with either the callback URL
/// (on success) or an NSError (on user cancellation / browser failure).
///
/// Installed into `BrowserLauncher.Companion.shared` at app-init time so the
/// Kotlin login VM can call `BrowserLauncher().launchAndAwait(authorizeUrl:)`
/// without knowing anything about `ASWebAuthenticationSession`.
@MainActor
final class OAuthLauncher: NSObject, IosBrowserLauncher, ASWebAuthenticationPresentationContextProviding {
  private var currentSession: ASWebAuthenticationSession?

  func launchAndAwait(authorizeUrl: String, redirectUri: String) async throws -> String {
    try await withCheckedThrowingContinuation { continuation in
      guard let authURL = URL(string: authorizeUrl),
            let components = URLComponents(string: redirectUri),
            let scheme = components.scheme else {
        continuation.resume(throwing: BrowserCancelledException())
        return
      }

      let session = ASWebAuthenticationSession(url: authURL, callbackURLScheme: scheme) { callbackURL, error in
        if let error = error {
          let nsError = error as NSError
          if nsError.domain == ASWebAuthenticationSessionErrorDomain &&
             nsError.code == ASWebAuthenticationSessionError.canceledLogin.rawValue {
            continuation.resume(throwing: BrowserCancelledException())
          } else {
            continuation.resume(throwing: error)
          }
          return
        }
        if let callbackURL = callbackURL {
          continuation.resume(returning: callbackURL.absoluteString)
        } else {
          continuation.resume(throwing: BrowserCancelledException())
        }
      }
      session.presentationContextProvider = self
      session.prefersEphemeralWebBrowserSession = true

      self.currentSession = session
      if !session.start() {
        continuation.resume(throwing: BrowserCancelledException())
      }
    }
  }

  func presentationAnchor(for session: ASWebAuthenticationSession) -> ASPresentationAnchor {
    let scenes = UIApplication.shared.connectedScenes
    let windowScene = scenes.first as? UIWindowScene
    return windowScene?.windows.first ?? ASPresentationAnchor()
  }
}
