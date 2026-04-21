package dev.horologia.mobile.core.screen.profile

sealed interface ProfileUiState {
  data object Loading : ProfileUiState

  data class Success(val displayName: String) : ProfileUiState

  data class Error(val message: String, val retryable: Boolean) : ProfileUiState
}
