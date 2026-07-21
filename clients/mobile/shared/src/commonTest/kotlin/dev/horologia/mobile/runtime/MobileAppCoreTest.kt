package dev.horologia.mobile.runtime

import dev.horologia.mobile.auth.AuthorizationSession
import dev.horologia.mobile.auth.CredentialBundle
import dev.horologia.mobile.auth.CredentialStore
import dev.horologia.mobile.domain.MobileAuthConfig
import dev.horologia.mobile.domain.MobileProfileUpdate
import dev.horologia.mobile.domain.MobileRecipe
import dev.horologia.mobile.domain.MobileRecipeUpdate
import dev.horologia.mobile.domain.MobileSearchResult
import dev.horologia.mobile.domain.MobileSpace
import dev.horologia.mobile.domain.MobileTask
import dev.horologia.mobile.domain.MobileTaskEffortDefinition
import dev.horologia.mobile.domain.MobileTaskPriorityDefinition
import dev.horologia.mobile.domain.MobileTaskStatusDefinition
import dev.horologia.mobile.domain.MobileTaskVisualMetadata
import dev.horologia.mobile.domain.MobileTaskUpdate
import dev.horologia.mobile.domain.MobileUser
import dev.horologia.mobile.domain.Page
import dev.horologia.mobile.domain.ServerScope
import dev.horologia.mobile.domain.SessionScope
import dev.horologia.mobile.domain.TaskListIndicatorKind
import dev.horologia.mobile.domain.TaskStatusCategory
import dev.horologia.mobile.persistence.CachedRecipe
import dev.horologia.mobile.persistence.CachedSearchResult
import dev.horologia.mobile.persistence.CachedSnapshot
import dev.horologia.mobile.persistence.CachedSpace
import dev.horologia.mobile.persistence.CachedTask
import dev.horologia.mobile.persistence.SnapshotStore
import dev.horologia.mobile.repositories.MobileRepository
import kotlin.coroutines.cancellation.CancellationException
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertNull
import kotlin.test.assertTrue
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.delay
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.launch
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.withTimeout

class MobileAppCoreTest {
    @Test
    fun signedOutBootstrapWithoutCredentials() = runBlocking {
        val fixture = Fixture(credentials = null)

        fixture.core.start()

        assertEquals(MobileSessionPhase.SIGNED_OUT, fixture.core.state.value.phase)
        assertNull(fixture.core.state.value.accountId)
        assertFalse(fixture.core.state.value.loading.bootstrap)
        assertEquals(listOf(ServerScope("server-a", "https://a.example")), fixture.repository.authConfigScopes)
        assertTrue(fixture.repository.sessionScopes.isEmpty())
        fixture.close()
    }

    @Test
    fun unreachableFirstLaunchServerPublishesEditableSignedOutRecovery() = runBlocking {
        val fixture = Fixture(credentials = null).apply {
            repository.authConfigFailure = IllegalStateException("server unreachable")
        }
        val unreachableServer = ServerProfile("local-development", "http://localhost:8080")

        fixture.core.start(unreachableServer.serverId, unreachableServer.baseUrl)

        val state = fixture.core.state.value
        assertEquals(MobileSessionPhase.SIGNED_OUT, state.phase)
        assertFalse(state.loading.bootstrap)
        assertEquals("server unreachable", state.error?.message)
        assertEquals(unreachableServer, state.server)
        assertNull(state.authConfig)
        assertNull(state.accountId)
        assertEquals(listOf(ServerScope(unreachableServer.serverId, unreachableServer.baseUrl)), fixture.repository.authConfigScopes)
        fixture.close()
    }

