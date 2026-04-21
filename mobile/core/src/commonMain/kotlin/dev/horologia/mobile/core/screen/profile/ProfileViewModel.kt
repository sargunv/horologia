package dev.horologia.mobile.core.screen.profile

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

class ProfileViewModel(private val gateway: ProfileGateway) : ViewModel() {
  private val _uiState = MutableStateFlow<ProfileUiState>(ProfileUiState.Loading)
  val uiState: StateFlow<ProfileUiState> = _uiState.asStateFlow()

  private var fetchJob: Job? = null

  init {
    refresh()
  }

  fun refresh() {
    fetchJob?.cancel()
    _uiState.value = ProfileUiState.Loading
    fetchJob = viewModelScope.launch {
      _uiState.value =
        when (val result = gateway.fetchMe()) {
          is FetchProfileResult.Ok -> ProfileUiState.Success(displayName = result.displayName)

          FetchProfileResult.AuthFailure ->
            ProfileUiState.Error(
              message = "Authentication failed. Check the dev token.",
              retryable = false,
            )

          is FetchProfileResult.Retryable ->
            ProfileUiState.Error(message = result.message, retryable = true)

          is FetchProfileResult.Permanent ->
            ProfileUiState.Error(message = result.message, retryable = false)
        }
    }
  }
}
