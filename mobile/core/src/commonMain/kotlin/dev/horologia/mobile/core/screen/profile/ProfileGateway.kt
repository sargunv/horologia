package dev.horologia.mobile.core.screen.profile

import com.kroegerama.openapi.kmp.gen.companion.HttpCallException
import com.kroegerama.openapi.kmp.gen.companion.IOCallException
import dev.horologia.mobile.generated.api.UsersApi

interface ProfileGateway {
  suspend fun fetchMe(): FetchProfileResult
}

sealed interface FetchProfileResult {
  data class Ok(val displayName: String) : FetchProfileResult

  data object AuthFailure : FetchProfileResult

  data class Retryable(val message: String) : FetchProfileResult
}

class LiveProfileGateway : ProfileGateway {
  override suspend fun fetchMe(): FetchProfileResult =
    UsersApi.usersMe()
      .fold(
        ifRight = { response -> FetchProfileResult.Ok(displayName = response.data.name) },
        ifLeft = { exception ->
          when (exception) {
            is HttpCallException ->
              if (exception.code in AUTH_FAILURE_CODES) {
                FetchProfileResult.AuthFailure
              } else {
                FetchProfileResult.Retryable(
                  message = "HTTP ${exception.code}: ${exception.message}"
                )
              }

            is IOCallException ->
              FetchProfileResult.Retryable(message = exception.message ?: "Network error")

            else -> FetchProfileResult.Retryable(message = exception.message ?: "Unknown error")
          }
        },
      )

  private companion object {
    val AUTH_FAILURE_CODES = setOf(401, 403)
  }
}