    @Test
    fun failedRestoreWithStoredCredentialsPreservesCachedSignedInState() = runBlocking {
        val cached = task("cached", "Cached")
        val fixture = Fixture(credentials = credentials("account-a")).apply {
            cache.taskSnapshots["server-a" to "account-a"] = snapshot(cached, generatedAt = 41)
            repository.authConfigFailure = IllegalStateException("server unreachable")
        }

        fixture.core.start()

        val state = fixture.core.state.value
        assertEquals(MobileSessionPhase.SIGNED_IN, state.phase)
        assertFalse(state.loading.bootstrap)
        assertEquals("server unreachable", state.error?.message)
        assertEquals("account-a", state.accountId)
        assertEquals(listOf(cached), state.myTasks)
        assertTrue(state.myTasksFromCache)
        fixture.close()
    }

    @Test
    fun restoredCredentialsPublishScopedCachedTasksBeforeNetworkCompletes() = runBlocking {
        val gate = CompletableDeferred<Unit>()
        val enteredNetwork = CompletableDeferred<Unit>()
        val cached = task("cached", "Cached")
        val fixture = Fixture(credentials = credentials("account-a")).apply {
            cache.taskSnapshots["server-a" to "account-a"] = snapshot(cached, generatedAt = 41, cursor = "cached-next")
            repository.myTasksHandler = { scope, _ ->
                enteredNetwork.complete(Unit)
                gate.await()
                repository.sessionScopes += scope
                Page(listOf(task("network", "Network")), null)
            }
        }

        val start = launch { fixture.core.start() }
        enteredNetwork.await()

        assertEquals(listOf(cached), fixture.core.state.value.myTasks)
        assertTrue(fixture.core.state.value.myTasksFromCache)
        assertEquals(41, fixture.core.state.value.myTasksGeneratedAtEpochSeconds)
        assertEquals(listOf("server-a" to "account-a"), fixture.cache.taskReads)
        assertFalse(fixture.cache.taskReads.contains("server-a" to "other-account"))

        gate.complete(Unit)
        start.join()
        fixture.close()
    }

    @Test
    fun networkRefreshReplacesCacheUsingActiveServerAndAccount() = runBlocking {
        val network = task("network", "Fresh")
        val fixture = Fixture(credentials = credentials("account-a"), clock = 900).apply {
            cache.taskSnapshots["server-a" to "account-a"] = snapshot(task("cached", "Stale"), generatedAt = 20)
            cache.taskSnapshots["other-server" to "account-a"] = snapshot(task("wrong-server", "Wrong"), generatedAt = 30)
            cache.taskSnapshots["server-a" to "other-account"] = snapshot(task("wrong-account", "Wrong"), generatedAt = 40)
            repository.firstTasks = Page(listOf(network), "page-2")
        }

        fixture.core.start()

        val state = fixture.core.state.value
        assertEquals(listOf(network), state.myTasks)
        assertEquals("page-2", state.myTasksCursor)
        assertEquals(900, state.myTasksGeneratedAtEpochSeconds)
        assertFalse(state.myTasksFromCache)
        assertEquals("server-a", fixture.repository.sessionScopes.first().serverId)
        assertEquals("account-a", fixture.repository.sessionScopes.first().accountId)
        assertEquals(listOf("server-a" to "account-a"), fixture.cache.taskWrites.map { it.first })
        assertEquals(listOf(network.id), fixture.cache.taskWrites.single().second.items.map(CachedTask::id))
        fixture.close()
    }

    @Test
    fun paginationDeduplicatesByIdAndTransitionsCursor() = runBlocking {
        val original = task("one", "Original")
        val duplicateUpdated = task("one", "Updated")
        val second = task("two", "Second")
        val fixture = Fixture(credentials = credentials("account-a"), clock = 123).apply {
            repository.firstTasks = Page(listOf(original), "next")
            repository.moreTasks["next"] = Page(listOf(duplicateUpdated, second, second), null)
        }
        fixture.core.start()

        fixture.core.loadMoreMyTasks()

        assertEquals(listOf(duplicateUpdated, second), fixture.core.state.value.myTasks)
        assertNull(fixture.core.state.value.myTasksCursor)
        assertEquals(listOf(null, "next"), fixture.repository.taskCursors)
        assertEquals(listOf("one", "two"), fixture.cache.taskWrites.last().second.items.map(CachedTask::id))
        fixture.close()
    }

