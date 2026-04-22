package dev.horologia.mobile.core.feature.login

import kotlin.test.Test
import kotlin.test.assertEquals

class UrlNormalizerTest {
  // A bare host becomes two candidates: https-first, http-fallback. Probing tries them in order.
  @Test
  fun bareHostProducesHttpsAndHttpCandidates() {
    assertEquals(
      listOf("https://tasks.example.com", "http://tasks.example.com"),
      UrlNormalizer.candidates("tasks.example.com"),
    )
  }

  @Test
  fun explicitHttpIsPreservedAndSingleCandidate() {
    assertEquals(listOf("http://localhost:8080"), UrlNormalizer.candidates("http://localhost:8080"))
  }

  @Test
  fun explicitHttpsIsPreservedAndSingleCandidate() {
    assertEquals(
      listOf("https://tasks.example.com"),
      UrlNormalizer.candidates("https://tasks.example.com"),
    )
  }

  @Test
  fun whitespaceIsTrimmed() {
    assertEquals(
      listOf("https://tasks.example.com", "http://tasks.example.com"),
      UrlNormalizer.candidates("  tasks.example.com  "),
    )
  }

  @Test
  fun trailingSlashOnHostOnlyIsStripped() {
    assertEquals(
      listOf("https://tasks.example.com"),
      UrlNormalizer.candidates("https://tasks.example.com/"),
    )
  }

  @Test
  fun pathIsPreservedVerbatim() {
    assertEquals(
      listOf("https://tasks.example.com/api/"),
      UrlNormalizer.candidates("https://tasks.example.com/api/"),
    )
  }

  @Test
  fun emptyReturnsEmptyList() {
    assertEquals(emptyList(), UrlNormalizer.candidates(""))
    assertEquals(emptyList(), UrlNormalizer.candidates("   "))
  }

  @Test
  fun hostWithPortIsPreserved() {
    assertEquals(
      listOf("https://tasks.example.com:8443", "http://tasks.example.com:8443"),
      UrlNormalizer.candidates("tasks.example.com:8443"),
    )
  }

  @Test
  fun bareIpv4IsAccepted() {
    assertEquals(
      listOf("https://192.168.1.10", "http://192.168.1.10"),
      UrlNormalizer.candidates("192.168.1.10"),
    )
  }

  @Test
  fun userinfoIsPreserved() {
    assertEquals(
      listOf("https://user:pass@tasks.example.com"),
      UrlNormalizer.candidates("https://user:pass@tasks.example.com"),
    )
  }

  @Test
  fun uppercaseSchemeIsPreserved() {
    // Ktor lower-cases the scheme during parse, but our normalizer returns the
    // verbatim input once it parses — so the stored URL retains the caller's shape.
    assertEquals(
      listOf("HTTPS://tasks.example.com"),
      UrlNormalizer.candidates("HTTPS://tasks.example.com"),
    )
  }
}
