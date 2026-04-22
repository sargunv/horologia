package dev.horologia.mobile.core.feature.login

import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull

class UrlNormalizerTest {
  @Test
  fun bareHostDefaultsToHttps() {
    assertEquals("https://tasks.example.com", UrlNormalizer.normalize("tasks.example.com"))
  }

  @Test
  fun explicitHttpIsPreserved() {
    assertEquals("http://localhost:8080", UrlNormalizer.normalize("http://localhost:8080"))
  }

  @Test
  fun explicitHttpsIsPreserved() {
    assertEquals("https://tasks.example.com", UrlNormalizer.normalize("https://tasks.example.com"))
  }

  @Test
  fun whitespaceIsTrimmed() {
    assertEquals("https://tasks.example.com", UrlNormalizer.normalize("  tasks.example.com  "))
  }

  @Test
  fun trailingSlashOnHostOnlyIsStripped() {
    assertEquals("https://tasks.example.com", UrlNormalizer.normalize("https://tasks.example.com/"))
  }

  @Test
  fun pathIsPreservedVerbatim() {
    assertEquals(
      "https://tasks.example.com/api/",
      UrlNormalizer.normalize("https://tasks.example.com/api/"),
    )
  }

  @Test
  fun emptyReturnsNull() {
    assertNull(UrlNormalizer.normalize(""))
    assertNull(UrlNormalizer.normalize("   "))
  }

  @Test
  fun hostWithPortIsPreserved() {
    assertEquals(
      "https://tasks.example.com:8443",
      UrlNormalizer.normalize("tasks.example.com:8443"),
    )
  }

  @Test
  fun bareIpv4IsAccepted() {
    assertEquals("https://192.168.1.10", UrlNormalizer.normalize("192.168.1.10"))
  }

  @Test
  fun userinfoIsPreserved() {
    assertEquals(
      "https://user:pass@tasks.example.com",
      UrlNormalizer.normalize("https://user:pass@tasks.example.com"),
    )
  }

  @Test
  fun uppercaseSchemeIsPreserved() {
    // Ktor lower-cases the scheme during parse, but our normalizer returns the
    // verbatim input once it parses — so the stored URL retains the caller's shape.
    assertEquals("HTTPS://tasks.example.com", UrlNormalizer.normalize("HTTPS://tasks.example.com"))
  }
}
