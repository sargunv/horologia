package dev.horologia.mobile.core.net

import com.kroegerama.openapi.kmp.gen.companion.CallException
import com.kroegerama.openapi.kmp.gen.companion.HttpCallException
import com.kroegerama.openapi.kmp.gen.companion.IOCallException
import com.kroegerama.openapi.kmp.gen.companion.SerializationException

/**
 * Common classification that every feature-gateway needs when it converts a [CallException] from
 * the generated OpenAPI client into a user-facing error state. Each feature owns its own result
 * type (e.g. `FetchSpacesResult`, `FetchProfileResult`) and maps [Failure] cases onto that type —
 * but the classification itself is the same everywhere, so it lives here.
 */
internal sealed interface Failure {
  data object Auth : Failure

  data class Retryable(val message: String) : Failure

  data class Permanent(val message: String) : Failure
}

private val authFailureCodes = setOf(401, 403)

/**
 * Classifies a [CallException] into one of the common [Failure] buckets. Gateways are expected to
 * wrap this call in their own `try { withTimeout { ... } }` block and handle
 * `TimeoutCancellationException` separately, since timeouts aren't represented as a
 * `CallException`.
 */
internal fun CallException.classify(): Failure =
  when (this) {
    is HttpCallException ->
      if (code in authFailureCodes) {
        Failure.Auth
      } else {
        // Hide the server-provided body from the user: it could contain stack
        // traces or internal paths we shouldn't surface.
        Failure.Retryable(message = "Server error (HTTP $code).")
      }

    is IOCallException -> Failure.Retryable(message = message ?: "Network error")

    is SerializationException ->
      Failure.Permanent(message = "Response format mismatch — the app may be out of date.")

    else -> Failure.Retryable(message = message ?: "Unknown error")
  }
