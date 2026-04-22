package dev.horologia.mobile.core.session

/**
 * Per-host secure storage for OAuth tokens.
 *
 * Keyed on the server's host (scheme/port stripped) so that switching between two Horologia servers
 * without signing out of the first stays legal. Backed by the OS secure store on each platform:
 * - Android: `EncryptedSharedPreferences`
 * - iOS: Keychain (`kSecAttrAccessibleAfterFirstUnlock`)
 * - Desktop: AES-GCM-encrypted file at the platform user-data dir with a sibling random-key file
 *   (mode 0600).
 */
expect class SessionStore {
  suspend fun read(host: String): StoredSession?

  suspend fun write(host: String, session: StoredSession)

  suspend fun clear(host: String)
}
