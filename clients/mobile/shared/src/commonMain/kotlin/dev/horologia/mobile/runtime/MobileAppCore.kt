package dev.horologia.mobile.runtime

import dev.horologia.mobile.auth.AuthorizationSession
import dev.horologia.mobile.auth.CredentialBundle
import dev.horologia.mobile.auth.CredentialStore
import dev.horologia.mobile.auth.OAuthClient
import dev.horologia.mobile.auth.OAuthException
import dev.horologia.mobile.auth.currentEpochSeconds
import dev.horologia.mobile.domain.MobileAuthConfig
import dev.horologia.mobile.domain.MobileProfileUpdate
import dev.horologia.mobile.domain.MobileRecipe
import dev.horologia.mobile.domain.MobileRecipeUpdate
import dev.horologia.mobile.domain.MobileSearchResult
import dev.horologia.mobile.domain.MobileSpace
import dev.horologia.mobile.domain.MobileTask
import dev.horologia.mobile.domain.MobileTaskVisualMetadata
import dev.horologia.mobile.domain.MobileTaskUpdate
import dev.horologia.mobile.domain.MobileUser
import dev.horologia.mobile.domain.TaskListIndicator
import dev.horologia.mobile.domain.TaskListIndicatorKind
import dev.horologia.mobile.domain.TaskListItemModel
import dev.horologia.mobile.domain.TaskStatusCategory
import dev.horologia.mobile.domain.RepositoryException
import dev.horologia.mobile.domain.ServerScope
import dev.horologia.mobile.domain.SessionScope
import dev.horologia.mobile.persistence.CachedRecipe
import dev.horologia.mobile.persistence.CachedSearchResult
import dev.horologia.mobile.persistence.CachedSpace
import dev.horologia.mobile.persistence.CachedTask
import dev.horologia.mobile.persistence.SnapshotCache
import dev.horologia.mobile.persistence.SnapshotStore
import dev.horologia.mobile.repositories.MobileRepository
import kotlin.coroutines.cancellation.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.async
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.collect
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock

data class ServerProfile(
    val serverId: String,
    val baseUrl: String,
) {
    init {
        require(serverId.isNotBlank()) { "serverId must not be blank" }
        require(baseUrl.isNotBlank()) { "baseUrl must not be blank" }
    }

    companion object {
        val Default = ServerProfile("default", "http://localhost:8080")
    }
}

enum class MobileSessionPhase {
    BOOTSTRAP,
    SIGNED_OUT,
    AUTHORIZING,
    SIGNED_IN,
}

data class MobileLoadingState(
    val bootstrap: Boolean = false,
    val myTasks: Boolean = false,
    val moreMyTasks: Boolean = false,
    val task: Boolean = false,
    val taskUpdate: Boolean = false,
    val spaces: Boolean = false,
    val space: Boolean = false,
    val moreSpaceTasks: Boolean = false,
    val moreSpaceRecipes: Boolean = false,
    val recipe: Boolean = false,
    val recipeUpdate: Boolean = false,
    val search: Boolean = false,
    val accountUpdate: Boolean = false,
)

data class MobileAppError(
    val message: String,
    val statusCode: Int? = null,
)