    @Test
    fun failedMutationDoesNotPublishLocalChanges() = runBlocking {
        val original = task("one", "Original")
        val failure = IllegalStateException("save failed")
        val fixture = signedInFixture(original).apply { repository.updateTaskResult = Result.failure(failure) }
        fixture.core.selectTask(original.spaceSlug, original.id)
        val beforeList = fixture.core.state.value.myTasks
        val beforeDetail = fixture.core.state.value.selectedTask

        fixture.core.updateTask(original.spaceSlug, original.id, MobileTaskUpdate(title = "Optimistic"))

        assertEquals(beforeList, fixture.core.state.value.myTasks)
        assertEquals(beforeDetail, fixture.core.state.value.selectedTask)
        assertEquals("save failed", fixture.core.state.value.error?.message)
        fixture.close()
    }

    @Test
    fun successfulMutationReplacesListAndSelectedDetail() = runBlocking {
        val original = task("one", "Original")
        val saved = task("one", "Saved by server")
        val fixture = signedInFixture(original).apply { repository.updateTaskResult = Result.success(saved) }
        fixture.core.selectTask(original.spaceSlug, original.id)

        fixture.core.updateTask(original.spaceSlug, original.id, MobileTaskUpdate(title = "Requested"))

        assertEquals(listOf(saved), fixture.core.state.value.myTasks)
        assertEquals(saved, fixture.core.state.value.selectedTask)
        assertNull(fixture.core.state.value.error)
        fixture.close()
    }

    @Test
    fun cancellationIsRethrownWithoutBecomingAppError() = runBlocking {
        val original = task("one", "Original")
        val fixture = signedInFixture(original).apply {
            repository.updateTaskResult = Result.failure(CancellationException("screen closed"))
        }

        var thrown: CancellationException? = null
        try {
            fixture.core.updateTask(original.spaceSlug, original.id, MobileTaskUpdate(title = "Ignored"))
        } catch (cancelled: CancellationException) {
            thrown = cancelled
        }

        assertEquals("screen closed", thrown?.message)
        assertNull(fixture.core.state.value.error)
        assertEquals(listOf(original), fixture.core.state.value.myTasks)
        assertFalse(fixture.core.state.value.loading.taskUpdate)
        fixture.close()
    }

    @Test
    fun signOutDeletesLocalCredentialsAndCacheWhenRevokeFails() = runBlocking {
        val active = credentials("account-a")
        val fixture = Fixture(credentials = active).apply {
            auth.revokeFailure = IllegalStateException("server unavailable")
        }
        fixture.core.start()

        fixture.core.signOut()

        assertEquals(listOf("server-a" to "account-a"), fixture.credentials.deleted)
        assertEquals(listOf<Pair<String, String?>>("server-a" to null), fixture.credentials.activeChanges)
        assertEquals(listOf("server-a" to "account-a"), fixture.cache.clearedAccounts)
        assertEquals(MobileSessionPhase.SIGNED_OUT, fixture.core.state.value.phase)
        assertNull(fixture.core.state.value.accountId)
        assertEquals("server unavailable", fixture.core.state.value.error?.message)
        fixture.close()
    }

    @Test
    fun closingObserverStopsLaterStateEmissions() = runBlocking {
        val fixture = signedInFixture(task("one", "Original"))
        val observed = Channel<MobileAppState>(Channel.UNLIMITED)
        val observer = fixture.core.observe { observed.trySend(it) }
        withTimeout(2_000) { observed.receive() }
        observer.close()
        delay(25)
        while (observed.tryReceive().isSuccess) {
            // Discard emissions that were already queued before cancellation completed.
        }

        fixture.core.clearSelection()
        fixture.core.refreshMyTasks()
        delay(50)

        assertTrue(observed.tryReceive().isFailure)
        fixture.close()
    }

