package dev.horologia.mobile.auth

import android.content.Intent
import android.net.Uri
import androidx.browser.customtabs.CustomTabsIntent
import java.security.SecureRandom
import java.util.concurrent.atomic.AtomicReference
import kotlin.coroutines.resume
import kotlin.coroutines.resumeWithException
import kotlinx.coroutines.suspendCancellableCoroutine

actual class PlatformAuthorizationSession actual constructor() : AuthorizationSession {
    actual override suspend fun authorize(authorizationUrl: String, callbackScheme: String): String =
        suspendCancellableCoroutine { continuation ->
            val request = AndroidAuthorizationHandoff.Request(callbackScheme) { result ->
                result.fold(
                    onSuccess = { if (continuation.isActive) continuation.resume(it) },
                    onFailure = { if (continuation.isActive) continuation.resumeWithException(it) },
                )
            }
            if (!AndroidAuthorizationHandoff.begin(request, authorizationUrl)) {
                continuation.resumeWithException(OAuthException("No Android authorization launch handler is installed, or another authorization is active"))
                return@suspendCancellableCoroutine
            }
            continuation.invokeOnCancellation { AndroidAuthorizationHandoff.cancel(request) }
        }
}

/**
 * Activity-side bridge for Custom Tabs. Install a launch handler while the Activity is started,
 * and pass deep-link intents from onCreate/onNewIntent to [handleCallback]. The bridge stores no Activity.
 */
object AndroidAuthorizationHandoff {
    internal class Request(
        val callbackScheme: String,
        val complete: (Result<String>) -> Unit,
    )

    private val launchHandler = AtomicReference<((Intent) -> Unit)?>(null)
    private val pending = AtomicReference<Request?>(null)

    fun installLaunchHandler(handler: (Intent) -> Unit): AutoCloseable {
        check(launchHandler.compareAndSet(null, handler)) { "An authorization launch handler is already installed" }
        return AutoCloseable { launchHandler.compareAndSet(handler, null) }
    }

    fun handleCallback(intent: Intent): Boolean {
        val callback = intent.data ?: return false
        val request = pending.get() ?: return false
        if (callback.scheme != request.callbackScheme || callback.host != "oauth" || callback.path != "/callback") return false
        if (!pending.compareAndSet(request, null)) return false
        request.complete(Result.success(callback.toString()))
        return true
    }

    internal fun begin(request: Request, authorizationUrl: String): Boolean {
        val handler = launchHandler.get() ?: return false
        if (!pending.compareAndSet(null, request)) return false
        val intent = CustomTabsIntent.Builder().setShareState(CustomTabsIntent.SHARE_STATE_OFF).build().intent.apply {
            data = Uri.parse(authorizationUrl)
        }
        return try {
            handler(intent)
            true
        } catch (error: Throwable) {
            pending.compareAndSet(request, null)
            request.complete(Result.failure(error))
            true
        }
    }

    internal fun cancel(request: Request) {
        pending.compareAndSet(request, null)
    }
}

internal actual fun secureRandomBytes(size: Int): ByteArray = ByteArray(size).also(SecureRandom()::nextBytes)
internal actual fun currentEpochSeconds(): Long = System.currentTimeMillis() / 1_000L
