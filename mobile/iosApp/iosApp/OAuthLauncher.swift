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
          // The completion fires on an arbitrary queue, but `currentSession` is @MainActor.
          // `Task { @MainActor in … }` would be idiomatic but Swift 6's parser reads the inner
          // braces as an extra trailing closure on the enclosing `withTaskCancellationHandler`;
          // `DispatchQueue.main.async` + `MainActor.assumeIsolated` achieves the same static
          // isolation guarantee without the parser ambiguity.
          defer {
            DispatchQueue.main.async {
              MainActor.assumeIsolated { self?.clearCurrentSession(cancel: false) }
            }
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
      // onCancel is non-isolated `@Sendable`; hop to the main actor to touch `currentSession`.
      // See the matching comment above for why this goes through GCD + `assumeIsolated`
      // instead of `Task { @MainActor in … }` (Swift 6 trailing-closure parser ambiguity).
      DispatchQueue.main.async {
        MainActor.assumeIsolated { self?.clearCurrentSession(cancel: true) }
      }
    })
  }

  /// Drop the current session reference, optionally calling `.cancel()` first. The completion
  /// handler path just clears the reference (the session's already done); the onCancel path
  /// also cancels so `ASWebAuthenticationSession` tears down the sheet if it's still up.
  @MainActor
  private func clearCurrentSession(cancel: Bool) {
    if cancel { currentSession?.cancel() }
    currentSession = nil
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

  /// Swift-idiomatic outcomes for the browser bridge. The Kotlin side of [IosBrowserLauncher]
  /// checks the resulting NSError's domain+code to decide which typed exception to re-throw —
  /// see [OAuthLauncherError.toNSError] for the mapping. Keep the enum + bridge in one place so
  /// the integer codes aren't scattered across call sites.
  private enum OAuthLauncherError: Error {
    case cancelled(message: String = "Sign-in cancelled.")
    case failed(message: String)

    func toNSError() -> NSError {
      let companion = IosBrowserLauncherBridgeCompanion.shared
      switch self {
      case .cancelled(let message):
        return NSError(
          domain: companion.ErrorDomain,
          code: Int(companion.CancelledCode),
          userInfo: [NSLocalizedDescriptionKey: message]
        )
      case .failed(let message):
        return NSError(
          domain: companion.ErrorDomain,
          code: Int(companion.FailedCode),
          userInfo: [NSLocalizedDescriptionKey: message]
        )
      }
    }
  }

  private static func cancelledError() -> NSError {
    OAuthLauncherError.cancelled().toNSError()
  }

  private static func failedError(_ message: String) -> NSError {
    OAuthLauncherError.failed(message: message).toNSError()
  }
}
