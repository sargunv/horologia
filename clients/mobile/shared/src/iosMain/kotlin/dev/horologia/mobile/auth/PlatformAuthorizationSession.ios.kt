@file:OptIn(kotlinx.cinterop.ExperimentalForeignApi::class)

package dev.horologia.mobile.auth

import kotlin.coroutines.resume
import kotlin.coroutines.resumeWithException
import kotlinx.cinterop.addressOf
import kotlinx.cinterop.usePinned
import kotlinx.coroutines.suspendCancellableCoroutine
import platform.AuthenticationServices.ASPresentationAnchor
import platform.AuthenticationServices.ASWebAuthenticationPresentationContextProvidingProtocol
import platform.AuthenticationServices.ASWebAuthenticationSession
import platform.Foundation.NSDate
import platform.CoreFoundation.kCFAbsoluteTimeIntervalSince1970
import platform.Foundation.NSURL
import platform.Security.SecRandomCopyBytes
import platform.Security.errSecSuccess
import platform.Security.kSecRandomDefault
import platform.UIKit.UIApplication
import platform.darwin.NSObject

actual class PlatformAuthorizationSession actual constructor() : AuthorizationSession {
    private val provider = PresentationContextProvider()
    actual override suspend fun authorize(authorizationUrl: String, callbackScheme: String): String =
        suspendCancellableCoroutine { continuation ->
            val url = NSURL.URLWithString(authorizationUrl)
            if (url == null) {
                continuation.resumeWithException(OAuthException("Invalid authorization URL"))
                return@suspendCancellableCoroutine
            }
            val session = ASWebAuthenticationSession(url, callbackScheme) { callback, error ->
                when {
                    callback != null && continuation.isActive -> continuation.resume(callback.absoluteString ?: callback.toString())
                    error != null && continuation.isActive -> continuation.resumeWithException(
                        OAuthException(error.localizedDescription, cause = null),
                    )
                    continuation.isActive -> continuation.resumeWithException(OAuthException("Authorization session returned no callback"))
                }
            }
            session.presentationContextProvider = provider
            session.prefersEphemeralWebBrowserSession = false
            if (!session.start()) {
                continuation.resumeWithException(OAuthException("Could not start authorization session"))
                return@suspendCancellableCoroutine
            }
            continuation.invokeOnCancellation { session.cancel() }
        }
}

private class PresentationContextProvider : NSObject(), ASWebAuthenticationPresentationContextProvidingProtocol {
    override fun presentationAnchorForWebAuthenticationSession(session: ASWebAuthenticationSession): ASPresentationAnchor =
        UIApplication.sharedApplication.keyWindow
            ?: throw OAuthException("No window is available to present authorization")
}

internal actual fun secureRandomBytes(size: Int): ByteArray = ByteArray(size).also { bytes ->
    val status = bytes.usePinned { pinned -> SecRandomCopyBytes(kSecRandomDefault, size.toULong(), pinned.addressOf(0)) }
    check(status == errSecSuccess) { "Secure random generation failed (OSStatus $status)" }
}


internal actual fun currentEpochSeconds(): Long =
    (NSDate().timeIntervalSinceReferenceDate + kCFAbsoluteTimeIntervalSince1970).toLong()
