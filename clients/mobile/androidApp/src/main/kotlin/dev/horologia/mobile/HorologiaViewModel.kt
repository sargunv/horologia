package dev.horologia.mobile

import android.app.Application
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import dev.horologia.mobile.background.AndroidBackgroundScheduler
import dev.horologia.mobile.domain.MobileProfileUpdate
import dev.horologia.mobile.domain.MobileRecipeUpdate
import dev.horologia.mobile.domain.MobileTaskUpdate
import dev.horologia.mobile.navigation.HorologiaDeepLinks
import dev.horologia.mobile.navigation.SemanticDestination
import dev.horologia.mobile.runtime.AndroidAppCoreFactory
import dev.horologia.mobile.runtime.MobileAppCore
import dev.horologia.mobile.runtime.MobileSessionPhase
import dev.horologia.mobile.widgets.publishMyTasksWidgetPreview
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

internal data class PendingDeepLink(
    val deliveryId: Long,
    val destination: SemanticDestination,
    val serverId: String,
    val baseUrl: String,
    val sourceLink: String,
)

/** Owns the MobileAppCore for the activity lifecycle. */
class HorologiaViewModel(application: Application) : AndroidViewModel(application) {
    private val core: MobileAppCore = AndroidAppCoreFactory(application).create()
    val state = core.state

    private var scheduledScopeKey: String? = null
    private val queuedDeepLinks = ArrayDeque<PendingDeepLink>()
    private val _deepLinkDestination = MutableStateFlow<PendingDeepLink?>(null)
    internal val deepLinkDestination = _deepLinkDestination.asStateFlow()
    private var nextDeepLinkDeliveryId = 0L


    init {
        viewModelScope.launch {
            core.state.collect { value ->
                val accountId = value.accountId
                if (value.phase == MobileSessionPhase.SIGNED_IN && accountId != null) {
                    val key = "${value.server.serverId}/$accountId"
                    if (scheduledScopeKey != key) {
                        scheduledScopeKey = key
                        AndroidBackgroundScheduler.scheduleAfterAuthorization(
                            getApplication(),
                            value.server.serverId,
                            value.server.baseUrl,
                            accountId,
                        )
                    }
                }
                if (value.phase == MobileSessionPhase.SIGNED_IN) {
                    publishNextDeepLink()
                } else {
                    _deepLinkDestination.value = null
                }
            }
        }
        viewModelScope.launch { core.start() }
        viewModelScope.launch { publishMyTasksWidgetPreview(getApplication()) }
    }

    fun handleDeepLink(link: String) {
        val current = core.state.value
        val destination =
            HorologiaDeepLinks.parse(
                link = link,
                expectedServerId = current.server.serverId,
                expectedBaseUrl = current.server.baseUrl,
            ) ?: return
        if (
            destination is SemanticDestination.OAuthCallback ||
            queuedDeepLinks.any { it.sourceLink == link }
        ) return
        queuedDeepLinks.addLast(
            PendingDeepLink(
                deliveryId = ++nextDeepLinkDeliveryId,
                destination = destination,
                serverId = current.server.serverId,
                baseUrl = current.server.baseUrl,
                sourceLink = link,
            ),
        )
        if (current.phase == MobileSessionPhase.SIGNED_IN) publishNextDeepLink()
    }

    internal fun consumeDeepLink(pendingDeepLink: PendingDeepLink) {
        if (
            core.state.value.phase != MobileSessionPhase.SIGNED_IN ||
            queuedDeepLinks.firstOrNull() != pendingDeepLink
        ) return
        queuedDeepLinks.removeFirst()
        _deepLinkDestination.value = null
        publishNextDeepLink()
    }

    private fun publishNextDeepLink() {
        if (core.state.value.phase != MobileSessionPhase.SIGNED_IN || _deepLinkDestination.value != null) return
        val server = core.state.value.server
        while (
            queuedDeepLinks.firstOrNull()?.let {
                it.serverId != server.serverId || it.baseUrl != server.baseUrl
            } == true
        ) {
            queuedDeepLinks.removeFirst()
        }
        _deepLinkDestination.value = queuedDeepLinks.firstOrNull()
    }

    fun retryBootstrap() {
        viewModelScope.launch { core.retry() }
    }

    fun connect(baseUrl: String) {
        viewModelScope.launch {
            val trimmed = baseUrl.trim()
            if (trimmed.isEmpty()) return@launch
            val normalized =
                if (trimmed.startsWith("http://") || trimmed.startsWith("https://")) {
                    trimmed
                } else {
                    "https://$trimmed"
                }
            val current = core.state.value.server
            if (current.baseUrl != normalized) {
                core.start(current.serverId, normalized)
            }
            if (core.state.value.phase != MobileSessionPhase.SIGNED_IN) {
                core.authorize()
            }
        }
    }

    fun signOut() {
        viewModelScope.launch {
            val before = core.state.value
            val accountId = before.accountId
            core.signOut()
            scheduledScopeKey = null
            if (accountId != null) {
                AndroidBackgroundScheduler.cancel(getApplication(), before.server.serverId, accountId)
            }
        }
    }

    fun refreshMyTasks() = viewModelScope.launch { core.refreshMyTasks() }

    fun loadMoreMyTasks() = viewModelScope.launch { core.loadMoreMyTasks() }

    fun selectTask(spaceSlug: String, taskId: String) = viewModelScope.launch { core.selectTask(spaceSlug, taskId) }

    fun loadSpaces() = viewModelScope.launch { core.loadSpaces() }

    fun selectSpace(spaceSlug: String) = viewModelScope.launch { core.selectSpace(spaceSlug) }

    fun loadMoreSpaceTasks() = viewModelScope.launch { core.loadMoreSpaceTasks() }

    fun loadMoreSpaceRecipes() = viewModelScope.launch { core.loadMoreSpaceRecipes() }

    fun selectRecipe(spaceSlug: String, recipeId: String) = viewModelScope.launch { core.selectRecipe(spaceSlug, recipeId) }

    fun submitSearch(query: String) = viewModelScope.launch { core.search(query) }

    /** Returns true when the write completed without an error. */
    suspend fun updateTask(spaceSlug: String, taskId: String, update: MobileTaskUpdate): Boolean {
        core.updateTask(spaceSlug, taskId, update)
        return core.state.value.error == null
    }

    /** Returns true when the delete completed without an error. */
    suspend fun deleteTask(spaceSlug: String, taskId: String): Boolean {
        core.deleteTask(spaceSlug, taskId)
        return core.state.value.error == null
    }

    /** Returns true when the write completed without an error. */
    suspend fun updateRecipe(spaceSlug: String, recipeId: String, update: MobileRecipeUpdate): Boolean {
        core.updateRecipe(spaceSlug, recipeId, update)
        return core.state.value.error == null
    }

    /** Returns true when the write completed without an error. */
    suspend fun updateProfile(name: String, email: String): Boolean {
        core.updateProfile(MobileProfileUpdate(name, email))
        return core.state.value.error == null
    }

    override fun onCleared() {
        core.close()
    }
}
