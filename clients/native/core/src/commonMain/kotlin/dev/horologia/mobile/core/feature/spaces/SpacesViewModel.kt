package dev.horologia.mobile.core.feature.spaces

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

class SpacesViewModel internal constructor(private val gateway: SpacesGateway) : ViewModel() {
  private val _uiState = MutableStateFlow<SpacesUiState>(SpacesUiState.Loading)
  val uiState: StateFlow<SpacesUiState> = _uiState.asStateFlow()

  private var fetchJob: Job? = null

  init {
    refresh()
  }

  fun refresh() {
    fetchJob?.cancel()
    _uiState.value = SpacesUiState.Loading
    fetchJob = viewModelScope.launch {
      _uiState.value =
        when (val result = gateway.fetchSpaces()) {
          is FetchSpacesResult.Ok -> SpacesUiState.Success(items = result.items)

          FetchSpacesResult.AuthFailure ->
            SpacesUiState.Error(
              message = "Authentication failed. Check the dev token.",
              retryable = false,
            )

          is FetchSpacesResult.Retryable ->
            SpacesUiState.Error(message = result.message, retryable = true)

          is FetchSpacesResult.Permanent ->
            SpacesUiState.Error(message = result.message, retryable = false)
        }
    }
  }
}
