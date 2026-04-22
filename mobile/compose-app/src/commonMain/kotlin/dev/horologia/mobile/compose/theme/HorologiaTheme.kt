package dev.horologia.mobile.compose.theme

import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable

/**
 * Compose theme wrapper for the Horologia mobile app.
 *
 * Material 3 Expressive was considered (per R21) — the design spec's § I / J call for wavy
 * [LoadingIndicator], expressive shape motion, and `MaterialExpressiveTheme`. As of 2026-04-21
 * (Compose Multiplatform 1.9.0, verified via
 * `https://github.com/JetBrains/compose-multiplatform/blob/master/CHANGELOG.md` WebFetch during
 * implementation) the JetBrains CMP artifact at this version does NOT ship the expressive API
 * surface — no `MaterialExpressiveTheme`, no wavy `LoadingIndicator`. Mixing androidx `material3`
 * 1.4+ (where expressive lives) with CMP would require Android-only code paths and a rewrite of the
 * theme binding; out of scope for this task.
 *
 * Consequence: this wrapper currently delegates to baseline `MaterialTheme`. When CMP catches up,
 * swap the body for `MaterialExpressiveTheme { content() }` and the call sites don't move.
 */
@Composable
fun HorologiaTheme(content: @Composable () -> Unit) {
  MaterialTheme { content() }
}
