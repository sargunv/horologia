package dev.horologia.mobile.core.platform

import java.security.MessageDigest
import java.security.SecureRandom

actual object Crypto {
  private val random = SecureRandom()

  actual fun sha256(input: ByteArray): ByteArray =
    MessageDigest.getInstance("SHA-256").digest(input)

  actual fun randomBytes(size: Int): ByteArray {
    val out = ByteArray(size)
    random.nextBytes(out)
    return out
  }
}
