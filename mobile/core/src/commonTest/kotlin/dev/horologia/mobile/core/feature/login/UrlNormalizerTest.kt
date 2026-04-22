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
}
