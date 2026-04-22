package dev.horologia.mobile.core.feature.login

import io.ktor.http.Url

/**
 * Defaults bare hosts to `https://`, honours explicit `http://` / `https://`, trims whitespace,
 * strips a single trailing slash on host-only input, and returns `null` if the result doesn't parse
 * as a URL.
 *
 * Per R3 the app performs no network-range validation — any host is accepted as long as it parses.
 * An empty / all-whitespace input returns `null`.
 */
object UrlNormalizer {
  fun normalize(raw: String): String? {
    val trimmed = raw.trim()
    if (trimmed.isEmpty()) return null

    val withScheme = if (trimmed.contains("://")) trimmed else "https://$trimmed"

    // Strip a single trailing slash only when the user typed a bare host
    // (no path) — preserves `.../api/` style paths verbatim.
    val withoutTrailingSlash =
      if (withScheme.endsWith("/") && withScheme.count { it == '/' } == 3) {
        withScheme.dropLast(1)
      } else {
        withScheme
      }

    return try {
      Url(withoutTrailingSlash)
      withoutTrailingSlash
    } catch (_: IllegalArgumentException) {
      null
    }
  }
}