    @Test
    fun taskListItemMapsConfiguredVisualsInPriorityThenEffortOrder() {
        val state = MobileAppState(
            taskVisualMetadataBySpace = mapOf(
                "home" to MobileTaskVisualMetadata(
                    statuses = listOf(
                        MobileTaskStatusDefinition("Doing", TaskStatusCategory.INTERMEDIATE, "loader"),
                    ),
                    effortLevels = listOf(MobileTaskEffortDefinition("Large", "flame")),
                    priorityLevels = listOf(MobileTaskPriorityDefinition("High", "signal-high")),
                ),
            ),
        )
        val item = state.taskListItem(
            task("configured", "Write tests").copy(
                status = "Doing",
                priority = "High",
                effort = "Large",
                dueText = "2026-07-21",
            ),
        )

        assertEquals("Write tests", item.title)
        assertEquals("2026-07-21", item.dueText)
        assertEquals("Doing", item.statusLabel)
        assertEquals(TaskStatusCategory.INTERMEDIATE, item.statusCategory)
        assertEquals("loader", item.statusIconToken)
        assertEquals(listOf(TaskListIndicatorKind.PRIORITY, TaskListIndicatorKind.EFFORT), item.trailingIndicators.map { it.kind })
        assertEquals(listOf("signal-high", "flame"), item.trailingIndicators.map { it.iconToken })
        assertEquals(
            "Write tests. Status: Doing. Due: 2026-07-21. Priority: High. Effort: Large",
            item.accessibilityLabel,
        )
    }

    @Test
    fun taskListItemUsesNeutralFallbacksForMissingConfiguration() {
        val item = MobileAppState().taskListItem(
            task("fallback", "Unconfigured").copy(
                status = "",
                priority = "Urgent",
                effort = "Unknown",
                dueText = "",
            ),
        )

        assertEquals("Unknown status", item.statusLabel)
        assertEquals(TaskStatusCategory.NEUTRAL, item.statusCategory)
        assertEquals("circle", item.statusIconToken)
        assertNull(item.dueText)
        assertEquals(listOf("flag", "gauge"), item.trailingIndicators.map { it.iconToken })
        assertEquals("Unconfigured. Status: Unknown status. Priority: Urgent. Effort: Unknown", item.accessibilityLabel)
    }

    @Test
    fun signedInBootstrapLoadsEverySpacesMetadataAndIsolatesFailures() = runBlocking {
        val fixture = Fixture(credentials = credentials("account-a")).apply {
            repository.spacesResult = listOf(MobileSpace("home", "Home"), MobileSpace("work", "Work"))
            repository.statuses["home"] = Result.success(
                listOf(MobileTaskStatusDefinition("Open", TaskStatusCategory.INITIAL, "circle-dot")),
            )
            repository.efforts["work"] = Result.success(listOf(MobileTaskEffortDefinition("Small", "gauge")))
            repository.priorities["home"] = Result.failure(IllegalStateException("metadata unavailable"))
        }

        fixture.core.start()

        val state = fixture.core.state.value
        assertEquals(listOf("home", "work"), state.spaces.map { it.slug })
        assertEquals(setOf("home", "work"), state.taskVisualMetadataBySpace.keys)
        assertEquals("circle-dot", state.taskVisualMetadataBySpace.getValue("home").statuses.single().iconToken)
        assertTrue(state.taskVisualMetadataBySpace.getValue("home").priorityLevels.isEmpty())
        assertEquals("gauge", state.taskVisualMetadataBySpace.getValue("work").effortLevels.single().iconToken)
        assertEquals(listOf("home", "work"), fixture.repository.statusRequests)
        assertEquals(listOf("home", "work"), fixture.repository.effortRequests)
        assertEquals(listOf("home", "work"), fixture.repository.priorityRequests)
        assertNull(state.error)
        assertEquals(MobileSessionPhase.SIGNED_IN, state.phase)
        fixture.close()
    }

