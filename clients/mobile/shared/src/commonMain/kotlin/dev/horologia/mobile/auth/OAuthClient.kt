package dev.horologia.mobile.auth

import io.ktor.client.HttpClient
import io.ktor.client.call.body
import io.ktor.client.request.forms.submitForm
import io.ktor.client.request.bearerAuth
import io.ktor.client.request.get
import io.ktor.http.Parameters
import io.ktor.http.Url
import io.ktor.http.URLBuilder
import io.ktor.http.isSuccess
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

const val OAUTH_CLIENT_ID: String = "horologia-mobile"
const val OAUTH_REDIRECT_URI: String = "horologia://oauth/callback"
val REQUIRED_OAUTH_SCOPES: Set<String> = setOf(
    "profile:read",
    "recipes:read",
    "spaces:read",
    "tasks:read",
    "tasks:write",
)

fun authorizeEndpoint(baseUrl: String): String = endpoint(baseUrl, "oauth/authorize")
fun tokenEndpoint(baseUrl: String): String = endpoint(baseUrl, "oauth/token")
fun revokeEndpoint(baseUrl: String): String = endpoint(baseUrl, "oauth/revoke")

private fun endpoint(baseUrl: String, path: String): String =
    "${baseUrl.trimEnd('/')}/$path"

data class PendingAuthorization(
    val serverId: String,
    val state: String,
    val codeVerifier: String,
    val expectedAccountId: String?,
    val authorizationUrl: String,
)

interface AuthorizationSession {
    suspend fun authorize(authorizationUrl: String, callbackScheme: String = "horologia"): String
}

expect class PlatformAuthorizationSession() : AuthorizationSession {
    override suspend fun authorize(authorizationUrl: String, callbackScheme: String): String
}

