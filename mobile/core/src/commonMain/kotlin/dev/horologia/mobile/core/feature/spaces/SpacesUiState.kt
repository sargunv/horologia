package dev.horologia.mobile.core.feature.spaces

sealed interface SpacesUiState {
  data object Loading : SpacesUiState

  data class Success(val items: List<SpacesListItem>) : SpacesUiState

  data class Error(val message: String, val retryable: Boolean) : SpacesUiState
}

data class SpacesListItem(val slug: String, val name: String)