data class MobileAppState(
    val phase: MobileSessionPhase = MobileSessionPhase.BOOTSTRAP,
    val server: ServerProfile = ServerProfile.Default,
    val accountId: String? = null,
    val authConfig: MobileAuthConfig? = null,
    val user: MobileUser? = null,
    val myTasks: List<MobileTask> = emptyList(),
    val myTasksCursor: String? = null,
    val myTasksGeneratedAtEpochSeconds: Long? = null,
    val myTasksFromCache: Boolean = false,
    val selectedTask: MobileTask? = null,
    val spaces: List<MobileSpace> = emptyList(),
    val spacesGeneratedAtEpochSeconds: Long? = null,
    val spacesFromCache: Boolean = false,
    val selectedSpace: MobileSpace? = null,
    val spaceTasks: List<MobileTask> = emptyList(),
    val spaceTasksCursor: String? = null,
    val spaceRecipes: List<MobileRecipe> = emptyList(),
    val spaceRecipesCursor: String? = null,
    val selectedRecipe: MobileRecipe? = null,
    val searchQuery: String = "",
    val searchResults: List<MobileSearchResult> = emptyList(),
    val searchGeneratedAtEpochSeconds: Long? = null,
    val searchFromCache: Boolean = false,
    val taskVisualMetadataBySpace: Map<String, MobileTaskVisualMetadata> = emptyMap(),
    val loading: MobileLoadingState = MobileLoadingState(),
    val error: MobileAppError? = null,
) {
    fun taskListItem(task: MobileTask): TaskListItemModel {
        val metadata = taskVisualMetadataBySpace[task.spaceSlug]
        val configuredStatus = metadata?.statuses?.firstOrNull { it.label == task.status }
        val statusLabel = configuredStatus?.label ?: task.status.ifBlank { "Unknown status" }
        val priority = task.priority?.takeIf(String::isNotBlank)?.let { value ->
            val configured = metadata?.priorityLevels?.firstOrNull { it.label == value }
            TaskListIndicator(
                kind = TaskListIndicatorKind.PRIORITY,
                label = configured?.label ?: value,
                iconToken = configured?.iconToken?.takeIf(String::isNotBlank) ?: NEUTRAL_PRIORITY_ICON,
            )
        }
        val effort = task.effort?.takeIf(String::isNotBlank)?.let { value ->
            val configured = metadata?.effortLevels?.firstOrNull { it.label == value }
            TaskListIndicator(
                kind = TaskListIndicatorKind.EFFORT,
                label = configured?.label ?: value,
                iconToken = configured?.iconToken?.takeIf(String::isNotBlank) ?: NEUTRAL_EFFORT_ICON,
            )
        }
        val indicators = listOfNotNull(priority, effort)
        val accessibilityParts = buildList {
            add(task.title)
            add("Status: $statusLabel")
            task.dueText?.takeIf(String::isNotBlank)?.let { add("Due: $it") }
            priority?.let { add("Priority: ${it.label}") }
            effort?.let { add("Effort: ${it.label}") }
        }
        return TaskListItemModel(
            title = task.title,
            dueText = task.dueText?.takeIf(String::isNotBlank),
            statusLabel = statusLabel,
            statusCategory = configuredStatus?.category ?: TaskStatusCategory.NEUTRAL,
            statusIconToken = configuredStatus?.iconToken?.takeIf(String::isNotBlank) ?: NEUTRAL_STATUS_ICON,
            trailingIndicators = indicators,
            accessibilityLabel = accessibilityParts.joinToString(". "),
        )
    }
}

private const val NEUTRAL_STATUS_ICON = "circle"
private const val NEUTRAL_PRIORITY_ICON = "flag"
private const val NEUTRAL_EFFORT_ICON = "gauge"

internal interface RuntimeAuthClient {
    suspend fun authorize(
        serverId: String,
        baseUrl: String,
        session: AuthorizationSession,
    ): CredentialBundle

    suspend fun refresh(serverId: String, baseUrl: String, credentials: CredentialBundle): CredentialBundle
    suspend fun revokeAndDelete(serverId: String, baseUrl: String, credentials: CredentialBundle)
}

