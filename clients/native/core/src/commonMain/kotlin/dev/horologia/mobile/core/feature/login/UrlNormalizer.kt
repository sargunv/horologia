package dev.horologia.mobile.core.feature.login

import io.ktor.http.Url

/**
 * Normalizes raw server-URL input the user typed into the login picker:
 * - trims whitespace
 * - defaults a missing scheme by probing both `https://` and `http://` (callers try them in order)
 * - strips a single trailing slash when the input is host-only (preserves `.../api/` paths)
 * - returns `null` candidates that don't parse as a URL
 *
 * Per R3 the app performs no network-range validation — any host is accepted as long as it parses.
 */
object UrlNormalizer {
  /**
   * Returns the ordered list of URLs the probe should try. When the user typed an explicit scheme
   * we honour it exactly (single candidate). When the scheme is missing we try `https://` first and
   * fall back to `http://` — so bare `localhost:8080`, `192.168.1.5:8080`, or `my.lan:9000` all
   * work against cleartext servers without the user hand-typing a scheme, while a public host like
   * `tasks.example.com` still prefers TLS. [LoginViewModel] uses whichever candidate answers the
   * probe as the resolved URL for the rest of the flow.
   */
  fun candidates(raw: String): List<String> {
    val trimmed = raw.trim()
    if (trimmed.isEmpty()) return emptyList()

    val withSchemes =
      if (trimmed.contains("://")) listOf(trimmed)
      else listOf("https://$trimmed", "http://$trimmed")

    return withSchemes.mapNotNull { candidate ->
      val parsed = runCatching { Url(candidate) }.getOrNull() ?: return@mapNotNull null
      // Strip a single trailing slash only when the user typed a bare host — i.e. Ktor parsed
      // the path as empty or just `/`. `.../api/` etc. is preserved verbatim because its
      // encodedPath is non-trivial.
      val hostOnly = parsed.encodedPath.isEmpty() || parsed.encodedPath == "/"
      if (hostOnly && candidate.endsWith("/")) candidate.dropLast(1) else candidate
    }
  }
}
