package dev.horologia.mobile.core.screen.profile

import kotlin.test.AfterTest
import kotlin.test.BeforeTest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertTrue
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain

@OptIn(ExperimentalCoroutinesApi::class)
class ProfileViewModelTest {
  private val dispatcher = StandardTestDispatcher()

  @BeforeTest
  fun setUp() {
    Dispatchers.setMain(dispatcher)
  }

  @AfterTest
  fun tearDown() {
    Dispatchers.resetMain()
  }

  @Test
  fun loadingThenSuccess() = runTest {
    val gateway = FakeProfileGateway { FetchProfileResult.Ok(displayName = "Alice") }

    val viewModel = ProfileViewModel(gateway)

    assertEquals(ProfileUiState.Loading, viewModel.uiState.value)
    testScheduler.advanceUntilIdle()
    assertEquals(ProfileUiState.Success("Alice"), viewModel.uiState.value)
  }

  @Test
  fun loadingThenRetryableError() = runTest {
    val gateway = FakeProfileGateway {
      FetchProfileResult.Retryable(message = "connection refused")
    }

    val viewModel = ProfileViewModel(gateway)

    testScheduler.advanceUntilIdle()
    val state = viewModel.uiState.value
    assertTrue(state is ProfileUiState.Error, "expected Error, got $state")
    assertEquals("connection refused", state.message)
    assertTrue(state.retryable, "network errors should be retryable")
  }

  @Test
  fun loadingThenAuthFailureIsNotRetryable() = runTest {
    val gateway = FakeProfileGateway { FetchProfileResult.AuthFailure }

    val viewModel = ProfileViewModel(gateway)

    testScheduler.advanceUntilIdle()
    val state = viewModel.uiState.value
    assertTrue(state is ProfileUiState.Error, "expected Error, got $state")
    assertEquals("Authentication failed. Check the dev token.", state.message)
    assertFalse(state.retryable, "auth failure must not be retryable")
  }

  @Test
  fun loadingThenPermanentErrorIsNotRetryable() = runTest {
    val gateway = FakeProfileGateway {
      FetchProfileResult.Permanent(message = "Response format mismatch")
    }

    val viewModel = ProfileViewModel(gateway)

    testScheduler.advanceUntilIdle()
    val state = viewModel.uiState.value
    assertTrue(state is ProfileUiState.Error, "expected Error, got $state")
    assertEquals("Response format mismatch", state.message)
    assertFalse(state.retryable, "permanent errors must not be retryable")
  }

  @Test
  fun refreshAfterErrorCyclesBackToLoadingThenSuccess() = runTest {
    var callCount = 0
    val gateway = FakeProfileGateway {
      when (callCount++) {
        0 -> FetchProfileResult.Retryable(message = "network blip")
        else -> FetchProfileResult.Ok(displayName = "Alice")
      }
    }

    val viewModel = ProfileViewModel(gateway)
    testScheduler.advanceUntilIdle()
    val firstState = viewModel.uiState.value
    assertTrue(firstState is ProfileUiState.Error, "first call should error, got $firstState")

    viewModel.refresh()
    assertEquals(ProfileUiState.Loading, viewModel.uiState.value)
    testScheduler.advanceUntilIdle()
    assertEquals(ProfileUiState.Success("Alice"), viewModel.uiState.value)
    assertEquals(2, callCount, "refresh() should re-invoke the gateway")
  }

  @Test
  fun concurrentRefreshCancelsInFlightCall() = runTest {
    val gate = CompletableDeferred<FetchProfileResult>()
    var callCount = 0
    val gateway = FakeProfileGateway {
      when (callCount++) {
        0 -> gate.await()
        else -> FetchProfileResult.Ok(displayName = "Bob")
      }
    }

    val viewModel = ProfileViewModel(gateway)
    // Let the init-time refresh reach gate.await before firing a second refresh.
    testScheduler.advanceUntilIdle()
    assertEquals(ProfileUiState.Loading, viewModel.uiState.value)
    assertEquals(1, callCount, "init should have invoked the gateway once")

    viewModel.refresh()
    testScheduler.advanceUntilIdle()
    assertEquals(ProfileUiState.Success("Bob"), viewModel.uiState.value)
    assertEquals(2, callCount, "second refresh should have invoked the gateway again")

    // Completing the now-cancelled first call should NOT overwrite Bob.
    gate.complete(FetchProfileResult.Retryable(message = "stale blip"))
    testScheduler.advanceUntilIdle()
    assertEquals(ProfileUiState.Success("Bob"), viewModel.uiState.value)
  }

  private class FakeProfileGateway(private val result: suspend () -> FetchProfileResult) :
    ProfileGateway {
    override suspend fun fetchMe(): FetchProfileResult = result()
  }
}
