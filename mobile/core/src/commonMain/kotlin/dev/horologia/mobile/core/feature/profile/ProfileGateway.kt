package dev.horologia.mobile.core.feature.profile

import com.kroegerama.openapi.kmp.gen.companion.HttpCallException
import com.kroegerama.openapi.kmp.gen.companion.IOCallException
import com.kroegerama.openapi.kmp.gen.companion.SerializationException
import dev.horologia.mobile.generated.api.UsersApi
import kotlin.time.Duration
import kotlin.time.Duration.Companion.seconds
import kotlinx.coroutines.TimeoutCancellationException
import kotlinx.coroutines.withTimeout

internal interface ProfileGateway {
  suspend fun fetchMe(): FetchProfileResult
}

internal sealed interface FetchProfileResult {
  data class Ok(val displayName: String) : FetchProfileResult

  data object AuthFailure : FetchProfileResult

  data class Retryable(val message: String) : FetchProfileResult

  data class Permanent(val message: String) : FetchProfileResult
}

private val authFailureCodes = setOf(401, 403)

internal class LiveProfileGateway(private val requestTimeout: Duration = 15.seconds) :
  ProfileGateway {
  override suspend fun fetchMe(): FetchProfileResult =
    try {
      withTimeout(requestTimeout) {
        UsersApi.usersMe()
          .fold(
            ifRight = { response -> FetchProfileResult.Ok(displayName = response.data.name) },
            ifLeft = { exception -> mapCallException(exception) },
          )
      }
    } catch (_: TimeoutCancellationException) {
      FetchProfileResult.Retryable(message = "Request timed out after $requestTimeout.")
    }

  private fun mapCallException(
    exception: com.kroegerama.openapi.kmp.gen.companion.CallException
  ): FetchProfileResult =
    when (exception) {
      is HttpCallException ->
        if (exception.code in authFailureCodes) {
          FetchProfileResult.AuthFailure
        } else {
          // Show a generic user-facing message and hide the server-provided body: it
          // could contain stack traces or internal paths we shouldn't surface.
          FetchProfileResult.Retryable(message = "Server error (HTTP ${exception.code}).")
        }

      is IOCallException ->
        FetchProfileResult.Retryable(message = exception.message ?: "Network error")

      is SerializationException ->
        FetchProfileResult.Permanent(
          message = "Response format mismatch — the app may be out of date."
        )

      else -> FetchProfileResult.Retryable(message = exception.message ?: "Unknown error")
    }
}