class MobileAppCore private constructor(
    initialServer: ServerProfile,
    private val repository: MobileRepository,
    private val credentialStore: CredentialStore,
    private val authClient: RuntimeAuthClient,
    private val authorizationSession: AuthorizationSession,
    private val cache: SnapshotStore,
    private val clock: () -> Long,
    private val closeResources: () -> Unit,
    @Suppress("UNUSED_PARAMETER") constructorMarker: Unit,
) : AutoCloseable {
    constructor(
        initialServer: ServerProfile,
        repository: MobileRepository,
        credentialStore: CredentialStore,
        oauthClient: OAuthClient,
        authorizationSession: AuthorizationSession,
        cache: SnapshotCache,
        closeResources: () -> Unit = {},
    ) : this(
        initialServer = initialServer,
        repository = repository,
        credentialStore = credentialStore,
        authClient = object : RuntimeAuthClient {
            override suspend fun authorize(
                serverId: String,
                baseUrl: String,
                session: AuthorizationSession,
            ) = oauthClient.authorize(serverId, baseUrl, session)

            override suspend fun refresh(
                serverId: String,
                baseUrl: String,
                credentials: CredentialBundle,
            ) = oauthClient.refresh(serverId, baseUrl, credentials)

            override suspend fun revokeAndDelete(
                serverId: String,
                baseUrl: String,
                credentials: CredentialBundle,
            ) = oauthClient.revokeAndDelete(serverId, baseUrl, credentials)
        },
        authorizationSession = authorizationSession,
        cache = cache,
        clock = ::currentEpochSeconds,
        closeResources = closeResources,
        constructorMarker = Unit,
    )

    internal constructor(
        initialServer: ServerProfile,
        repository: MobileRepository,
        credentialStore: CredentialStore,
        authClient: RuntimeAuthClient,
        authorizationSession: AuthorizationSession,
        cache: SnapshotStore,
        clock: () -> Long = ::currentEpochSeconds,
        closeResources: () -> Unit = {},
    ) : this(
        initialServer,
        repository,
        credentialStore,
        authClient,
        authorizationSession,
        cache,
        clock,
        closeResources,
        Unit,
    )
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    private val commandMutex = Mutex()
    private val refreshMutex = Mutex()
    private var credentials: CredentialBundle? = null

    private val mutableState = MutableStateFlow(MobileAppState(server = initialServer))
    val state: StateFlow<MobileAppState> = mutableState.asStateFlow()

    fun observe(observer: (MobileAppState) -> Unit): AutoCloseable {
        val job = scope.launch { state.collect(observer) }
        return AutoCloseable { job.cancel() }
    }

    suspend fun start() = commandMutex.withLock {
        restore(mutableState.value.server)
    }

    suspend fun start(serverId: String, baseUrl: String) = commandMutex.withLock {
        restore(ServerProfile(serverId, baseUrl))
    }

    suspend fun authorize() = commandMutex.withLock {
        val server = mutableState.value.server
        mutableState.update { it.copy(phase = MobileSessionPhase.AUTHORIZING, error = null) }
        try {
            val authorized = authClient.authorize(server.serverId, server.baseUrl, authorizationSession)
            credentials = authorized
            val user = repository.currentUser(sessionScope(authorized))
            mutableState.update {
                it.copy(
                    phase = MobileSessionPhase.SIGNED_IN,
                    accountId = authorized.accountId,
                    user = user,
                    error = null,
                )
            }
            publishCachedAccount(authorized.accountId)
            refreshMyTasksInternal(readCache = false, refreshMetadata = false)
            loadSpacesInternal(readCache = false)
        } catch (cancelled: CancellationException) {
            mutableState.update { it.copy(phase = MobileSessionPhase.SIGNED_OUT) }
            throw cancelled
        } catch (error: Throwable) {
            mutableState.update { it.copy(phase = MobileSessionPhase.SIGNED_OUT, error = error.toAppError()) }
        }
    }

    suspend fun retry() = commandMutex.withLock {
        restore(mutableState.value.server)
    }

    suspend fun signOut() = commandMutex.withLock {
        val server = mutableState.value.server
        val active = credentials ?: credentialStore.loadActive(server.serverId)
        var failure: Throwable? = null
        if (active != null) {
            try {
                authClient.revokeAndDelete(server.serverId, server.baseUrl, active)
            } catch (cancelled: CancellationException) {
                failure = cancelled
            } catch (error: Throwable) {
                failure = error
            } finally {
                try {
                    credentialStore.delete(server.serverId, active.accountId)
                    credentialStore.setActiveAccount(server.serverId, null)
                } catch (cleanupError: Throwable) {
                    if (failure == null) failure = cleanupError
                }
                try {
                    cache.clearAccount(server.serverId, active.accountId)
                } catch (cleanupError: Throwable) {
                    if (failure == null) failure = cleanupError
                }
            }
        }
        credentials = null
        mutableState.value = MobileAppState(
            phase = MobileSessionPhase.SIGNED_OUT,
            server = server,
            authConfig = mutableState.value.authConfig,
            error = failure?.takeUnless { it is CancellationException }?.toAppError(),
        )
        if (failure is CancellationException) throw failure
    }

    suspend fun refreshMyTasks() = commandMutex.withLock {
        refreshMyTasksInternal(readCache = true)
    }

    suspend fun loadMoreMyTasks() = commandMutex.withLock {
        val cursor = mutableState.value.myTasksCursor ?: return@withLock
        loading({ copy(moreMyTasks = true) }) {
            val page = repository.myTasks(authenticatedScope(), cursor)
            val items = deduplicate(mutableState.value.myTasks, page.items, MobileTask::id)
            cache.replaceTasks(
                mutableState.value.server.serverId,
                requireAccountId(),
                items.map(MobileTask::toCached),
                clock(),
                page.nextCursor,
                page.nextCursor != null,
            )
            mutableState.update {
                it.copy(
                    myTasks = items,
                    myTasksCursor = page.nextCursor,
                    myTasksGeneratedAtEpochSeconds = clock(),
                    myTasksFromCache = false,
                )
            }
        }
    }

    suspend fun selectTask(spaceSlug: String, taskId: String) = commandMutex.withLock {
        loading({ copy(task = true) }) {
            val task = repository.task(authenticatedScope(), spaceSlug, taskId)
            mutableState.update { it.copy(selectedTask = task) }
        }
    }

    suspend fun updateTask(spaceSlug: String, taskId: String, update: MobileTaskUpdate) = commandMutex.withLock {
        loading({ copy(taskUpdate = true) }) {
            val saved = repository.updateTask(authenticatedScope(), spaceSlug, taskId, update)
            mutableState.update { current ->
                current.copy(
                    selectedTask = if (current.selectedTask?.id == saved.id) saved else current.selectedTask,
                    myTasks = current.myTasks.replaceBy(saved, MobileTask::id),
                    spaceTasks = current.spaceTasks.replaceBy(saved, MobileTask::id),
                )
            }
        }
    }

    suspend fun loadSpaces() = commandMutex.withLock {
        loadSpacesInternal(readCache = true)
    }

    suspend fun selectSpace(spaceSlug: String) = commandMutex.withLock {
        val selected = mutableState.value.spaces.firstOrNull { it.slug == spaceSlug }
            ?: MobileSpace(spaceSlug, spaceSlug)
        mutableState.update {
            it.copy(
                selectedSpace = selected,
                selectedTask = null,
                selectedRecipe = null,
                spaceTasks = emptyList(),
                spaceTasksCursor = null,
                spaceRecipes = emptyList(),
                spaceRecipesCursor = null,
                error = null,
            )
        }
        loading({ copy(space = true) }) {
            coroutineScope {
                val session = authenticatedScope()
                val tasks = async { repository.spaceTasks(session, spaceSlug) }
                val recipes = async { repository.spaceRecipes(session, spaceSlug) }
                val metadata = async { loadTaskVisualMetadata(session, listOf(spaceSlug)) }
                val taskPage = tasks.await()
                val recipePage = recipes.await()
                metadata.await()
                mutableState.update {
                    it.copy(
                        spaceTasks = taskPage.items.distinctBy(MobileTask::id),
                        spaceTasksCursor = taskPage.nextCursor,
                        spaceRecipes = recipePage.items.distinctBy(MobileRecipe::id),
                        spaceRecipesCursor = recipePage.nextCursor,
                    )
                }
            }
        }
    }

    suspend fun loadMoreSpaceTasks() = commandMutex.withLock {
        val selected = mutableState.value.selectedSpace ?: return@withLock
        val cursor = mutableState.value.spaceTasksCursor ?: return@withLock
        loading({ copy(moreSpaceTasks = true) }) {
            val page = repository.spaceTasks(authenticatedScope(), selected.slug, cursor)
            mutableState.update {
                it.copy(
                    spaceTasks = deduplicate(it.spaceTasks, page.items, MobileTask::id),
                    spaceTasksCursor = page.nextCursor,
                )
            }
        }
    }

    suspend fun loadMoreSpaceRecipes() = commandMutex.withLock {
        val selected = mutableState.value.selectedSpace ?: return@withLock
        val cursor = mutableState.value.spaceRecipesCursor ?: return@withLock
        loading({ copy(moreSpaceRecipes = true) }) {
            val page = repository.spaceRecipes(authenticatedScope(), selected.slug, cursor)
            mutableState.update {
                it.copy(
                    spaceRecipes = deduplicate(it.spaceRecipes, page.items, MobileRecipe::id),
                    spaceRecipesCursor = page.nextCursor,
                )
            }
        }
    }

    suspend fun selectRecipe(spaceSlug: String, recipeId: String) = commandMutex.withLock {
        loading({ copy(recipe = true) }) {
            val recipe = repository.recipe(authenticatedScope(), spaceSlug, recipeId)
            mutableState.update { it.copy(selectedRecipe = recipe) }
        }
    }

    suspend fun updateRecipe(spaceSlug: String, recipeId: String, update: MobileRecipeUpdate) = commandMutex.withLock {
        loading({ copy(recipeUpdate = true) }) {
            val saved = repository.updateRecipe(authenticatedScope(), spaceSlug, recipeId, update)
            mutableState.update { current ->
                current.copy(
                    selectedRecipe = if (current.selectedRecipe?.id == saved.id) saved else current.selectedRecipe,
                    spaceRecipes = current.spaceRecipes.replaceBy(saved, MobileRecipe::id),
                )
            }
        }
    }

    suspend fun search(query: String) = commandMutex.withLock {
        val normalized = query.trim()
        mutableState.update { it.copy(searchQuery = normalized, error = null) }
        if (normalized.isEmpty()) {
            mutableState.update {
                it.copy(searchResults = emptyList(), searchGeneratedAtEpochSeconds = null, searchFromCache = false)
            }
            return@withLock
        }
        val accountId = requireAccountId()
        val serverId = mutableState.value.server.serverId
        cache.readSearch(serverId, accountId, normalized)?.let { snapshot ->
            mutableState.update {
                it.copy(
                    searchResults = snapshot.items.map(CachedSearchResult::toDomain),
                    searchGeneratedAtEpochSeconds = snapshot.generatedAt,
                    searchFromCache = true,
                )
            }
        }
        loading({ copy(search = true) }) {
            val results = repository.search(
                authenticatedScope(),
                normalized,
                mutableState.value.selectedSpace?.slug,
            ).distinctBy(MobileSearchResult::id)
            val generatedAt = clock()
            cache.replaceSearch(serverId, accountId, normalized, results.map(MobileSearchResult::toCached), generatedAt, null, false)
            mutableState.update {
                it.copy(searchResults = results, searchGeneratedAtEpochSeconds = generatedAt, searchFromCache = false)
            }
        }
    }

    suspend fun updateProfile(update: MobileProfileUpdate) = commandMutex.withLock {
        loading({ copy(accountUpdate = true) }) {
            val saved = repository.updateProfile(authenticatedScope(), update)
            mutableState.update { it.copy(user = saved) }
        }
    }

    fun clearSelection() {
        mutableState.update {
            it.copy(
                selectedTask = null,
                selectedSpace = null,
                spaceTasks = emptyList(),
                spaceTasksCursor = null,
                spaceRecipes = emptyList(),
                spaceRecipesCursor = null,
                selectedRecipe = null,
                searchQuery = "",
                searchResults = emptyList(),
                searchGeneratedAtEpochSeconds = null,
                searchFromCache = false,
                error = null,
            )
        }
    }

    override fun close() {
        scope.cancel()
        closeResources()
    }

    private suspend fun restore(server: ServerProfile) {
        mutableState.value = MobileAppState(
            phase = MobileSessionPhase.BOOTSTRAP,
            server = server,
            loading = MobileLoadingState(bootstrap = true),
        )
        var storedAccountId: String? = null
        try {
            val stored = credentialStore.loadActive(server.serverId)
            storedAccountId = stored?.accountId
            credentials = stored
            if (stored != null) publishCachedAccount(stored.accountId)

            val config = repository.authConfig(ServerScope(server.serverId, server.baseUrl))
            mutableState.update { it.copy(authConfig = config) }
            if (stored == null) {
                mutableState.update {
                    it.copy(phase = MobileSessionPhase.SIGNED_OUT, loading = it.loading.copy(bootstrap = false))
                }
                return
            }
            val user = repository.currentUser(authenticatedScope())
            mutableState.update {
                it.copy(phase = MobileSessionPhase.SIGNED_IN, accountId = stored.accountId, user = user)
            }
            refreshMyTasksInternal(readCache = false, refreshMetadata = false)
            loadSpacesInternal(readCache = false)
            mutableState.update { it.copy(loading = it.loading.copy(bootstrap = false)) }
        } catch (cancelled: CancellationException) {
            mutableState.update { it.copy(loading = it.loading.copy(bootstrap = false)) }
            throw cancelled
        } catch (error: Throwable) {
            mutableState.update {
                it.copy(
                    phase = if (storedAccountId == null) MobileSessionPhase.SIGNED_OUT else MobileSessionPhase.SIGNED_IN,
                    loading = it.loading.copy(bootstrap = false),
                    error = error.toAppError(),
                )
            }
        }
    }

    private suspend fun refreshMyTasksInternal(readCache: Boolean, refreshMetadata: Boolean = true) {
        val accountId = requireAccountId()
        val serverId = mutableState.value.server.serverId
        if (readCache) publishCachedTasks(serverId, accountId)
        loading({ copy(myTasks = true) }) {
            val session = authenticatedScope()
            val page = repository.myTasks(session)
            val items = page.items.distinctBy(MobileTask::id)
            val generatedAt = clock()
            cache.replaceTasks(serverId, accountId, items.map(MobileTask::toCached), generatedAt, page.nextCursor, page.nextCursor != null)
            mutableState.update {
                it.copy(
                    myTasks = items,
                    myTasksCursor = page.nextCursor,
                    myTasksGeneratedAtEpochSeconds = generatedAt,
                    myTasksFromCache = false,
                )
            }
            if (refreshMetadata) {
                loadTaskVisualMetadata(session, items.map(MobileTask::spaceSlug))
            }
        }
    }

    private suspend fun loadSpacesInternal(readCache: Boolean) {
        val accountId = requireAccountId()
        val serverId = mutableState.value.server.serverId
        if (readCache) publishCachedSpaces(serverId, accountId)
        loading({ copy(spaces = true) }) {
            val session = authenticatedScope()
            val spaces = repository.spaces(session).distinctBy(MobileSpace::slug)
            val generatedAt = clock()
            cache.replaceSpaces(serverId, accountId, spaces.map(MobileSpace::toCached), generatedAt, null, false)
            mutableState.update {
                it.copy(spaces = spaces, spacesGeneratedAtEpochSeconds = generatedAt, spacesFromCache = false)
            }
            loadTaskVisualMetadata(session, spaces.map(MobileSpace::slug))
        }
    }

    private suspend fun loadTaskVisualMetadata(session: SessionScope, spaceSlugs: List<String>) = coroutineScope {
        val previous = mutableState.value.taskVisualMetadataBySpace
        val metadataJobs = spaceSlugs.distinct().associateWith { spaceSlug ->
            async {
                val current = previous[spaceSlug] ?: MobileTaskVisualMetadata()
                val statuses = async {
                    metadataValue(current.statuses) { repository.taskStatuses(session, spaceSlug) }
                }
                val effortLevels = async {
                    metadataValue(current.effortLevels) { repository.taskEffortLevels(session, spaceSlug) }
                }
                val priorityLevels = async {
                    metadataValue(current.priorityLevels) { repository.taskPriorityLevels(session, spaceSlug) }
                }
                MobileTaskVisualMetadata(
                    statuses = statuses.await(),
                    effortLevels = effortLevels.await(),
                    priorityLevels = priorityLevels.await(),
                )
            }
        }
        val refreshed = metadataJobs.mapValues { it.value.await() }
        mutableState.update {
            it.copy(taskVisualMetadataBySpace = previous + refreshed)
        }
    }

    private suspend fun <T> metadataValue(fallback: T, request: suspend () -> T): T =
        try {
            request()
        } catch (cancelled: CancellationException) {
            throw cancelled
        } catch (_: Throwable) {
            fallback
        }

    private fun publishCachedAccount(accountId: String) {
        val serverId = mutableState.value.server.serverId
        mutableState.update { it.copy(accountId = accountId) }
        publishCachedTasks(serverId, accountId)
        publishCachedSpaces(serverId, accountId)
    }

    private fun publishCachedTasks(serverId: String, accountId: String) {
        cache.readTasks(serverId, accountId)?.let { snapshot ->
            mutableState.update {
                it.copy(
                    myTasks = snapshot.items.map(CachedTask::toDomain),
                    myTasksCursor = snapshot.cursor,
                    myTasksGeneratedAtEpochSeconds = snapshot.generatedAt,
                    myTasksFromCache = true,
                )
            }
        }
    }

    private fun publishCachedSpaces(serverId: String, accountId: String) {
        cache.readSpaces(serverId, accountId)?.let { snapshot ->
            mutableState.update {
                it.copy(
                    spaces = snapshot.items.map(CachedSpace::toDomain),
                    spacesGeneratedAtEpochSeconds = snapshot.generatedAt,
                    spacesFromCache = true,
                )
            }
        }
    }

    private suspend fun authenticatedScope(): SessionScope {
        val server = mutableState.value.server
        val active = refreshMutex.withLock {
            val current = credentials ?: credentialStore.loadActive(server.serverId)
                ?: throw IllegalStateException("No signed-in account")
            val expiresAt = current.expiresAtEpochSeconds
            if (expiresAt != null && expiresAt <= clock()) {
                authClient.refresh(server.serverId, server.baseUrl, current).also { credentials = it }
            } else {
                credentials = current
                current
            }
        }
        return sessionScope(active)
    }

    private fun sessionScope(bundle: CredentialBundle): SessionScope {
        val server = mutableState.value.server
        return SessionScope(server.serverId, server.baseUrl, bundle.accountId, bundle.accessToken)
    }

    private fun requireAccountId(): String = credentials?.accountId
        ?: mutableState.value.accountId
        ?: throw IllegalStateException("No signed-in account")

    private suspend fun loading(
        setLoading: MobileLoadingState.() -> MobileLoadingState,
        block: suspend () -> Unit,
    ) {
        mutableState.update { it.copy(loading = it.loading.setLoading(), error = null) }
        try {
            block()
        } catch (cancelled: CancellationException) {
            throw cancelled
        } catch (error: Throwable) {
            mutableState.update { it.copy(error = error.toAppError()) }
        } finally {
            val before = mutableState.value.loading
            val cleared = when {
                before.myTasks -> before.copy(myTasks = false)
                before.moreMyTasks -> before.copy(moreMyTasks = false)
                before.task -> before.copy(task = false)
                before.taskUpdate -> before.copy(taskUpdate = false)
                before.spaces -> before.copy(spaces = false)
                before.space -> before.copy(space = false)
                before.moreSpaceTasks -> before.copy(moreSpaceTasks = false)
                before.moreSpaceRecipes -> before.copy(moreSpaceRecipes = false)
                before.recipe -> before.copy(recipe = false)
                before.recipeUpdate -> before.copy(recipeUpdate = false)
                before.search -> before.copy(search = false)
                before.accountUpdate -> before.copy(accountUpdate = false)
                else -> before
            }
            mutableState.update { it.copy(loading = cleared) }
        }
    }
}