    @Test
    fun metadataEndpointsForAllSpacesStartConcurrently() = runBlocking {
        val gate = CompletableDeferred<Unit>()
        val entered = Channel<String>(Channel.UNLIMITED)
        val fixture = Fixture(credentials = credentials("account-a")).apply {
            repository.spacesResult = listOf(MobileSpace("home", "Home"), MobileSpace("work", "Work"))
            repository.metadataHandler = { kind, slug ->
                entered.send("$slug:$kind")
                gate.await()
            }
        }

        val start = launch { fixture.core.start() }
        val requests = withTimeout(2_000) { List(6) { entered.receive() }.toSet() }

        assertEquals(
            setOf(
                "home:status", "home:effort", "home:priority",
                "work:status", "work:effort", "work:priority",
            ),
            requests,
        )
        gate.complete(Unit)
        start.join()
        fixture.close()
    }

    @Test
    fun taskAndSelectedSpaceRefreshesReloadVisualMetadata() = runBlocking {
        val fixture = Fixture(credentials = credentials("account-a")).apply {
            repository.firstTasks = Page(listOf(task("one", "Task")), null)
            repository.spacesResult = listOf(MobileSpace("home", "Home"))
            repository.statuses["home"] = Result.success(
                listOf(MobileTaskStatusDefinition("Open", TaskStatusCategory.INITIAL, "circle")),
            )
        }
        fixture.core.start()
        assertEquals("circle", fixture.core.state.value.taskVisualMetadataBySpace.getValue("home").statuses.single().iconToken)

        fixture.repository.statuses["home"] = Result.success(
            listOf(MobileTaskStatusDefinition("Open", TaskStatusCategory.INTERMEDIATE, "loader")),
        )
        fixture.core.refreshMyTasks()
        assertEquals("loader", fixture.core.state.value.taskVisualMetadataBySpace.getValue("home").statuses.single().iconToken)

        fixture.repository.statuses["home"] = Result.success(
            listOf(MobileTaskStatusDefinition("Open", TaskStatusCategory.COMPLETION, "circle-check")),
        )
        fixture.core.selectSpace("home")
        assertEquals(
            "circle-check",
            fixture.core.state.value.taskVisualMetadataBySpace.getValue("home").statuses.single().iconToken,
        )
        fixture.close()
    }

    private suspend fun signedInFixture(initial: MobileTask): Fixture =
        Fixture(credentials = credentials("account-a")).also {
            it.repository.firstTasks = Page(listOf(initial), null)
            it.repository.selectedTask = initial
            it.core.start()
        }
}

private class Fixture(
    credentials: CredentialBundle?,
    clock: Long = 500,
) {
    val repository = FakeRepository()
    val credentials = FakeCredentialStore(credentials)
    val auth = FakeAuthClient()
    val cache = FakeSnapshotStore()
    val core = MobileAppCore(
        initialServer = ServerProfile("server-a", "https://a.example"),
        repository = repository,
        credentialStore = this.credentials,
        authClient = auth,
        authorizationSession = object : AuthorizationSession {
            override suspend fun authorize(authorizationUrl: String, callbackScheme: String) = error("not used")
        },
        cache = cache,
        clock = { clock },
    )

    fun close() = core.close()
}

private class FakeCredentialStore(initial: CredentialBundle?) : CredentialStore {
    private val values = mutableMapOf<Pair<String, String>, CredentialBundle>()
    private val active = mutableMapOf<String, String>()
    val deleted = mutableListOf<Pair<String, String>>()
    val activeChanges = mutableListOf<Pair<String, String?>>()

    init {
        if (initial != null) {
            values["server-a" to initial.accountId] = initial
            active["server-a"] = initial.accountId
        }
    }

    override suspend fun save(serverId: String, credentials: CredentialBundle) {
        values[serverId to credentials.accountId] = credentials
    }

    override suspend fun load(serverId: String, accountId: String) = values[serverId to accountId]

    override suspend fun delete(serverId: String, accountId: String) {
        deleted += serverId to accountId
        values.remove(serverId to accountId)
    }

    override suspend fun setActiveAccount(serverId: String, accountId: String?) {
        activeChanges += serverId to accountId
        if (accountId == null) active.remove(serverId) else active[serverId] = accountId
    }

    override suspend fun getActiveAccount(serverId: String) = active[serverId]
}

