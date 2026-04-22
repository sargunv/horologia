package dev.horologia.mobile.core.platform

/**
 * Platform-provided SHA-256 + CSPRNG. Backs [dev.horologia.mobile.core.feature.login.Pkce]:
 * - `randomBytes(32)` for `code_verifier` and `state`.
 * - `sha256(verifier.encodeToByteArray())` for `code_challenge`.
 *
 * Implementations:
 * - Android / Desktop: `java.security.MessageDigest` + `java.security.SecureRandom`.
 * - iOS: `CC_SHA256` + `SecRandomCopyBytes` via `Security.framework`.
 */
expect object Crypto {
  fun sha256(input: ByteArray): ByteArray

  fun randomBytes(size: Int): ByteArray
}