class OAuthClient(
    private val httpClient: HttpClient,
    private val credentialStore: CredentialStore,
) {
    private val pendingMutex = Mutex()
    private val pendingByServer = mutableMapOf<String, PendingAuthorization>()

    suspend fun prepareAuthorization(
        serverId: String,
        baseUrl: String,
        expectedAccountId: String? = null,
    ): PendingAuthorization {
        require(serverId.isNotBlank()) { "serverId must not be blank" }
        val verifier = secureRandomBytes(64).base64Url()
        val state = secureRandomBytes(32).base64Url()
        val challenge = sha256(verifier.encodeToByteArray()).base64Url()
        val url = URLBuilder(authorizeEndpoint(baseUrl)).apply {
            parameters.append("response_type", "code")
            parameters.append("client_id", OAUTH_CLIENT_ID)
            parameters.append("redirect_uri", OAUTH_REDIRECT_URI)
            parameters.append("scope", REQUIRED_OAUTH_SCOPES.sorted().joinToString(" "))
            parameters.append("state", state)
            parameters.append("code_challenge", challenge)
            parameters.append("code_challenge_method", "S256")
        }.buildString()
        return PendingAuthorization(serverId, state, verifier, expectedAccountId, url).also { record ->
            pendingMutex.withLock {
                if (pendingByServer.containsKey(serverId)) {
                    throw OAuthException("An authorization is already pending for this server")
                }
                pendingByServer[serverId] = record
            }
        }
    }

    suspend fun authorize(
        serverId: String,
        baseUrl: String,
        session: AuthorizationSession,
        expectedAccountId: String? = null,
    ): CredentialBundle {
        val pending = prepareAuthorization(serverId, baseUrl, expectedAccountId)
        try {
            val callback = session.authorize(pending.authorizationUrl)
            return exchangeCallback(serverId, baseUrl, callback)
        } finally {
            cancelAuthorization(serverId, pending.state)
        }
    }

    suspend fun cancelAuthorization(serverId: String, state: String? = null) {
        pendingMutex.withLock {
            if (state == null || pendingByServer[serverId]?.state == state) pendingByServer.remove(serverId)
        }
    }

    suspend fun exchangeCallback(serverId: String, baseUrl: String, callbackUrl: String): CredentialBundle {
        val callback = try {
            Url(callbackUrl)
        } catch (error: Throwable) {
            throw OAuthException("Invalid OAuth callback URL", cause = error)
        }
        if (callback.protocol.name != "horologia" || callback.host != "oauth" || callback.encodedPath != "/callback") {
            throw OAuthException("OAuth callback does not match the registered redirect")
        }
        val state = callback.parameters["state"] ?: throw OAuthException("OAuth callback is missing state")
        val pending = pendingMutex.withLock {
            val record = pendingByServer[serverId]
                ?: throw OAuthException("No authorization is pending for this server")
            if (record.serverId != serverId || !constantTimeEquals(record.state, state)) {
                throw OAuthException("OAuth callback state or server does not match")
            }
            pendingByServer.remove(serverId)
            record
        }
        callback.parameters["error"]?.let { error ->
            val detail = callback.parameters["error_description"]
            throw OAuthException(if (detail.isNullOrBlank()) error else "$error: $detail")
        }
        val code = callback.parameters["code"] ?: throw OAuthException("OAuth callback is missing code")
        val token = requestToken(
            baseUrl,
            Parameters.build {
                append("grant_type", "authorization_code")
                append("client_id", OAUTH_CLIENT_ID)
                append("redirect_uri", OAUTH_REDIRECT_URI)
                append("code", code)
                append("code_verifier", pending.codeVerifier)
            },
            pending.expectedAccountId,
        )
        credentialStore.save(serverId, token)
        credentialStore.setActiveAccount(serverId, token.accountId)
        return token
    }

    suspend fun refresh(serverId: String, baseUrl: String, credentials: CredentialBundle): CredentialBundle {
        val refreshToken = credentials.refreshToken ?: throw OAuthException("No refresh token is available")
        val refreshed = requestToken(
            baseUrl,
            Parameters.build {
                append("grant_type", "refresh_token")
                append("client_id", OAUTH_CLIENT_ID)
                append("refresh_token", refreshToken)
            },
            credentials.accountId,
            fallbackRefreshToken = refreshToken,
            fallbackScopes = credentials.scope,
        )
        credentialStore.save(serverId, refreshed)
        return refreshed
    }

    suspend fun revokeAndDelete(serverId: String, baseUrl: String, credentials: CredentialBundle) {
        var failure: Throwable? = null
        try {
            val token = credentials.refreshToken ?: credentials.accessToken
            val response = httpClient.submitForm(
                url = revokeEndpoint(baseUrl),
                formParameters = Parameters.build {
                    append("client_id", OAUTH_CLIENT_ID)
                    append("token", token)
                    append("token_type_hint", if (credentials.refreshToken != null) "refresh_token" else "access_token")
                },
            )
            if (!response.status.isSuccess()) {
                failure = OAuthException("OAuth revocation failed: ${response.body<String>()}", response.status.value)
            }
        } catch (error: Throwable) {
            failure = error
        } finally {
            credentialStore.delete(serverId, credentials.accountId)
            if (credentialStore.getActiveAccount(serverId) == credentials.accountId) {
                credentialStore.setActiveAccount(serverId, null)
            }
        }
        failure?.let { throw it }
    }

    private suspend fun requestToken(
        baseUrl: String,
        form: Parameters,
        expectedAccountId: String?,
        fallbackRefreshToken: String? = null,
        fallbackScopes: Set<String>? = null,
    ): CredentialBundle {
        val response = httpClient.submitForm(tokenEndpoint(baseUrl), form)
        if (!response.status.isSuccess()) {
            throw OAuthException("OAuth token request failed: ${response.body<String>()}", response.status.value)
        }
        val payload = try {
            response.body<TokenResponse>()
        } catch (error: Throwable) {
            throw OAuthException("OAuth token response was invalid", response.status.value, error)
        }
        val accountId = payload.accountId ?: expectedAccountId
            ?: requestAccountId(baseUrl, payload.accessToken)
        if (expectedAccountId != null && accountId != expectedAccountId) {
            throw OAuthException("OAuth token response belongs to a different account")
        }
        val scopes = payload.scope
            ?.split(' ')
            ?.filterTo(linkedSetOf()) { it.isNotBlank() }
            ?: fallbackScopes
            ?: emptySet()
        val missingScopes = REQUIRED_OAUTH_SCOPES - scopes
        if (missingScopes.isNotEmpty()) {
            throw OAuthException("OAuth token is missing required scopes: ${missingScopes.sorted().joinToString()}")
        }
        return CredentialBundle(
            accessToken = payload.accessToken,
            refreshToken = payload.refreshToken ?: fallbackRefreshToken,
            expiresAtEpochSeconds = payload.expiresIn?.let { currentEpochSeconds() + it },
            scope = scopes,
            accountId = accountId,
        )
    }
    private suspend fun requestAccountId(baseUrl: String, accessToken: String): String {
        val response = httpClient.get(endpoint(baseUrl, "api/users/me")) {
            bearerAuth(accessToken)
        }
        if (!response.status.isSuccess()) {
            throw OAuthException("Could not load the authorized account: ${response.body<String>()}", response.status.value)
        }
        val account = try {
            response.body<AuthorizedAccount>()
        } catch (error: Throwable) {
            throw OAuthException("Authorized account response was invalid", response.status.value, error)
        }
        return account.id.takeIf(String::isNotBlank)
            ?: throw OAuthException("Authorized account response is missing id", response.status.value)
    }

}

@Serializable
private data class AuthorizedAccount(
    val id: String,
)

@Serializable
private data class TokenResponse(
    @SerialName("access_token") val accessToken: String,
    @SerialName("refresh_token") val refreshToken: String? = null,
    @SerialName("expires_in") val expiresIn: Long? = null,
    val scope: String? = null,
    @SerialName("account_id") val accountId: String? = null,
)

internal expect fun currentEpochSeconds(): Long

private fun constantTimeEquals(left: String, right: String): Boolean {
    val leftBytes = left.encodeToByteArray()
    val rightBytes = right.encodeToByteArray()
    var difference = leftBytes.size xor rightBytes.size
    val length = maxOf(leftBytes.size, rightBytes.size)
    for (index in 0 until length) {
        difference = difference or ((leftBytes.getOrElse(index) { 0 }).toInt() xor (rightBytes.getOrElse(index) { 0 }).toInt())
    }
    return difference == 0
}
