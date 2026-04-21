package dev.horologia.mobile.compose

import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.remember
import androidx.lifecycle.ViewModel
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.ViewModelStore
import kotlin.reflect.KClass

/**
 * Remembers a [ViewModel] obtained from [factory] on the desktop target. A per-call
 * [ViewModelStore] is created so the ViewModel's `onCleared` fires when the composable leaves the
 * composition.
 *
 * The Android entry point uses `androidx.lifecycle.viewmodel.compose.viewModel` instead, which
 * scopes the store to the host Activity/Fragment.
 */
@Composable
fun <VM : ViewModel> rememberViewModel(
  modelClass: KClass<VM>,
  factory: ViewModelProvider.Factory,
): VM {
  val store = remember { ViewModelStore() }
  DisposableEffect(store) { onDispose { store.clear() } }
  return remember(store, factory) { ViewModelProvider.create(store, factory)[modelClass] }
}