private class FakeAuthClient : RuntimeAuthClient {
    var revokeFailure: Throwable? = null

    override suspend fun authorize(serverId: String, baseUrl: String, session: AuthorizationSession) =
        error("not used")

    override suspend fun refresh(serverId: String, baseUrl: String, credentials: CredentialBundle) = credentials

    override suspend fun revokeAndDelete(serverId: String, baseUrl: String, credentials: CredentialBundle) {
        revokeFailure?.let { throw it }
    }
}

private class FakeRepository : MobileRepository {
    val authConfigScopes = mutableListOf<ServerScope>()
    val sessionScopes = mutableListOf<SessionScope>()
    val taskCursors = mutableListOf<String?>()
    var firstTasks = Page<MobileTask>(emptyList(), null)
    val moreTasks = mutableMapOf<String, Page<MobileTask>>()
    var selectedTask: MobileTask = task("selected", "Selected")
    var updateTaskResult: Result<MobileTask> = Result.success(selectedTask)
    var myTasksHandler: (suspend (SessionScope, String?) -> Page<MobileTask>)? = null
    var authConfigFailure: Throwable? = null
    var spacesResult = emptyList<MobileSpace>()
    val statuses = mutableMapOf<String, Result<List<MobileTaskStatusDefinition>>>()
    val efforts = mutableMapOf<String, Result<List<MobileTaskEffortDefinition>>>()
    val priorities = mutableMapOf<String, Result<List<MobileTaskPriorityDefinition>>>()
    val statusRequests = mutableListOf<String>()
    val effortRequests = mutableListOf<String>()
    val priorityRequests = mutableListOf<String>()
    var metadataHandler: suspend (kind: String, spaceSlug: String) -> Unit = { _, _ -> }

    override suspend fun authConfig(scope: ServerScope): MobileAuthConfig {
        authConfigScopes += scope
        authConfigFailure?.let { throw it }
        return MobileAuthConfig(true, "SSO", false, false)
    }

    override suspend fun currentUser(scope: SessionScope): MobileUser {
        sessionScopes += scope
        return MobileUser(scope.accountId, "user@example.com", "User", false)
    }

    override suspend fun myTasks(scope: SessionScope, cursor: String?, limit: Int?): Page<MobileTask> {
        taskCursors += cursor
        myTasksHandler?.let { return it(scope, cursor) }
        sessionScopes += scope
        return if (cursor == null) firstTasks else moreTasks.getValue(cursor)
    }

    override suspend fun task(scope: SessionScope, spaceSlug: String, taskId: String): MobileTask {
        sessionScopes += scope
        return selectedTask
    }

    override suspend fun updateTask(
        scope: SessionScope,
        spaceSlug: String,
        taskId: String,
        update: MobileTaskUpdate,
    ): MobileTask {
        sessionScopes += scope
        return updateTaskResult.getOrThrow()
    }

    override suspend fun spaces(scope: SessionScope): List<MobileSpace> {
        sessionScopes += scope
        return spacesResult
    }

    override suspend fun taskStatuses(
        scope: SessionScope,
        spaceSlug: String,
    ): List<MobileTaskStatusDefinition> {
        statusRequests += spaceSlug
        metadataHandler("status", spaceSlug)
        return statuses[spaceSlug]?.getOrThrow().orEmpty()
    }

    override suspend fun taskEffortLevels(
        scope: SessionScope,
        spaceSlug: String,
    ): List<MobileTaskEffortDefinition> {
        effortRequests += spaceSlug
        metadataHandler("effort", spaceSlug)
        return efforts[spaceSlug]?.getOrThrow().orEmpty()
    }

    override suspend fun taskPriorityLevels(
        scope: SessionScope,
        spaceSlug: String,
    ): List<MobileTaskPriorityDefinition> {
        priorityRequests += spaceSlug
        metadataHandler("priority", spaceSlug)
        return priorities[spaceSlug]?.getOrThrow().orEmpty()
    }

    override suspend fun spaceTasks(scope: SessionScope, spaceSlug: String, cursor: String?, limit: Int?) =
        Page<MobileTask>(emptyList(), null)