private fun Throwable.toAppError(): MobileAppError = MobileAppError(
    message = message ?: "Unexpected error",
    statusCode = when (this) {
        is RepositoryException -> statusCode
        is OAuthException -> statusCode
        else -> null
    },
)

private fun MobileTask.toCached() = CachedTask(id, spaceSlug, title, description, status, effort, priority, dueText, tags)
private fun CachedTask.toDomain() = MobileTask(id, spaceSlug, title, description.orEmpty(), status, effort, priority, dueText, tags)
private fun MobileSpace.toCached() = CachedSpace(slug, name)
private fun CachedSpace.toDomain() = MobileSpace(slug, name)
private fun MobileRecipe.toCached() = CachedRecipe(id, spaceSlug, title, description, tags)
private fun CachedRecipe.toDomain() = MobileRecipe(id, spaceSlug, title, description.orEmpty(), tags)
private fun MobileSearchResult.toCached() = CachedSearchResult(id, spaceSlug, title, kind, detail)
private fun CachedSearchResult.toDomain() = MobileSearchResult(id, spaceSlug, title, kind, detail.orEmpty())

private inline fun <T, K> deduplicate(existing: List<T>, incoming: List<T>, key: (T) -> K): List<T> {
    val values = LinkedHashMap<K, T>(existing.size + incoming.size)
    existing.forEach { values[key(it)] = it }
    incoming.forEach { values[key(it)] = it }
    return values.values.toList()
}

private inline fun <T, K> List<T>.replaceBy(value: T, key: (T) -> K): List<T> {
    val expected = key(value)
    var replaced = false
    val result = map {
        if (key(it) == expected) {
            replaced = true
            value
        } else it
    }
    return if (replaced) result else this
}
