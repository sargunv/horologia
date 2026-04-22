package dev.horologia.mobile.core.feature.spaces

import dev.horologia.mobile.core.net.Failure
import dev.horologia.mobile.core.net.classify
import dev.horologia.mobile.generated.api.SpacesApi
import kotlin.time.Duration
import kotlin.time.Duration.Companion.seconds
import kotlinx.coroutines.TimeoutCancellationException
import kotlinx.coroutines.withTimeout

internal interface SpacesGateway {
  suspend fun fetchSpaces(): FetchSpacesResult
}

internal sealed interface FetchSpacesResult {
  data class Ok(val items: List<SpacesListItem>) : FetchSpacesResult

  data object AuthFailure : FetchSpacesResult

  data class Retryable(val message: String) : FetchSpacesResult

  data class Permanent(val message: String) : FetchSpacesResult
}

internal class LiveSpacesGateway(private val requestTimeout: Duration = 15.seconds) :
  SpacesGateway {
  override suspend fun fetchSpaces(): FetchSpacesResult =
    try {
      withTimeout(requestTimeout) {
        SpacesApi.spacesList()
          .fold(
            ifRight = { response ->
              FetchSpacesResult.Ok(
                items =
                  response.data.items.map { space ->
                    SpacesListItem(slug = space.slug, name = space.name)
                  }
              )
            },
            ifLeft = { exception ->
              when (val failure = exception.classify()) {
                Failure.Auth -> FetchSpacesResult.AuthFailure
                is Failure.Retryable -> FetchSpacesResult.Retryable(message = failure.message)
                is Failure.Permanent -> FetchSpacesResult.Permanent(message = failure.message)
              }
            },
          )
      }
    } catch (_: TimeoutCancellationException) {
      FetchSpacesResult.Retryable(message = "Request timed out after $requestTimeout.")
    }
}
