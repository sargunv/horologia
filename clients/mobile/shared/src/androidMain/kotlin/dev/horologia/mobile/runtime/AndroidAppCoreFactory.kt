package dev.horologia.mobile.runtime

import android.content.Context
import dev.horologia.mobile.auth.AndroidCredentialStore
import dev.horologia.mobile.auth.OAuthClient
import dev.horologia.mobile.auth.PlatformAuthorizationSession
import dev.horologia.mobile.persistence.DatabaseDriverFactory
import dev.horologia.mobile.persistence.SnapshotCache
import dev.horologia.mobile.repositories.GeneratedMobileRepository
import io.ktor.client.HttpClient
import io.ktor.client.engine.okhttp.OkHttp
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.json.Json

class AndroidAppCoreFactory(context: Context) {
    private val applicationContext = context.applicationContext

    fun create(): MobileAppCore {
        val client = HttpClient(OkHttp) {
            install(ContentNegotiation) {
                json(Json { ignoreUnknownKeys = true })
            }
        }
        val store = AndroidCredentialStore(applicationContext)
        return MobileAppCore(
            initialServer = ServerProfile.Default,
            repository = GeneratedMobileRepository(),
            credentialStore = store,
            oauthClient = OAuthClient(client, store),
            authorizationSession = PlatformAuthorizationSession(),
            cache = SnapshotCache(DatabaseDriverFactory(applicationContext).createDriver()),
            closeResources = client::close,
        )
    }
}
