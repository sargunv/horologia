package dev.horologia.mobile.core.feature.login

import dev.horologia.mobile.core.platform.Crypto
import kotlin.io.encoding.Base64
import kotlin.io.encoding.ExperimentalEncodingApi

/**
 * RFC 7636 PKCE helpers. `code_verifier` is 32 random bytes base64url-no-padded (yielding a
 * 43-character verifier, in the valid 43..128 range). `code_challenge` is `SHA-256(verifier)`
 * base64url-no-padded (43 characters, method `S256`).
 *
 * Kept as a tiny stateless `object` so the VM can call it without DI wiring and tests can feed in a
 * fake [Crypto] via `CryptoInjector` (the expect object isn't directly fake-able).
 */
@OptIn(ExperimentalEncodingApi::class)
object Pkce {
  private val base64Url = Base64.UrlSafe.withPadding(Base64.PaddingOption.ABSENT)

  fun generateCodeVerifier(): String = base64Url.encode(Crypto.randomBytes(size = 32))

  fun generateState(): String = base64Url.encode(Crypto.randomBytes(size = 16))

  fun codeChallengeS256(verifier: String): String =
    base64Url.encode(Crypto.sha256(verifier.encodeToByteArray()))
}
