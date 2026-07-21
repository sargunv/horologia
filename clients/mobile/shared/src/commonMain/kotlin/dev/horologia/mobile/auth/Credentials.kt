package dev.horologia.mobile.auth

import kotlinx.serialization.Serializable

@Serializable
data class CredentialBundle(
    val accessToken: String,
    val refreshToken: String?,
    val expiresAtEpochSeconds: Long?,
    val scope: Set<String>,
    val accountId: String,
)

interface CredentialStore {
    suspend fun save(serverId: String, credentials: CredentialBundle)
    suspend fun load(serverId: String, accountId: String): CredentialBundle?
    suspend fun delete(serverId: String, accountId: String)
    suspend fun setActiveAccount(serverId: String, accountId: String?)
    suspend fun getActiveAccount(serverId: String): String?

    suspend fun loadActive(serverId: String): CredentialBundle? =
        getActiveAccount(serverId)?.let { load(serverId, it) }
}

class OAuthException(
    message: String,
    val statusCode: Int? = null,
    cause: Throwable? = null,
) : Exception(message, cause)
