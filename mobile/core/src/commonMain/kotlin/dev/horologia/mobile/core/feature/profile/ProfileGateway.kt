package dev.horologia.mobile.core.feature.profile

import dev.horologia.mobile.core.net.Failure
import dev.horologia.mobile.core.net.classify
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

internal class LiveProfileGateway(private val requestTimeout: Duration = 15.seconds) :
  ProfileGateway {
  override suspend fun fetchMe(): FetchProfileResult =
    try {
      withTimeout(requestTimeout) {
        UsersApi.usersMe()
          .fold(
            ifRight = { response -> FetchProfileResult.Ok(displayName = response.data.name) },
            ifLeft = { exception ->
              when (val failure = exception.classify()) {
                Failure.Auth -> FetchProfileResult.AuthFailure
                is Failure.Retryable -> FetchProfileResult.Retryable(message = failure.message)
                is Failure.Permanent -> FetchProfileResult.Permanent(message = failure.message)
              }
            },
          )
      }
    } catch (_: TimeoutCancellationException) {
      FetchProfileResult.Retryable(message = "Request timed out after $requestTimeout.")
    }
}
