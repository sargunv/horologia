package dev.horologia.mobile.core.feature.login

import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue

class PkceTest {
  @Test
  fun codeVerifierLengthIs43() {
    // 32 bytes -> base64url-no-pad -> 43 chars (ceil(32*4/3) = 43).
    val verifier = Pkce.generateCodeVerifier()
    assertEquals(43, verifier.length, "Expected 43-char verifier, got \"$verifier\"")
    val allowed = (('A'..'Z') + ('a'..'z') + ('0'..'9') + listOf('-', '_')).toSet()
    assertTrue(verifier.all { it in allowed }, "Verifier contains non-base64url chars: $verifier")
  }

  @Test
  fun stateLengthIs22() {
    // 16 bytes -> base64url-no-pad -> 22 chars.
    val state = Pkce.generateState()
    assertEquals(22, state.length, "Expected 22-char state, got \"$state\"")
  }

  @Test
  fun codeChallengeMatchesRfc7636Vector() {
    // RFC 7636 § A.3 reference:
    //   verifier = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
    //   challenge = "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
    val verifier = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
    val expected = "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
    assertEquals(expected, Pkce.codeChallengeS256(verifier = verifier))
  }

  @Test
  fun verifiersAreUnique() {
    val sampled = buildSet { repeat(32) { add(Pkce.generateCodeVerifier()) } }
    assertTrue(sampled.size > 1, "Verifier is not random across invocations")
  }

  @Test
  fun codeChallengeIsDeterministic() {
    val verifier = Pkce.generateCodeVerifier()
    assertEquals(Pkce.codeChallengeS256(verifier), Pkce.codeChallengeS256(verifier))
  }

  @Test
  fun codeChallengePropertyCheck() {
    val verifiers = buildSet { repeat(256) { add(Pkce.generateCodeVerifier()) } }
    assertEquals(256, verifiers.size, "verifier RNG produced duplicates")
    val challenges = verifiers.map { Pkce.codeChallengeS256(verifier = it) }
    assertTrue(challenges.all { it.length == 43 }, "all challenges must be 43 chars")
    assertEquals(
      challenges.size,
      challenges.toSet().size,
      "unique verifiers must produce unique challenges",
    )
  }
}