    override suspend fun spaceRecipes(scope: SessionScope, spaceSlug: String, cursor: String?, limit: Int?) =
        Page<MobileRecipe>(emptyList(), null)

    override suspend fun recipe(scope: SessionScope, spaceSlug: String, recipeId: String) =
        MobileRecipe(recipeId, spaceSlug, "Recipe", "", emptyList())

    override suspend fun updateRecipe(
        scope: SessionScope,
        spaceSlug: String,
        recipeId: String,
        update: MobileRecipeUpdate,
    ) = recipe(scope, spaceSlug, recipeId)

    override suspend fun search(scope: SessionScope, query: String, spaceSlug: String?, limit: Int?) =
        emptyList<MobileSearchResult>()

    override suspend fun updateProfile(scope: SessionScope, update: MobileProfileUpdate) = currentUser(scope)
}

private class FakeSnapshotStore : SnapshotStore {
    val taskSnapshots = mutableMapOf<Pair<String, String>, CachedSnapshot<CachedTask>>()
    val taskReads = mutableListOf<Pair<String, String>>()
    val taskWrites = mutableListOf<Pair<Pair<String, String>, CachedSnapshot<CachedTask>>>()
    val clearedAccounts = mutableListOf<Pair<String, String>>()

    override fun replaceTasks(
        serverId: String,
        accountId: String,
        items: List<CachedTask>,
        generatedAt: Long,
        cursor: String?,
        hasMore: Boolean,
    ) {
        val value = CachedSnapshot(items, generatedAt, cursor, hasMore)
        taskSnapshots[serverId to accountId] = value
        taskWrites += (serverId to accountId) to value
    }

    override fun readTasks(serverId: String, accountId: String): CachedSnapshot<CachedTask>? {
        taskReads += serverId to accountId
        return taskSnapshots[serverId to accountId]
    }

    override fun replaceSpaces(serverId: String, accountId: String, items: List<CachedSpace>, generatedAt: Long, cursor: String?, hasMore: Boolean) = Unit
    override fun readSpaces(serverId: String, accountId: String): CachedSnapshot<CachedSpace>? = null
    override fun replaceRecipes(serverId: String, accountId: String, items: List<CachedRecipe>, generatedAt: Long, cursor: String?, hasMore: Boolean) = Unit
    override fun readRecipes(serverId: String, accountId: String): CachedSnapshot<CachedRecipe>? = null
    override fun replaceSearch(serverId: String, accountId: String, query: String, items: List<CachedSearchResult>, generatedAt: Long, cursor: String?, hasMore: Boolean) = Unit
    override fun readSearch(serverId: String, accountId: String, query: String): CachedSnapshot<CachedSearchResult>? = null

    override fun clearAccount(serverId: String, accountId: String) {
        clearedAccounts += serverId to accountId
        taskSnapshots.remove(serverId to accountId)
    }

    override fun clearServer(serverId: String) {
        taskSnapshots.keys.removeAll { it.first == serverId }
    }
}

private fun credentials(accountId: String) = CredentialBundle(
    accessToken = "token-$accountId",
    refreshToken = "refresh-$accountId",
    expiresAtEpochSeconds = null,
    scope = setOf("tasks:read", "tasks:write"),
    accountId = accountId,
)

private fun task(id: String, title: String) = MobileTask(
    id = id,
    spaceSlug = "home",
    title = title,
    description = "Description",
    status = "open",
    effort = null,
    priority = null,
    dueText = null,
    tags = emptyList(),
)

private fun snapshot(
    task: MobileTask,
    generatedAt: Long,
    cursor: String? = null,
) = CachedSnapshot(
    items = listOf(
        CachedTask(
            task.id,
            task.spaceSlug,
            task.title,
            task.description,
            task.status,
            task.effort,
            task.priority,
            task.dueText,
            task.tags,
        ),
    ),
    generatedAt = generatedAt,
    cursor = cursor,
    hasMore = cursor != null,
)
