package dev.horologia.mobile.core.feature.spaces

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
class SpacesViewModelTest {
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
    val items =
      listOf(
        SpacesListItem(slug = "team-a", name = "Team A"),
        SpacesListItem(slug = "team-b", name = "Team B"),
      )
    val gateway = FakeSpacesGateway { FetchSpacesResult.Ok(items = items) }

    val viewModel = SpacesViewModel(gateway)

    assertEquals(SpacesUiState.Loading, viewModel.uiState.value)
    testScheduler.advanceUntilIdle()
    assertEquals(SpacesUiState.Success(items), viewModel.uiState.value)
  }

  @Test
  fun loadingThenRetryableError() = runTest {
    val gateway = FakeSpacesGateway { FetchSpacesResult.Retryable(message = "connection refused") }

    val viewModel = SpacesViewModel(gateway)

    testScheduler.advanceUntilIdle()
    val state = viewModel.uiState.value
    assertTrue(state is SpacesUiState.Error, "expected Error, got $state")
    assertEquals("connection refused", state.message)
    assertTrue(state.retryable, "network errors should be retryable")
  }

  @Test
  fun loadingThenPermanentErrorIsNotRetryable() = runTest {
    val gateway = FakeSpacesGateway {
      FetchSpacesResult.Permanent(message = "Response format mismatch")
    }

    val viewModel = SpacesViewModel(gateway)

    testScheduler.advanceUntilIdle()
    val state = viewModel.uiState.value
    assertTrue(state is SpacesUiState.Error, "expected Error, got $state")
    assertEquals("Response format mismatch", state.message)
    assertFalse(state.retryable, "permanent errors must not be retryable")
  }

  @Test
  fun refreshAfterErrorCyclesBackToLoadingThenSuccess() = runTest {
    val items = listOf(SpacesListItem(slug = "team-a", name = "Team A"))
    var callCount = 0
    val gateway = FakeSpacesGateway {
      when (callCount++) {
        0 -> FetchSpacesResult.Retryable(message = "network blip")
        else -> FetchSpacesResult.Ok(items = items)
      }
    }

    val viewModel = SpacesViewModel(gateway)
    testScheduler.advanceUntilIdle()
    val firstState = viewModel.uiState.value
    assertTrue(firstState is SpacesUiState.Error, "first call should error, got $firstState")

    viewModel.refresh()
    assertEquals(SpacesUiState.Loading, viewModel.uiState.value)
    testScheduler.advanceUntilIdle()
    assertEquals(SpacesUiState.Success(items), viewModel.uiState.value)
    assertEquals(2, callCount, "refresh() should re-invoke the gateway")
  }

  @Test
  fun concurrentRefreshCancelsInFlightCall() = runTest {
    val gate = CompletableDeferred<FetchSpacesResult>()
    val freshItems = listOf(SpacesListItem(slug = "fresh", name = "Fresh"))
    var callCount = 0
    val gateway = FakeSpacesGateway {
      when (callCount++) {
        0 -> gate.await()
        else -> FetchSpacesResult.Ok(items = freshItems)
      }
    }

    val viewModel = SpacesViewModel(gateway)
    // Let the init-time refresh reach gate.await before firing a second refresh.
    testScheduler.advanceUntilIdle()
    assertEquals(SpacesUiState.Loading, viewModel.uiState.value)
    assertEquals(1, callCount, "init should have invoked the gateway once")

    viewModel.refresh()
    testScheduler.advanceUntilIdle()
    assertEquals(SpacesUiState.Success(freshItems), viewModel.uiState.value)
    assertEquals(2, callCount, "second refresh should have invoked the gateway again")

    // Completing the now-cancelled first call should NOT overwrite the fresh result.
    val stale = listOf(SpacesListItem(slug = "stale", name = "Stale"))
    gate.complete(FetchSpacesResult.Ok(items = stale))
    testScheduler.advanceUntilIdle()
    assertEquals(SpacesUiState.Success(freshItems), viewModel.uiState.value)
  }

  private class FakeSpacesGateway(private val result: suspend () -> FetchSpacesResult) :
    SpacesGateway {
    override suspend fun fetchSpaces(): FetchSpacesResult = result()
  }
}
