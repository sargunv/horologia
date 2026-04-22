package dev.horologia.mobile.core.platform

import kotlinx.cinterop.ExperimentalForeignApi
import kotlinx.cinterop.UByteVar
import kotlinx.cinterop.addressOf
import kotlinx.cinterop.convert
import kotlinx.cinterop.reinterpret
import kotlinx.cinterop.usePinned
import platform.CoreCrypto.CC_SHA256
import platform.CoreCrypto.CC_SHA256_DIGEST_LENGTH
import platform.Security.SecRandomCopyBytes
import platform.Security.kSecRandomDefault

@OptIn(ExperimentalForeignApi::class)
actual object Crypto {
  actual fun sha256(input: ByteArray): ByteArray {
    val digest = ByteArray(CC_SHA256_DIGEST_LENGTH)
    input.usePinned { pinnedInput ->
      digest.usePinned { pinnedDigest ->
        CC_SHA256(
          pinnedInput.addressOf(0),
          input.size.convert(),
          pinnedDigest.addressOf(0).reinterpret<UByteVar>(),
        )
      }
    }
    return digest
  }

  actual fun randomBytes(size: Int): ByteArray {
    val bytes = ByteArray(size)
    bytes.usePinned { pinned ->
      val status = SecRandomCopyBytes(kSecRandomDefault, size.convert(), pinned.addressOf(0))
      check(status == 0) { "SecRandomCopyBytes failed with status $status" }
    }
    return bytes
  }
}
