package dev.horologia.mobile.compose

import androidx.compose.runtime.staticCompositionLocalOf
import dev.horologia.mobile.core.AppContainer

/**
 * Carries the per-process [AppContainer] down the composition tree so screens can resolve their own
 * ViewModels via `viewModel<T>(factory = LocalAppContainer.current.xxxFactory)` without taking the
 * container as a parameter.
 *
 * `staticCompositionLocalOf` is correct here: the container is installed once by [HorologiaApp] and
 * never changes, so we skip the recomposition tracking of the regular variant.
 */
val LocalAppContainer =
  staticCompositionLocalOf<AppContainer> { error("AppContainer not provided") }
