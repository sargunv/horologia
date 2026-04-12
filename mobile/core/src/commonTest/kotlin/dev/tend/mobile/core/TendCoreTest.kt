package dev.tend.mobile.core

import kotlin.test.Test
import kotlin.test.assertEquals

class TendCoreTest {
  @Test
  fun statusLineIsStable() {
    assertEquals("Tend mobile scaffold", TendCore.statusLine())
  }
}
