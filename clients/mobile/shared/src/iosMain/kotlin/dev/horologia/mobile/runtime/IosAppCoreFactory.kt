package dev.horologia.mobile.runtime

import dev.horologia.mobile.auth.IosCredentialStore
import dev.horologia.mobile.auth.OAuthClient
import dev.horologia.mobile.auth.PlatformAuthorizationSession
import dev.horologia.mobile.persistence.DatabaseDriverFactory
import dev.horologia.mobile.persistence.SnapshotCache
import dev.horologia.mobile.repositories.GeneratedMobileRepository
import io.ktor.client.HttpClient
import io.ktor.client.engine.darwin.Darwin
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.json.Json

class IosAppCoreFactory {
    fun create(): MobileAppCore {
        val client = HttpClient(Darwin) {
            install(ContentNegotiation) {
                json(Json { ignoreUnknownKeys = true })
            }
        }
        val store = IosCredentialStore()
        return MobileAppCore(
            initialServer = ServerProfile.Default,
            repository = GeneratedMobileRepository(),
            credentialStore = store,
            oauthClient = OAuthClient(client, store),
            authorizationSession = PlatformAuthorizationSession(),
            cache = SnapshotCache(DatabaseDriverFactory().createDriver()),
            closeResources = client::close,
        )
    }
}
