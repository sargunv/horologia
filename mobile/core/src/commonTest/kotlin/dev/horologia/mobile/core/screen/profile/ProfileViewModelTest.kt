package dev.horologia.mobile.core.screen.profile

import kotlin.test.AfterTest
import kotlin.test.BeforeTest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertTrue
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
  fun loadingThenSuccess() =
    runTest(dispatcher) {
      val gateway = FakeProfileGateway { FetchProfileResult.Ok(displayName = "Alice") }

      val viewModel = ProfileViewModel(gateway)

      assertEquals(ProfileUiState.Loading, viewModel.uiState.value)
      testScheduler.advanceUntilIdle()
      assertEquals(ProfileUiState.Success("Alice"), viewModel.uiState.value)
    }

  @Test
  fun loadingThenRetryableError() =
    runTest(dispatcher) {
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
  fun loadingThenAuthFailureIsNotRetryable() =
    runTest(dispatcher) {
      val gateway = FakeProfileGateway { FetchProfileResult.AuthFailure }

      val viewModel = ProfileViewModel(gateway)

      testScheduler.advanceUntilIdle()
      val state = viewModel.uiState.value
      assertTrue(state is ProfileUiState.Error, "expected Error, got $state")
      assertEquals("Authentication failed. Check the dev token.", state.message)
      assertFalse(state.retryable, "auth failure must not be retryable")
    }

  @Test
  fun loadingThenPermanentErrorIsNotRetryable() =
    runTest(dispatcher) {
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

  private class FakeProfileGateway(private val result: suspend () -> FetchProfileResult) :
    ProfileGateway {
    override suspend fun fetchMe(): FetchProfileResult = result()
  }
}
