// `@preconcurrency` strips `@MainActor` from ASWebAuthenticationSession's imported
// completion-handler type. On macOS the framework delivers that completion from an
// NSXPC reply queue without hopping to main, so the Swift runtime's main-actor check
// at closure entry traps with SIGTRAP. See the class doc below for the full fix.
@preconcurrency import AuthenticationServices
import HorologiaCore
import SwiftUI

/// Swift-side implementation of the Kotlin `IosBrowserLauncherBridge` contract.
/// Uses `ASWebAuthenticationSession` for the outbound + inbound browser trip.
///
/// Throws `NSError` with domain `IosBrowserLauncherBridgeCompanion.shared.ErrorDomain`
/// and code `CancelledCode` / `FailedCode` to signal outcomes back to Kotlin — which
/// re-throws typed `BrowserCancelledException` / `BrowserFailedException` via
/// `IosBrowserLauncher`. See `IosBrowserLauncherBridge.kt` for the contract.
///
/// **Isolation note** — three things conspire to keep the completion closure truly
/// non-isolated so macOS's off-main XPC delivery doesn't trap:
/// 1. The class is not `@MainActor`; `currentSession` is guarded with `NSLock` instead.
/// 2. The completion is bound to an explicit `@Sendable (URL?, Error?) -> Void` local,
///    blocking isolation inference from the `ASWebAuthenticationPresentationContextProviding`
///    conformance (that protocol became `@MainActor` in recent SDKs).
/// 3. Inside the completion we hop to the main queue via `DispatchQueue.main.async`
///    before resuming the continuation or touching shared state.
final class OAuthLauncher: NSObject, IosBrowserLauncherBridge,
  ASWebAuthenticationPresentationContextProviding, @unchecked Sendable
{
  private let sessionLock = NSLock()
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

        // See class doc for why the explicit `@Sendable` type + main-queue hop are both needed.
        let handler: @Sendable (URL?, Error?) -> Void = { [weak self] callbackURL, error in
          DispatchQueue.main.async {
            self?.setCurrentSession(nil, cancelPrevious: false)
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
              continuation.resume(throwing: OAuthLauncher.failedError(error.localizedDescription))
              return
            }
            if let callbackURL = callbackURL {
              continuation.resume(returning: callbackURL.absoluteString)
            } else {
              continuation.resume(throwing: OAuthLauncher.cancelledError())
            }
          }
        }
        let session = ASWebAuthenticationSession(
          url: authURL,
          callbackURLScheme: scheme,
          completionHandler: handler
        )
        session.presentationContextProvider = self
        session.prefersEphemeralWebBrowserSession = true

        self.setCurrentSession(session, cancelPrevious: false)
        if !session.start() {
          self.setCurrentSession(nil, cancelPrevious: false)
          continuation.resume(throwing: OAuthLauncher.failedError("Couldn't present sign-in."))
        }
      }
    }, onCancel: { [weak self] in
      self?.setCurrentSession(nil, cancelPrevious: true)
    })
  }

  /// Swap `currentSession` under the lock. When `cancelPrevious` is true and a session was
  /// live, `.cancel()` on it tears down the browser sheet — used by the cancellation path.
  private func setCurrentSession(
    _ session: ASWebAuthenticationSession?,
    cancelPrevious: Bool
  ) {
    sessionLock.lock()
    let previous = currentSession
    currentSession = session
    sessionLock.unlock()
    if cancelPrevious { previous?.cancel() }
  }

  nonisolated func presentationAnchor(for session: ASWebAuthenticationSession) -> ASPresentationAnchor {
    // The framework invokes this on the main thread; `MainActor.assumeIsolated` validates
    // that at runtime and gives us the isolation needed for UIApplication / NSApplication.
    MainActor.assumeIsolated {
      // Falling back to an empty ASPresentationAnchor() triggers `.presentationContextInvalid`,
      // so prefer a real window — in practice SwiftUI always has at least one active window
      // by the time the user taps Continue.
      #if os(iOS)
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
      #else
      return NSApplication.shared.keyWindow
        ?? NSApplication.shared.mainWindow
        ?? NSApplication.shared.windows.first
        ?? ASPresentationAnchor()
      #endif
    }
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
