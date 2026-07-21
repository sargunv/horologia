package dev.horologia.mobile

import android.app.Application
import android.content.pm.PackageManager
import android.os.Build
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.clickable
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.ExitToApp
import androidx.compose.material.icons.filled.AccountCircle
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Clear
import androidx.compose.material.icons.filled.Edit
import androidx.compose.material.icons.filled.Info
import androidx.compose.material.icons.filled.Layers
import androidx.compose.material.icons.filled.MenuBook
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Checkbox
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.ListItem
import androidx.compose.material3.ListItemDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.SearchBar
import androidx.compose.material3.SearchBarDefaults
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.material3.adaptive.ExperimentalMaterial3AdaptiveApi
import androidx.compose.material3.adaptive.currentWindowAdaptiveInfo
import androidx.compose.material3.adaptive.layout.AnimatedPane
import androidx.compose.material3.adaptive.layout.ListDetailPaneScaffoldRole
import androidx.compose.material3.adaptive.layout.PaneAdaptedValue
import androidx.compose.material3.adaptive.navigation.NavigableListDetailPaneScaffold
import androidx.compose.material3.adaptive.navigation.rememberListDetailPaneScaffoldNavigator
import androidx.compose.material3.adaptive.navigationsuite.NavigationSuiteScaffold
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.derivedStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.runtime.snapshotFlow
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.LiveRegionMode
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.liveRegion
import androidx.compose.ui.semantics.selected
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.ui.tooling.preview.Preview
import androidx.compose.ui.unit.dp
import androidx.core.content.ContextCompat
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.compose.viewModel
import androidx.navigation3.runtime.NavKey
import androidx.navigation3.runtime.entryProvider
import androidx.navigation3.runtime.rememberNavBackStack
import androidx.navigation3.ui.NavDisplay
import androidx.window.core.layout.WindowSizeClass
import dev.horologia.mobile.background.AndroidBackgroundScheduler
import dev.horologia.mobile.domain.MobileRecipe
import dev.horologia.mobile.domain.MobileRecipeUpdate
import dev.horologia.mobile.domain.MobileRecipeYield
import dev.horologia.mobile.domain.MobileSearchResult
import dev.horologia.mobile.domain.MobileSpace
import dev.horologia.mobile.domain.MobileTask
import dev.horologia.mobile.domain.MobileTaskDue
import dev.horologia.mobile.domain.MobileTaskUpdate
import dev.horologia.mobile.domain.MobileProfileUpdate
import dev.horologia.mobile.domain.PatchField
import dev.horologia.mobile.runtime.AndroidAppCoreFactory
import dev.horologia.mobile.navigation.HorologiaDeepLinks
import dev.horologia.mobile.navigation.SemanticDestination
import dev.horologia.mobile.runtime.MobileAppCore
import dev.horologia.mobile.runtime.MobileAppError
import dev.horologia.mobile.runtime.MobileAppState
import dev.horologia.mobile.runtime.MobileSessionPhase
import java.text.DateFormat
import java.util.Date
import java.util.TimeZone
import kotlinx.coroutines.FlowPreview
import kotlinx.coroutines.flow.collectLatest
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.debounce
import kotlinx.coroutines.launch
import kotlinx.serialization.Serializable

// ---------------------------------------------------------------------------
// Theme
// ---------------------------------------------------------------------------

/**
 * Brand tokens derived from the existing app identity: deep blue primary
 * (#023C69 family) over green-tinted neutrals (#F6FAF7 / #E7F0EA family).
 * Values not listed fall back to the Material 3 defaults.
 */
private val LightColors =
    lightColorScheme(
        primary = Color(0xFF1F4E73),
        onPrimary = Color(0xFFFFFFFF),
        primaryContainer = Color(0xFFC7DCF3),
        onPrimaryContainer = Color(0xFF001D33),
        secondary = Color(0xFF4F655A),
        onSecondary = Color(0xFFFFFFFF),
        secondaryContainer = Color(0xFFD2E5D9),
        onSecondaryContainer = Color(0xFF0E1F15),
        background = Color(0xFFF6FAF7),
        onBackground = Color(0xFF17201B),
        surface = Color(0xFFF6FAF7),
        onSurface = Color(0xFF17201B),
        surfaceVariant = Color(0xFFDCE7DF),
        onSurfaceVariant = Color(0xFF404943),
        outline = Color(0xFF707972),
    )

private val DarkColors =
    darkColorScheme(
        primary = Color(0xFF8CB4DB),
        onPrimary = Color(0xFF02304F),
        primaryContainer = Color(0xFF1F4E73),
        onPrimaryContainer = Color(0xFFC7DCF3),
        secondary = Color(0xFFB6CCBE),
        onSecondary = Color(0xFF223027),
        secondaryContainer = Color(0xFF384A3F),
        onSecondaryContainer = Color(0xFFD2E5D9),
        background = Color(0xFF101511),
        onBackground = Color(0xFFDCE5DE),
        surface = Color(0xFF101511),
        onSurface = Color(0xFFDCE5DE),
        surfaceVariant = Color(0xFF424B45),
        onSurfaceVariant = Color(0xFFC2CBC4),
        outline = Color(0xFF8C938D),
    )

@Composable
fun HorologiaTheme(content: @Composable () -> Unit) {
    MaterialTheme(
        colorScheme = if (isSystemInDarkTheme()) DarkColors else LightColors,
        content = content,
    )
}

// ---------------------------------------------------------------------------
// ViewModel: owns MobileAppCore for the activity lifecycle
// ---------------------------------------------------------------------------

internal data class PendingDeepLink(
    val deliveryId: Long,
    val destination: SemanticDestination,
    val serverId: String,
    val baseUrl: String,
    val sourceLink: String,
)

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

// ---------------------------------------------------------------------------
// Navigation routes (Navigation 3 keys)
// ---------------------------------------------------------------------------

@Serializable
private sealed interface HorologiaRoute : NavKey {
    @Serializable
    data object Tasks : HorologiaRoute

    @Serializable
    data class TaskDetail(val spaceSlug: String, val taskId: String) : HorologiaRoute

    @Serializable
    data class TaskEdit(val spaceSlug: String, val taskId: String) : HorologiaRoute

    @Serializable
    data object Recipes : HorologiaRoute

    @Serializable
    data class RecipeDetail(val spaceSlug: String, val recipeId: String) : HorologiaRoute

    @Serializable
    data class RecipeEdit(val spaceSlug: String, val recipeId: String) : HorologiaRoute

    @Serializable
    data object Spaces : HorologiaRoute

    @Serializable
    data class SpaceDetail(val spaceSlug: String) : HorologiaRoute

    @Serializable
    data class Search(val query: String? = null) : HorologiaRoute

    @Serializable
    data object Account : HorologiaRoute
}

private data class TopLevelDestination(
    val route: HorologiaRoute,
    val labelRes: Int,
    val icon: ImageVector,
)

private val topLevelDestinations =
    listOf(
        TopLevelDestination(HorologiaRoute.Tasks, R.string.nav_tasks, Icons.Filled.CheckCircle),
        TopLevelDestination(HorologiaRoute.Recipes, R.string.nav_recipes, Icons.Filled.MenuBook),
        TopLevelDestination(HorologiaRoute.Spaces, R.string.nav_spaces, Icons.Filled.Layers),
        TopLevelDestination(HorologiaRoute.Search(), R.string.nav_search, Icons.Filled.Search),
        TopLevelDestination(HorologiaRoute.Account, R.string.nav_account, Icons.Filled.AccountCircle),
    )

// ---------------------------------------------------------------------------
// Root
// ---------------------------------------------------------------------------

@Composable
fun HorologiaApp(viewModel: HorologiaViewModel = viewModel()) {
    val state by viewModel.state.collectAsStateWithLifecycle()
    when (state.phase) {
        MobileSessionPhase.BOOTSTRAP ->
            BootstrapScreen(
                state = state,
                onRetry = viewModel::retryBootstrap,
            )

        MobileSessionPhase.SIGNED_OUT,
        MobileSessionPhase.AUTHORIZING,
        ->
            SignInScreen(
                state = state,
                onConnect = viewModel::connect,
            )

        MobileSessionPhase.SIGNED_IN ->
            SignedInShell(
                state = state,
                viewModel = viewModel,
            )
    }
}

// ---------------------------------------------------------------------------
// Bootstrap / sign-in
// ---------------------------------------------------------------------------

@Composable
private fun BootstrapScreen(state: MobileAppState, onRetry: () -> Unit) {
    Box(
        modifier = Modifier.fillMaxSize().statusBarsPadding().padding(24.dp),
        contentAlignment = Alignment.Center,
    ) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            Text(
                text = stringResource(R.string.app_name),
                style = MaterialTheme.typography.displaySmall,
                color = MaterialTheme.colorScheme.primary,
                modifier = Modifier.semantics { heading() },
            )
            val error = state.error
            if (error != null && !state.loading.bootstrap) {
                ErrorBlock(
                    title = stringResource(R.string.sign_in_error_title),
                    error = error,
                    onRetry = onRetry,
                )
            } else {
                LoadingRow(text = stringResource(R.string.status_preparing))
            }
        }
    }
}

@Composable
private fun SignInScreen(state: MobileAppState, onConnect: (String) -> Unit) {
    var serverUrl by rememberSaveable { mutableStateOf(state.server.baseUrl) }
    val busy = state.phase == MobileSessionPhase.AUTHORIZING || state.loading.bootstrap
    Column(
        modifier =
            Modifier
                .fillMaxSize()
                .statusBarsPadding()
                .verticalScroll(rememberScrollState())
                .padding(24.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        Spacer(modifier = Modifier.height(48.dp))
        Text(
            text = stringResource(R.string.app_name),
            style = MaterialTheme.typography.displaySmall,
            color = MaterialTheme.colorScheme.primary,
            modifier = Modifier.semantics { heading() },
        )
        Text(
            text = stringResource(R.string.sign_in_subtitle),
            style = MaterialTheme.typography.bodyLarge,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        OutlinedTextField(
            value = serverUrl,
            onValueChange = { serverUrl = it },
            modifier = Modifier.fillMaxWidth(),
            label = { Text(stringResource(R.string.server_url_label)) },
            singleLine = true,
            enabled = !busy,
            keyboardOptions =
                KeyboardOptions(
                    keyboardType = KeyboardType.Uri,
                    imeAction = ImeAction.Go,
                ),
            keyboardActions = KeyboardActions(onGo = { onConnect(serverUrl) }),
        )
        val authConfig = state.authConfig
        if (authConfig != null && authConfig.oidcEnabled) {
            Text(
                text = stringResource(R.string.sign_in_method, authConfig.oidcLabel),
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
        val error = state.error
        if (error != null && !busy) {
            Column(modifier = Modifier.semantics { liveRegion = LiveRegionMode.Polite }) {
                Text(
                    text = stringResource(R.string.sign_in_error_title),
                    style = MaterialTheme.typography.titleSmall,
                    color = MaterialTheme.colorScheme.error,
                )
                Text(
                    text = error.message,
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
        Button(
            onClick = { onConnect(serverUrl) },
            modifier = Modifier.fillMaxWidth(),
            enabled = serverUrl.isNotBlank() && !busy,
        ) {
            Text(stringResource(R.string.action_connect))
        }
        if (busy) {
            LoadingRow(
                text =
                    stringResource(
                        if (state.phase == MobileSessionPhase.AUTHORIZING) {
                            R.string.sign_in_authorizing
                        } else {
                            R.string.sign_in_connecting
                        },
                    ),
            )
        }
    }
}

// ---------------------------------------------------------------------------
// Signed-in shell: adaptive navigation suite + Navigation 3 back stack
// ---------------------------------------------------------------------------

@OptIn(ExperimentalMaterial3AdaptiveApi::class)
@Composable
private fun SignedInShell(state: MobileAppState, viewModel: HorologiaViewModel) {
    val backStack = rememberNavBackStack(HorologiaRoute.Tasks)
    val deepLinkDestination by viewModel.deepLinkDestination.collectAsStateWithLifecycle()

    LaunchedEffect(deepLinkDestination) {
        val pendingDeepLink = deepLinkDestination ?: return@LaunchedEffect
        val destination = pendingDeepLink.destination
        if (destination is SemanticDestination.OAuthCallback) {
            viewModel.consumeDeepLink(pendingDeepLink)
            return@LaunchedEffect
        }
        backStack.clear()
        when (destination) {
            SemanticDestination.Tasks -> backStack.add(HorologiaRoute.Tasks)
            is SemanticDestination.Task -> {
                backStack.add(HorologiaRoute.Tasks)
                backStack.add(HorologiaRoute.TaskDetail(destination.spaceSlug, destination.taskId))
            }
            SemanticDestination.Recipes -> backStack.add(HorologiaRoute.Recipes)
            is SemanticDestination.Recipe -> {
                backStack.add(HorologiaRoute.Recipes)
                backStack.add(HorologiaRoute.RecipeDetail(destination.spaceSlug, destination.recipeId))
            }
            SemanticDestination.Spaces -> backStack.add(HorologiaRoute.Spaces)
            is SemanticDestination.Space -> {
                backStack.add(HorologiaRoute.Spaces)
                backStack.add(HorologiaRoute.SpaceDetail(destination.spaceSlug))
            }
            is SemanticDestination.Search -> backStack.add(HorologiaRoute.Search(destination.query))
            SemanticDestination.Account -> backStack.add(HorologiaRoute.Account)
            is SemanticDestination.OAuthCallback -> Unit
        }
        viewModel.consumeDeepLink(pendingDeepLink)
    }

    val windowAdaptiveInfo = currentWindowAdaptiveInfo()
    val isExpandedWidth =
        windowAdaptiveInfo.windowSizeClass
            .isWidthAtLeastBreakpoint(WindowSizeClass.WIDTH_DP_MEDIUM_LOWER_BOUND)

    NotificationPermissionRequest()

    NavigationSuiteScaffold(
        navigationSuiteItems = {
            topLevelDestinations.forEach { destination ->
                item(
                    selected =
                        backStack.firstOrNull() == destination.route ||
                            (backStack.firstOrNull() is HorologiaRoute.Search &&
                                destination.route is HorologiaRoute.Search),
                    onClick = {
                        if (backStack.firstOrNull() != destination.route) {
                            backStack.clear()
                            backStack.add(destination.route)
                        }
                    },
                    icon = { Icon(destination.icon, contentDescription = null) },
                    label = { Text(stringResource(destination.labelRes)) },
                )
            }
        },
    ) {
        NavDisplay(
            backStack = backStack,
            onBack = { backStack.removeLastOrNull() },
            entryProvider =
                entryProvider {
                    entry<HorologiaRoute.Tasks> {
                        TasksDestination(
                            state = state,
                            viewModel = viewModel,
                            isExpandedWidth = isExpandedWidth,
                            onOpenTask = { spaceSlug, taskId ->
                                backStack.add(HorologiaRoute.TaskDetail(spaceSlug, taskId))
                            },
                            onEditTask = { spaceSlug, taskId ->
                                backStack.add(HorologiaRoute.TaskEdit(spaceSlug, taskId))
                            },
                        )
                    }
                    entry<HorologiaRoute.TaskDetail> { key ->
                        TaskDetailScreen(
                            state = state,
                            viewModel = viewModel,
                            spaceSlug = key.spaceSlug,
                            taskId = key.taskId,
                            showBackButton = true,
                            onBack = { backStack.removeLastOrNull() },
                            onEdit = { spaceSlug, taskId ->
                                backStack.add(HorologiaRoute.TaskEdit(spaceSlug, taskId))
                            },
                        )
                    }
                    entry<HorologiaRoute.TaskEdit> { key ->
                        TaskEditScreen(
                            state = state,
                            viewModel = viewModel,
                            spaceSlug = key.spaceSlug,
                            taskId = key.taskId,
                            onDone = { backStack.removeLastOrNull() },
                        )
                    }
                    entry<HorologiaRoute.Recipes> {
                        RecipesDestination(
                            state = state,
                            viewModel = viewModel,
                            isExpandedWidth = isExpandedWidth,
                            onOpenRecipe = { spaceSlug, recipeId ->
                                backStack.add(HorologiaRoute.RecipeDetail(spaceSlug, recipeId))
                            },
                            onEditRecipe = { spaceSlug, recipeId ->
                                backStack.add(HorologiaRoute.RecipeEdit(spaceSlug, recipeId))
                            },
                        )
                    }
                    entry<HorologiaRoute.RecipeDetail> { key ->
                        RecipeDetailScreen(
                            state = state,
                            viewModel = viewModel,
                            spaceSlug = key.spaceSlug,
                            recipeId = key.recipeId,
                            showBackButton = true,
                            onBack = { backStack.removeLastOrNull() },
                            onEdit = { spaceSlug, recipeId ->
                                backStack.add(HorologiaRoute.RecipeEdit(spaceSlug, recipeId))
                            },
                        )
                    }
                    entry<HorologiaRoute.RecipeEdit> { key ->
                        RecipeEditScreen(
                            state = state,
                            viewModel = viewModel,
                            spaceSlug = key.spaceSlug,
                            recipeId = key.recipeId,
                            onDone = { backStack.removeLastOrNull() },
                        )
                    }
                    entry<HorologiaRoute.Spaces> {
                        SpacesDestination(
                            state = state,
                            viewModel = viewModel,
                            isExpandedWidth = isExpandedWidth,
                            onOpenSpace = { spaceSlug ->
                                backStack.add(HorologiaRoute.SpaceDetail(spaceSlug))
                            },
                            onOpenTask = { spaceSlug, taskId ->
                                backStack.add(HorologiaRoute.TaskDetail(spaceSlug, taskId))
                            },
                            onOpenRecipe = { spaceSlug, recipeId ->
                                backStack.add(HorologiaRoute.RecipeDetail(spaceSlug, recipeId))
                            },
                        )
                    }
                    entry<HorologiaRoute.SpaceDetail> { key ->
                        SpaceDetailScreen(
                            state = state,
                            viewModel = viewModel,
                            spaceSlug = key.spaceSlug,
                            showBackButton = true,
                            onBack = { backStack.removeLastOrNull() },
                            onOpenTask = { spaceSlug, taskId ->
                                backStack.add(HorologiaRoute.TaskDetail(spaceSlug, taskId))
                            },
                            onOpenRecipe = { spaceSlug, recipeId ->
                                backStack.add(HorologiaRoute.RecipeDetail(spaceSlug, recipeId))
                            },
                        )
                    }
                    entry<HorologiaRoute.Search> { key ->
                        SearchDestination(
                            state = state,
                            viewModel = viewModel,
                            initialQuery = key.query,
                            onOpenTask = { spaceSlug, taskId ->
                                backStack.add(HorologiaRoute.TaskDetail(spaceSlug, taskId))
                            },
                            onOpenRecipe = { spaceSlug, recipeId ->
                                backStack.add(HorologiaRoute.RecipeDetail(spaceSlug, recipeId))
                            },
                        )
                    }
                    entry<HorologiaRoute.Account> {
                        AccountDestination(
                            state = state,
                            viewModel = viewModel,
                        )
                    }
                },
        )
    }
}

@Composable
private fun NotificationPermissionRequest() {
    if (Build.VERSION.SDK_INT < 33) return
    val context = LocalContext.current
    val launcher =
        rememberLauncherForActivityResult(
            ActivityResultContracts.RequestPermission(),
        ) { }
    LaunchedEffect(Unit) {
        val granted =
            ContextCompat.checkSelfPermission(
                context,
                android.Manifest.permission.POST_NOTIFICATIONS,
            ) == PackageManager.PERMISSION_GRANTED
        if (!granted) {
            launcher.launch(android.Manifest.permission.POST_NOTIFICATIONS)
        }
    }
}

// ---------------------------------------------------------------------------
// Tasks destination
// ---------------------------------------------------------------------------

@Composable
private fun TasksDestination(
    state: MobileAppState,
    viewModel: HorologiaViewModel,
    isExpandedWidth: Boolean,
    onOpenTask: (String, String) -> Unit,
    onEditTask: (String, String) -> Unit,
) {
    LaunchedEffect(Unit) { viewModel.refreshMyTasks() }
    if (isExpandedWidth) {
        TasksListDetail(
            state = state,
            viewModel = viewModel,
            onEditTask = onEditTask,
        )
    } else {
        ScreenFrame(title = stringResource(R.string.nav_tasks)) {
            MyTaskList(
                state = state,
                viewModel = viewModel,
                selectedTaskId = null,
                showSelection = false,
                onTaskClick = { task ->
                    viewModel.selectTask(task.spaceSlug, task.id)
                    onOpenTask(task.spaceSlug, task.id)
                },
            )
        }
    }
}

@OptIn(ExperimentalMaterial3AdaptiveApi::class)
@Composable
private fun TasksListDetail(
    state: MobileAppState,
    viewModel: HorologiaViewModel,
    onEditTask: (String, String) -> Unit,
) {
    val navigator = rememberListDetailPaneScaffoldNavigator<String>()
    val scope = rememberCoroutineScope()
    var selectedTaskId by rememberSaveable { mutableStateOf<String?>(null) }
    val scaffoldValue = navigator.scaffoldValue
    val bothPanesVisible =
        scaffoldValue[ListDetailPaneScaffoldRole.List] == PaneAdaptedValue.Expanded &&
            scaffoldValue[ListDetailPaneScaffoldRole.Detail] == PaneAdaptedValue.Expanded

    ScreenFrame(title = stringResource(R.string.nav_tasks)) {
        NavigableListDetailPaneScaffold(
            navigator = navigator,
            listPane = {
                AnimatedPane {
                    MyTaskList(
                        state = state,
                        viewModel = viewModel,
                        selectedTaskId = selectedTaskId,
                        showSelection = bothPanesVisible,
                        onTaskClick = { task ->
                            selectedTaskId = task.id
                            viewModel.selectTask(task.spaceSlug, task.id)
                            scope.launch {
                                navigator.navigateTo(ListDetailPaneScaffoldRole.Detail, task.id)
                            }
                        },
                    )
                }
            },
            detailPane = {
                AnimatedPane {
                    val detailId = navigator.currentDestination?.contentKey ?: selectedTaskId
                    if (detailId == null) {
                        PanePlaceholder(text = stringResource(R.string.task_select_prompt))
                    } else {
                        TaskDetailBody(
                            state = state,
                            viewModel = viewModel,
                            spaceSlug = null,
                            taskId = detailId,
                            onEdit = onEditTask,
                        )
                    }
                }
            },
        )
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun MyTaskList(
    state: MobileAppState,
    viewModel: HorologiaViewModel,
    selectedTaskId: String?,
    showSelection: Boolean,
    onTaskClick: (MobileTask) -> Unit,
) {
    val listState = rememberLazyListState()
    val shouldLoadMore by remember {
        derivedStateOf {
            val layoutInfo = listState.layoutInfo
            val lastVisible = layoutInfo.visibleItemsInfo.lastOrNull()?.index ?: return@derivedStateOf false
            layoutInfo.totalItemsCount > 0 && lastVisible >= layoutInfo.totalItemsCount - 3
        }
    }
    LaunchedEffect(shouldLoadMore, state.myTasksCursor, state.loading.moreMyTasks) {
        if (shouldLoadMore && state.myTasksCursor != null && !state.loading.moreMyTasks) {
            viewModel.loadMoreMyTasks()
        }
    }

    PullToRefreshBox(
        isRefreshing = state.loading.myTasks && state.myTasks.isNotEmpty(),
        onRefresh = { viewModel.refreshMyTasks() },
        modifier = Modifier.fillMaxSize(),
    ) {
        when {
            state.myTasks.isEmpty() && state.loading.myTasks ->
                LoadingPane()

            state.myTasks.isEmpty() && state.error != null ->
                ErrorPane(
                    title = stringResource(R.string.tasks_error_title),
                    error = state.error,
                    onRetry = { viewModel.refreshMyTasks() },
                )

            state.myTasks.isEmpty() ->
                EmptyPane(
                    title = stringResource(R.string.tasks_empty_title),
                    body = stringResource(R.string.tasks_empty_body),
                )

            else ->
                LazyColumn(
                    state = listState,
                    modifier = Modifier.fillMaxSize(),
                    contentPadding = PaddingValues(bottom = 24.dp),
                ) {
                    item {
                        ListBanners(
                            fromCache = state.myTasksFromCache,
                            generatedAtEpochSeconds = state.myTasksGeneratedAtEpochSeconds,
                            error = state.error,
                            onRetry = { viewModel.refreshMyTasks() },
                        )
                    }
                    items(state.myTasks, key = { it.id }) { task ->
                        TaskRow(
                            task = task,
                            spaceName = state.spaces.firstOrNull { it.slug == task.spaceSlug }?.name,
                            selected = showSelection && task.id == selectedTaskId,
                            onClick = { onTaskClick(task) },
                        )
                    }
                    if (state.myTasksCursor != null || state.loading.moreMyTasks) {
                        item {
                            LoadMoreRow(
                                loading = state.loading.moreMyTasks,
                                onLoadMore = { viewModel.loadMoreMyTasks() },
                            )
                        }
                    }
                }
        }
    }
}

@Composable
private fun TaskRow(
    task: MobileTask,
    spaceName: String?,
    selected: Boolean,
    onClick: () -> Unit,
) {
    val meta =
        listOfNotNull(
            task.status.ifBlank { null },
            task.dueText?.take(10),
            task.priority,
            task.effort,
        ).joinToString(" · ")
    ListItem(
        headlineContent = {
            Text(task.title, maxLines = 1, overflow = TextOverflow.Ellipsis)
        },
        supportingContent = {
            if (meta.isNotEmpty()) {
                Text(meta, maxLines = 1, overflow = TextOverflow.Ellipsis)
            }
        },
        overlineContent = {
            if (!spaceName.isNullOrBlank()) {
                Text(spaceName, maxLines = 1, overflow = TextOverflow.Ellipsis)
            }
        },
        colors =
            if (selected) {
                ListItemDefaults.colors(containerColor = MaterialTheme.colorScheme.secondaryContainer)
            } else {
                ListItemDefaults.colors()
            },
        modifier =
            Modifier
                .fillMaxWidth()
                .clickable(onClick = onClick)
                .semantics { this.selected = selected },
    )
}

// ---------------------------------------------------------------------------
// Task detail + edit
// ---------------------------------------------------------------------------

@Composable
private fun TaskDetailScreen(
    state: MobileAppState,
    viewModel: HorologiaViewModel,
    spaceSlug: String,
    taskId: String,
    showBackButton: Boolean,
    onBack: () -> Unit,
    onEdit: (String, String) -> Unit,
) {
    LaunchedEffect(spaceSlug, taskId) { viewModel.selectTask(spaceSlug, taskId) }
    TaskDetailBody(
        state = state,
        viewModel = viewModel,
        spaceSlug = spaceSlug,
        taskId = taskId,
        onEdit = onEdit,
        showBackButton = showBackButton,
        onBack = onBack,
    )
}

@Composable
private fun TaskDetailBody(
    state: MobileAppState,
    viewModel: HorologiaViewModel,
    spaceSlug: String?,
    taskId: String,
    onEdit: (String, String) -> Unit,
    showBackButton: Boolean = false,
    onBack: () -> Unit = {},
) {
    val task = state.selectedTask?.takeIf { it.id == taskId }
    Column(modifier = Modifier.fillMaxSize().statusBarsPadding()) {
        if (showBackButton) {
            DetailHeader(
                title = task?.title ?: stringResource(R.string.nav_tasks),
                onBack = onBack,
            )
        }
        when {
            task != null ->
                Column(
                    modifier =
                        Modifier
                            .fillMaxSize()
                            .verticalScroll(rememberScrollState())
                            .padding(horizontal = 24.dp, vertical = 16.dp),
                    verticalArrangement = Arrangement.spacedBy(12.dp),
                ) {
                    if (!showBackButton) {
                        Text(
                            text = task.title,
                            style = MaterialTheme.typography.headlineSmall,
                            modifier = Modifier.semantics { heading() },
                        )
                    }
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        Surface(
                            color = MaterialTheme.colorScheme.primaryContainer,
                            shape = MaterialTheme.shapes.small,
                        ) {
                            Text(
                                text = task.status,
                                style = MaterialTheme.typography.labelLarge,
                                color = MaterialTheme.colorScheme.onPrimaryContainer,
                                modifier = Modifier.padding(horizontal = 12.dp, vertical = 6.dp),
                            )
                        }
                    }
                    if (task.description.isNotBlank()) {
                        Text(
                            text = task.description,
                            style = MaterialTheme.typography.bodyLarge,
                            color = MaterialTheme.colorScheme.onSurface,
                        )
                    }
                    MetaRow(label = stringResource(R.string.detail_space_label), value = task.spaceSlug)
                    task.effort?.let { MetaRow(label = stringResource(R.string.detail_effort_label), value = it) }
                    task.priority?.let { MetaRow(label = stringResource(R.string.detail_priority_label), value = it) }
                    task.dueText?.let { MetaRow(label = stringResource(R.string.detail_due_label), value = it.take(10)) }
                    if (task.tags.isNotEmpty()) {
                        MetaRow(
                            label = stringResource(R.string.detail_tags_label),
                            value = task.tags.joinToString(", "),
                        )
                    }
                    Spacer(modifier = Modifier.height(8.dp))
                    Button(onClick = { onEdit(task.spaceSlug, task.id) }) {
                        Icon(
                            Icons.Filled.Edit,
                            contentDescription = null,
                            modifier = Modifier.size(18.dp),
                        )
                        Spacer(modifier = Modifier.width(8.dp))
                        Text(stringResource(R.string.action_edit))
                    }
                }

            state.error != null && !state.loading.task ->
                ErrorPane(
                    title = stringResource(R.string.task_detail_error_title),
                    error = state.error,
                    onRetry = { viewModel.selectTask(spaceSlug ?: task?.spaceSlug ?: "", taskId) },
                )

            else -> LoadingPane()
        }
    }
}

@Composable
private fun TaskEditScreen(
    state: MobileAppState,
    viewModel: HorologiaViewModel,
    spaceSlug: String,
    taskId: String,
    onDone: () -> Unit,
) {
    val task = state.selectedTask?.takeIf { it.id == taskId }
    if (task == null) {
        Column(modifier = Modifier.fillMaxSize().statusBarsPadding()) {
            DetailHeader(title = stringResource(R.string.task_edit_title), onBack = onDone)
            LoadingPane()
        }
        return
    }

    val scope = rememberCoroutineScope()
    var title by rememberSaveable(task.id) { mutableStateOf(task.title) }
    var description by rememberSaveable(task.id) { mutableStateOf(task.description) }
    var status by rememberSaveable(task.id) { mutableStateOf(task.status) }
    var effort by rememberSaveable(task.id) { mutableStateOf(task.effort.orEmpty()) }
    var clearEffort by rememberSaveable(task.id) { mutableStateOf(false) }
    var priority by rememberSaveable(task.id) { mutableStateOf(task.priority.orEmpty()) }
    var clearPriority by rememberSaveable(task.id) { mutableStateOf(false) }
    var dueDate by rememberSaveable(task.id) { mutableStateOf(task.dueText?.take(10).orEmpty()) }
    var clearDue by rememberSaveable(task.id) { mutableStateOf(false) }
    var tags by rememberSaveable(task.id) { mutableStateOf(task.tags.joinToString(", ")) }
    var saveAttempted by rememberSaveable(task.id) { mutableStateOf(false) }

    val saving = state.loading.taskUpdate
    val dueValid = clearDue || dueDate.isBlank() || isValidIsoDate(dueDate.trim())

    Column(modifier = Modifier.fillMaxSize().statusBarsPadding()) {
        DetailHeader(title = stringResource(R.string.task_edit_title), onBack = onDone)
        if (saving) {
            LinearProgressIndicator(modifier = Modifier.fillMaxWidth())
        }
        Column(
            modifier =
                Modifier
                    .fillMaxSize()
                    .verticalScroll(rememberScrollState())
                    .padding(horizontal = 24.dp, vertical = 16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            OutlinedTextField(
                value = title,
                onValueChange = { title = it },
                modifier = Modifier.fillMaxWidth(),
                label = { Text(stringResource(R.string.task_field_title)) },
                singleLine = true,
                enabled = !saving,
            )
            OutlinedTextField(
                value = description,
                onValueChange = { description = it },
                modifier = Modifier.fillMaxWidth(),
                label = { Text(stringResource(R.string.task_field_description)) },
                minLines = 3,
                enabled = !saving,
            )
            OutlinedTextField(
                value = status,
                onValueChange = { status = it },
                modifier = Modifier.fillMaxWidth(),
                label = { Text(stringResource(R.string.task_field_status)) },
                singleLine = true,
                enabled = !saving,
            )
            NullableTextField(
                value = effort,
                onValueChange = { effort = it },
                label = stringResource(R.string.task_field_effort),
                clear = clearEffort,
                onClearChange = { clearEffort = it },
                enabled = !saving,
            )
            NullableTextField(
                value = priority,
                onValueChange = { priority = it },
                label = stringResource(R.string.task_field_priority),
                clear = clearPriority,
                onClearChange = { clearPriority = it },
                enabled = !saving,
            )
            OutlinedTextField(
                value = dueDate,
                onValueChange = { dueDate = it },
                modifier = Modifier.fillMaxWidth(),
                label = { Text(stringResource(R.string.task_field_due_date)) },
                placeholder = { Text(stringResource(R.string.task_field_due_date_hint)) },
                singleLine = true,
                enabled = !saving && !clearDue,
                isError = !dueValid,
                supportingText = {
                    if (!dueValid) {
                        Text(stringResource(R.string.task_field_due_date_invalid))
                    }
                },
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Ascii),
            )
            ClearCheckbox(
                label = stringResource(R.string.field_clear_suffix, stringResource(R.string.task_field_due_date)),
                checked = clearDue,
                onCheckedChange = { clearDue = it },
                enabled = !saving,
            )
            OutlinedTextField(
                value = tags,
                onValueChange = { tags = it },
                modifier = Modifier.fillMaxWidth(),
                label = { Text(stringResource(R.string.task_field_tags)) },
                placeholder = { Text(stringResource(R.string.field_tags_hint)) },
                singleLine = true,
                enabled = !saving,
            )
            if (saveAttempted && state.error != null && !saving) {
                FormError(
                    title = stringResource(R.string.task_save_error_title),
                    error = state.error,
                )
            }
            Button(
                onClick = {
                    saveAttempted = true
                    val parsedTags = tags.split(',').map { it.trim() }.filter { it.isNotEmpty() }
                    val normalizedDue = dueDate.trim()
                    val originalDue = task.dueText?.take(10).orEmpty()
                    val update =
                        MobileTaskUpdate(
                            title = title.takeIf { it != task.title },
                            description = description.takeIf { it != task.description },
                            status = status.takeIf { it != task.status },
                            effort = nullablePatch(effort, clearEffort, task.effort),
                            priority = nullablePatch(priority, clearPriority, task.priority),
                            tags = parsedTags.takeIf { it != task.tags },
                            due =
                                when {
                                    clearDue || (normalizedDue.isBlank() && originalDue.isNotEmpty()) ->
                                        PatchField.Null

                                    normalizedDue.isBlank() || normalizedDue == originalDue ->
                                        PatchField.Absent

                                    else ->
                                        PatchField.Value(
                                            MobileTaskDue(normalizedDue, TimeZone.getDefault().id),
                                        )
                                },
                        )
                    scope.launch {
                        if (viewModel.updateTask(spaceSlug, taskId, update)) {
                            onDone()
                        }
                    }
                },
                modifier = Modifier.fillMaxWidth(),
                enabled = title.isNotBlank() && dueValid && !saving,
            ) {
                Text(stringResource(R.string.action_save))
            }
        }
    }
}

// ---------------------------------------------------------------------------
// Recipes destination
// ---------------------------------------------------------------------------

@Composable
private fun RecipesDestination(
    state: MobileAppState,
    viewModel: HorologiaViewModel,
    isExpandedWidth: Boolean,
    onOpenRecipe: (String, String) -> Unit,
    onEditRecipe: (String, String) -> Unit,
) {
    LaunchedEffect(Unit) { viewModel.loadSpaces() }
    LaunchedEffect(state.spaces, state.selectedSpace) {
        if (state.selectedSpace == null && state.spaces.isNotEmpty()) {
            viewModel.selectSpace(state.spaces.first().slug)
        }
    }

    ScreenFrame(title = stringResource(R.string.nav_recipes)) {
        when {
            state.spaces.isEmpty() && state.loading.spaces -> LoadingPane()

            state.spaces.isEmpty() && state.error != null ->
                ErrorPane(
                    title = stringResource(R.string.recipes_error_title),
                    error = state.error,
                    onRetry = { viewModel.loadSpaces() },
                )

            state.spaces.isEmpty() ->
                EmptyPane(
                    title = stringResource(R.string.spaces_empty_title),
                    body = stringResource(R.string.spaces_empty_body),
                )

            else -> {
                SpaceChipsRow(
                    spaces = state.spaces,
                    selectedSlug = state.selectedSpace?.slug,
                    onSelect = { viewModel.selectSpace(it) },
                )
                if (isExpandedWidth) {
                    RecipesListDetail(
                        state = state,
                        viewModel = viewModel,
                        onEditRecipe = onEditRecipe,
                    )
                } else {
                    SpaceRecipeList(
                        state = state,
                        viewModel = viewModel,
                        selectedRecipeId = null,
                        showSelection = false,
                        onRecipeClick = { recipe ->
                            viewModel.selectRecipe(recipe.spaceSlug, recipe.id)
                            onOpenRecipe(recipe.spaceSlug, recipe.id)
                        },
                    )
                }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3AdaptiveApi::class)
@Composable
private fun RecipesListDetail(
    state: MobileAppState,
    viewModel: HorologiaViewModel,
    onEditRecipe: (String, String) -> Unit,
) {
    val navigator = rememberListDetailPaneScaffoldNavigator<String>()
    val scope = rememberCoroutineScope()
    var selectedRecipeId by rememberSaveable { mutableStateOf<String?>(null) }
    val scaffoldValue = navigator.scaffoldValue
    val bothPanesVisible =
        scaffoldValue[ListDetailPaneScaffoldRole.List] == PaneAdaptedValue.Expanded &&
            scaffoldValue[ListDetailPaneScaffoldRole.Detail] == PaneAdaptedValue.Expanded

    NavigableListDetailPaneScaffold(
        navigator = navigator,
        listPane = {
            AnimatedPane {
                SpaceRecipeList(
                    state = state,
                    viewModel = viewModel,
                    selectedRecipeId = selectedRecipeId,
                    showSelection = bothPanesVisible,
                    onRecipeClick = { recipe ->
                        selectedRecipeId = recipe.id
                        viewModel.selectRecipe(recipe.spaceSlug, recipe.id)
                        scope.launch {
                            navigator.navigateTo(ListDetailPaneScaffoldRole.Detail, recipe.id)
                        }
                    },
                )
            }
        },
        detailPane = {
            AnimatedPane {
                val detailId = navigator.currentDestination?.contentKey ?: selectedRecipeId
                if (detailId == null) {
                    PanePlaceholder(text = stringResource(R.string.recipe_select_prompt))
                } else {
                    RecipeDetailBody(
                        state = state,
                        viewModel = viewModel,
                        recipeId = detailId,
                        onEdit = onEditRecipe,
                    )
                }
            }
        },
    )
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun SpaceRecipeList(
    state: MobileAppState,
    viewModel: HorologiaViewModel,
    selectedRecipeId: String?,
    showSelection: Boolean,
    onRecipeClick: (MobileRecipe) -> Unit,
) {
    val selectedSpace = state.selectedSpace
    if (selectedSpace == null) {
        EmptyPane(
            title = stringResource(R.string.nav_recipes),
            body = stringResource(R.string.recipes_select_space_prompt),
        )
        return
    }
    val listState = rememberLazyListState()
    val shouldLoadMore by remember {
        derivedStateOf {
            val layoutInfo = listState.layoutInfo
            val lastVisible = layoutInfo.visibleItemsInfo.lastOrNull()?.index ?: return@derivedStateOf false
            layoutInfo.totalItemsCount > 0 && lastVisible >= layoutInfo.totalItemsCount - 3
        }
    }
    LaunchedEffect(shouldLoadMore, state.spaceRecipesCursor, state.loading.moreSpaceRecipes) {
        if (shouldLoadMore && state.spaceRecipesCursor != null && !state.loading.moreSpaceRecipes) {
            viewModel.loadMoreSpaceRecipes()
        }
    }

    PullToRefreshBox(
        isRefreshing = state.loading.space && state.spaceRecipes.isNotEmpty(),
        onRefresh = { viewModel.selectSpace(selectedSpace.slug) },
        modifier = Modifier.fillMaxSize(),
    ) {
        when {
            state.spaceRecipes.isEmpty() && state.loading.space -> LoadingPane()

            state.spaceRecipes.isEmpty() && state.error != null ->
                ErrorPane(
                    title = stringResource(R.string.recipes_error_title),
                    error = state.error,
                    onRetry = { viewModel.selectSpace(selectedSpace.slug) },
                )

            state.spaceRecipes.isEmpty() ->
                EmptyPane(
                    title = stringResource(R.string.recipes_empty_title),
                    body = stringResource(R.string.recipes_empty_body),
                )

            else ->
                LazyColumn(
                    state = listState,
                    modifier = Modifier.fillMaxSize(),
                    contentPadding = PaddingValues(bottom = 24.dp),
                ) {
                    item {
                        ListBanners(
                            fromCache = false,
                            generatedAtEpochSeconds = null,
                            error = state.error,
                            onRetry = { viewModel.selectSpace(selectedSpace.slug) },
                        )
                    }
                    items(state.spaceRecipes, key = { it.id }) { recipe ->
                        RecipeRow(
                            recipe = recipe,
                            selected = showSelection && recipe.id == selectedRecipeId,
                            onClick = { onRecipeClick(recipe) },
                        )
                    }
                    if (state.spaceRecipesCursor != null || state.loading.moreSpaceRecipes) {
                        item {
                            LoadMoreRow(
                                loading = state.loading.moreSpaceRecipes,
                                onLoadMore = { viewModel.loadMoreSpaceRecipes() },
                            )
                        }
                    }
                }
        }
    }
}

@Composable
private fun RecipeRow(
    recipe: MobileRecipe,
    selected: Boolean,
    onClick: () -> Unit,
) {
    ListItem(
        headlineContent = {
            Text(recipe.title, maxLines = 1, overflow = TextOverflow.Ellipsis)
        },
        supportingContent = {
            if (recipe.tags.isNotEmpty()) {
                Text(
                    recipe.tags.joinToString(", "),
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
        },
        colors =
            if (selected) {
                ListItemDefaults.colors(containerColor = MaterialTheme.colorScheme.secondaryContainer)
            } else {
                ListItemDefaults.colors()
            },
        modifier =
            Modifier
                .fillMaxWidth()
                .clickable(onClick = onClick)
                .semantics { this.selected = selected },
    )
}

@Composable
private fun SpaceChipsRow(
    spaces: List<MobileSpace>,
    selectedSlug: String?,
    onSelect: (String) -> Unit,
) {
    LazyRow(
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        contentPadding = PaddingValues(horizontal = 24.dp, vertical = 4.dp),
    ) {
        items(spaces, key = { it.slug }) { space ->
            FilterChip(
                selected = space.slug == selectedSlug,
                onClick = { onSelect(space.slug) },
                label = { Text(space.name) },
            )
        }
    }
}

// ---------------------------------------------------------------------------
// Recipe detail + edit
// ---------------------------------------------------------------------------

@Composable
private fun RecipeDetailScreen(
    state: MobileAppState,
    viewModel: HorologiaViewModel,
    spaceSlug: String,
    recipeId: String,
    showBackButton: Boolean,
    onBack: () -> Unit,
    onEdit: (String, String) -> Unit,
) {
    LaunchedEffect(spaceSlug, recipeId) { viewModel.selectRecipe(spaceSlug, recipeId) }
    RecipeDetailBody(
        state = state,
        viewModel = viewModel,
        recipeId = recipeId,
        onEdit = onEdit,
        showBackButton = showBackButton,
        onBack = onBack,
        onRetry = { viewModel.selectRecipe(spaceSlug, recipeId) },
    )
}

@Composable
private fun RecipeDetailBody(
    state: MobileAppState,
    viewModel: HorologiaViewModel,
    recipeId: String,
    onEdit: (String, String) -> Unit,
    showBackButton: Boolean = false,
    onBack: () -> Unit = {},
    onRetry: () -> Unit = {},
) {
    val recipe = state.selectedRecipe?.takeIf { it.id == recipeId }
    Column(modifier = Modifier.fillMaxSize().statusBarsPadding()) {
        if (showBackButton) {
            DetailHeader(
                title = recipe?.title ?: stringResource(R.string.nav_recipes),
                onBack = onBack,
            )
        }
        when {
            recipe != null ->
                Column(
                    modifier =
                        Modifier
                            .fillMaxSize()
                            .verticalScroll(rememberScrollState())
                            .padding(horizontal = 24.dp, vertical = 16.dp),
                    verticalArrangement = Arrangement.spacedBy(12.dp),
                ) {
                    if (!showBackButton) {
                        Text(
                            text = recipe.title,
                            style = MaterialTheme.typography.headlineSmall,
                            modifier = Modifier.semantics { heading() },
                        )
                    }
                    if (recipe.description.isNotBlank()) {
                        Text(
                            text = recipe.description,
                            style = MaterialTheme.typography.bodyLarge,
                            color = MaterialTheme.colorScheme.onSurface,
                        )
                    }
                    MetaRow(label = stringResource(R.string.detail_space_label), value = recipe.spaceSlug)
                    if (recipe.tags.isNotEmpty()) {
                        MetaRow(
                            label = stringResource(R.string.detail_tags_label),
                            value = recipe.tags.joinToString(", "),
                        )
                    }
                    Spacer(modifier = Modifier.height(8.dp))
                    Button(onClick = { onEdit(recipe.spaceSlug, recipe.id) }) {
                        Icon(
                            Icons.Filled.Edit,
                            contentDescription = null,
                            modifier = Modifier.size(18.dp),
                        )
                        Spacer(modifier = Modifier.width(8.dp))
                        Text(stringResource(R.string.action_edit))
                    }
                }

            state.error != null && !state.loading.recipe ->
                ErrorPane(
                    title = stringResource(R.string.recipe_detail_error_title),
                    error = state.error,
                    onRetry = onRetry,
                )

            else -> LoadingPane()
        }
    }
}

@Composable
private fun RecipeEditScreen(
    state: MobileAppState,
    viewModel: HorologiaViewModel,
    spaceSlug: String,
    recipeId: String,
    onDone: () -> Unit,
) {
    val recipe = state.selectedRecipe?.takeIf { it.id == recipeId }
    if (recipe == null) {
        Column(modifier = Modifier.fillMaxSize().statusBarsPadding()) {
            DetailHeader(title = stringResource(R.string.recipe_edit_title), onBack = onDone)
            LoadingPane()
        }
        return
    }

    val scope = rememberCoroutineScope()
    var title by rememberSaveable(recipe.id) { mutableStateOf(recipe.title) }
    var description by rememberSaveable(recipe.id) { mutableStateOf(recipe.description) }
    var tags by rememberSaveable(recipe.id) { mutableStateOf(recipe.tags.joinToString(", ")) }
    var prepMinutes by rememberSaveable(recipe.id) { mutableStateOf("") }
    var clearPrep by rememberSaveable(recipe.id) { mutableStateOf(false) }
    var cookMinutes by rememberSaveable(recipe.id) { mutableStateOf("") }
    var clearCook by rememberSaveable(recipe.id) { mutableStateOf(false) }
    var yieldAmount by rememberSaveable(recipe.id) { mutableStateOf("") }
    var yieldUnit by rememberSaveable(recipe.id) { mutableStateOf("") }
    var clearYield by rememberSaveable(recipe.id) { mutableStateOf(false) }
    var saveAttempted by rememberSaveable(recipe.id) { mutableStateOf(false) }

    val saving = state.loading.recipeUpdate
    val prepValid = prepMinutes.isBlank() || prepMinutes.toIntOrNull() != null
    val cookValid = cookMinutes.isBlank() || cookMinutes.toIntOrNull() != null
    val yieldValid = yieldAmount.isBlank() || yieldAmount.toDoubleOrNull() != null

    Column(modifier = Modifier.fillMaxSize().statusBarsPadding()) {
        DetailHeader(title = stringResource(R.string.recipe_edit_title), onBack = onDone)
        if (saving) {
            LinearProgressIndicator(modifier = Modifier.fillMaxWidth())
        }
        Column(
            modifier =
                Modifier
                    .fillMaxSize()
                    .verticalScroll(rememberScrollState())
                    .padding(horizontal = 24.dp, vertical = 16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            OutlinedTextField(
                value = title,
                onValueChange = { title = it },
                modifier = Modifier.fillMaxWidth(),
                label = { Text(stringResource(R.string.recipe_field_title)) },
                singleLine = true,
                enabled = !saving,
            )
            OutlinedTextField(
                value = description,
                onValueChange = { description = it },
                modifier = Modifier.fillMaxWidth(),
                label = { Text(stringResource(R.string.recipe_field_description)) },
                minLines = 3,
                enabled = !saving,
            )
            OutlinedTextField(
                value = tags,
                onValueChange = { tags = it },
                modifier = Modifier.fillMaxWidth(),
                label = { Text(stringResource(R.string.task_field_tags)) },
                placeholder = { Text(stringResource(R.string.field_tags_hint)) },
                singleLine = true,
                enabled = !saving,
            )
            Text(
                text = stringResource(R.string.recipe_optional_details),
                style = MaterialTheme.typography.titleSmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.semantics { heading() },
            )
            MinutesField(
                value = prepMinutes,
                onValueChange = { prepMinutes = it },
                label = stringResource(R.string.recipe_field_prep_minutes),
                valid = prepValid,
                clear = clearPrep,
                onClearChange = { clearPrep = it },
                enabled = !saving,
            )
            MinutesField(
                value = cookMinutes,
                onValueChange = { cookMinutes = it },
                label = stringResource(R.string.recipe_field_cook_minutes),
                valid = cookValid,
                clear = clearCook,
                onClearChange = { clearCook = it },
                enabled = !saving,
            )
            OutlinedTextField(
                value = yieldAmount,
                onValueChange = { yieldAmount = it },
                modifier = Modifier.fillMaxWidth(),
                label = { Text(stringResource(R.string.recipe_field_yield_amount)) },
                singleLine = true,
                enabled = !saving && !clearYield,
                isError = !yieldValid,
                supportingText = {
                    if (!yieldValid) {
                        Text(stringResource(R.string.recipe_field_yield_invalid))
                    }
                },
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
            )
            OutlinedTextField(
                value = yieldUnit,
                onValueChange = { yieldUnit = it },
                modifier = Modifier.fillMaxWidth(),
                label = { Text(stringResource(R.string.recipe_field_yield_unit)) },
                singleLine = true,
                enabled = !saving && !clearYield,
            )
            ClearCheckbox(
                label = stringResource(R.string.field_clear_suffix, stringResource(R.string.recipe_field_yield_amount)),
                checked = clearYield,
                onCheckedChange = { clearYield = it },
                enabled = !saving,
            )
            if (saveAttempted && state.error != null && !saving) {
                FormError(
                    title = stringResource(R.string.recipe_save_error_title),
                    error = state.error,
                )
            }
            Button(
                onClick = {
                    saveAttempted = true
                    val parsedTags = tags.split(',').map { it.trim() }.filter { it.isNotEmpty() }
                    val update =
                        MobileRecipeUpdate(
                            title = title.takeIf { it != recipe.title },
                            description = description.takeIf { it != recipe.description },
                            tags = parsedTags.takeIf { it != recipe.tags },
                            prepMinutes = optionalIntPatch(prepMinutes, clearPrep),
                            cookMinutes = optionalIntPatch(cookMinutes, clearCook),
                            yield =
                                when {
                                    clearYield -> PatchField.Null
                                    yieldAmount.isBlank() -> PatchField.Absent
                                    else ->
                                        PatchField.Value(
                                            MobileRecipeYield(
                                                amount = yieldAmount.trim().toDouble(),
                                                unit = yieldUnit.trim(),
                                            ),
                                        )
                                },
                        )
                    scope.launch {
                        if (viewModel.updateRecipe(spaceSlug, recipeId, update)) {
                            onDone()
                        }
                    }
                },
                modifier = Modifier.fillMaxWidth(),
                enabled = title.isNotBlank() && prepValid && cookValid && yieldValid && !saving,
            ) {
                Text(stringResource(R.string.action_save))
            }
        }
    }
}

// ---------------------------------------------------------------------------
// Spaces destination
// ---------------------------------------------------------------------------

@Composable
private fun SpacesDestination(
    state: MobileAppState,
    viewModel: HorologiaViewModel,
    isExpandedWidth: Boolean,
    onOpenSpace: (String) -> Unit,
    onOpenTask: (String, String) -> Unit,
    onOpenRecipe: (String, String) -> Unit,
) {
    LaunchedEffect(Unit) { viewModel.loadSpaces() }
    if (isExpandedWidth) {
        SpacesListDetail(
            state = state,
            viewModel = viewModel,
            onOpenTask = onOpenTask,
            onOpenRecipe = onOpenRecipe,
        )
    } else {
        ScreenFrame(title = stringResource(R.string.nav_spaces)) {
            SpaceList(
                state = state,
                viewModel = viewModel,
                selectedSlug = null,
                showSelection = false,
                onSpaceClick = { space ->
                    viewModel.selectSpace(space.slug)
                    onOpenSpace(space.slug)
                },
            )
        }
    }
}

@OptIn(ExperimentalMaterial3AdaptiveApi::class)
@Composable
private fun SpacesListDetail(
    state: MobileAppState,
    viewModel: HorologiaViewModel,
    onOpenTask: (String, String) -> Unit,
    onOpenRecipe: (String, String) -> Unit,
) {
    val navigator = rememberListDetailPaneScaffoldNavigator<String>()
    val scope = rememberCoroutineScope()
    var selectedSlug by rememberSaveable { mutableStateOf<String?>(null) }
    val scaffoldValue = navigator.scaffoldValue
    val bothPanesVisible =
        scaffoldValue[ListDetailPaneScaffoldRole.List] == PaneAdaptedValue.Expanded &&
            scaffoldValue[ListDetailPaneScaffoldRole.Detail] == PaneAdaptedValue.Expanded

    ScreenFrame(title = stringResource(R.string.nav_spaces)) {
        NavigableListDetailPaneScaffold(
            navigator = navigator,
            listPane = {
                AnimatedPane {
                    SpaceList(
                        state = state,
                        viewModel = viewModel,
                        selectedSlug = selectedSlug,
                        showSelection = bothPanesVisible,
                        onSpaceClick = { space ->
                            selectedSlug = space.slug
                            viewModel.selectSpace(space.slug)
                            scope.launch {
                                navigator.navigateTo(ListDetailPaneScaffoldRole.Detail, space.slug)
                            }
                        },
                    )
                }
            },
            detailPane = {
                AnimatedPane {
                    val detailSlug = navigator.currentDestination?.contentKey ?: selectedSlug
                    if (detailSlug == null) {
                        PanePlaceholder(text = stringResource(R.string.space_select_prompt))
                    } else {
                        SpaceDetailBody(
                            state = state,
                            viewModel = viewModel,
                            spaceSlug = detailSlug,
                            onOpenTask = onOpenTask,
                            onOpenRecipe = onOpenRecipe,
                        )
                    }
                }
            },
        )
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun SpaceList(
    state: MobileAppState,
    viewModel: HorologiaViewModel,
    selectedSlug: String?,
    showSelection: Boolean,
    onSpaceClick: (MobileSpace) -> Unit,
) {
    PullToRefreshBox(
        isRefreshing = state.loading.spaces && state.spaces.isNotEmpty(),
        onRefresh = { viewModel.loadSpaces() },
        modifier = Modifier.fillMaxSize(),
    ) {
        when {
            state.spaces.isEmpty() && state.loading.spaces -> LoadingPane()

            state.spaces.isEmpty() && state.error != null ->
                ErrorPane(
                    title = stringResource(R.string.spaces_error_title),
                    error = state.error,
                    onRetry = { viewModel.loadSpaces() },
                )

            state.spaces.isEmpty() ->
                EmptyPane(
                    title = stringResource(R.string.spaces_empty_title),
                    body = stringResource(R.string.spaces_empty_body),
                )

            else ->
                LazyColumn(
                    modifier = Modifier.fillMaxSize(),
                    contentPadding = PaddingValues(bottom = 24.dp),
                ) {
                    item {
                        ListBanners(
                            fromCache = state.spacesFromCache,
                            generatedAtEpochSeconds = state.spacesGeneratedAtEpochSeconds,
                            error = state.error,
                            onRetry = { viewModel.loadSpaces() },
                        )
                    }
                    items(state.spaces, key = { it.slug }) { space ->
                        SpaceRow(
                            space = space,
                            selected = showSelection && space.slug == selectedSlug,
                            onClick = { onSpaceClick(space) },
                        )
                    }
                }
        }
    }
}

@Composable
private fun SpaceRow(
    space: MobileSpace,
    selected: Boolean,
    onClick: () -> Unit,
) {
    ListItem(
        headlineContent = {
            Text(space.name, maxLines = 1, overflow = TextOverflow.Ellipsis)
        },
        supportingContent = {
            if (space.name != space.slug) {
                Text(space.slug, maxLines = 1, overflow = TextOverflow.Ellipsis)
            }
        },
        colors =
            if (selected) {
                ListItemDefaults.colors(containerColor = MaterialTheme.colorScheme.secondaryContainer)
            } else {
                ListItemDefaults.colors()
            },
        modifier =
            Modifier
                .fillMaxWidth()
                .clickable(onClick = onClick)
                .semantics { this.selected = selected },
    )
}

@Composable
private fun SpaceDetailScreen(
    state: MobileAppState,
    viewModel: HorologiaViewModel,
    spaceSlug: String,
    showBackButton: Boolean,
    onBack: () -> Unit,
    onOpenTask: (String, String) -> Unit,
    onOpenRecipe: (String, String) -> Unit,
) {
    Column(modifier = Modifier.fillMaxSize().statusBarsPadding()) {
        if (showBackButton) {
            DetailHeader(
                title = state.selectedSpace?.takeIf { it.slug == spaceSlug }?.name ?: spaceSlug,
                onBack = onBack,
            )
        }
        SpaceDetailBody(
            state = state,
            viewModel = viewModel,
            spaceSlug = spaceSlug,
            onOpenTask = onOpenTask,
            onOpenRecipe = onOpenRecipe,
            showHeading = !showBackButton,
        )
    }
}

@Composable
private fun SpaceDetailBody(
    state: MobileAppState,
    viewModel: HorologiaViewModel,
    spaceSlug: String,
    onOpenTask: (String, String) -> Unit,
    onOpenRecipe: (String, String) -> Unit,
    showHeading: Boolean = true,
) {
    LaunchedEffect(spaceSlug) {
        if (state.selectedSpace?.slug != spaceSlug) {
            viewModel.selectSpace(spaceSlug)
        }
    }
    val space = state.selectedSpace?.takeIf { it.slug == spaceSlug }
    val ownsSublists = space != null
    Column(
        modifier =
            Modifier
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .padding(horizontal = 24.dp, vertical = 16.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        if (showHeading) {
            Text(
                text = space?.name ?: spaceSlug,
                style = MaterialTheme.typography.headlineSmall,
                modifier = Modifier.semantics { heading() },
            )
        }
        val error = state.error
        if (error != null && !state.loading.space && !ownsSublists) {
            ErrorBlock(
                title = stringResource(R.string.spaces_error_title),
                error = error,
                onRetry = { viewModel.selectSpace(spaceSlug) },
            )
            return@Column
        }
        if (state.loading.space && !ownsSublists) {
            LoadingRow(text = stringResource(R.string.loading_indicator_description))
        }

        SectionHeader(text = stringResource(R.string.space_tasks_section))
        when {
            !ownsSublists -> Unit
            state.spaceTasks.isEmpty() && state.loading.space ->
                LoadingRow(text = stringResource(R.string.loading_indicator_description))

            state.spaceTasks.isEmpty() ->
                Text(
                    text = stringResource(R.string.space_tasks_empty),
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )

            else -> {
                state.spaceTasks.forEach { task ->
                    SubListRow(
                        title = task.title,
                        subtitle = task.status,
                        onClick = {
                            viewModel.selectTask(task.spaceSlug, task.id)
                            onOpenTask(task.spaceSlug, task.id)
                        },
                    )
                }
                if (state.spaceTasksCursor != null || state.loading.moreSpaceTasks) {
                    LoadMoreRow(
                        loading = state.loading.moreSpaceTasks,
                        onLoadMore = { viewModel.loadMoreSpaceTasks() },
                    )
                }
            }
        }

        SectionHeader(text = stringResource(R.string.space_recipes_section))
        when {
            !ownsSublists -> Unit
            state.spaceRecipes.isEmpty() && state.loading.space ->
                LoadingRow(text = stringResource(R.string.loading_indicator_description))

            state.spaceRecipes.isEmpty() ->
                Text(
                    text = stringResource(R.string.space_recipes_empty),
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )

            else -> {
                state.spaceRecipes.forEach { recipe ->
                    SubListRow(
                        title = recipe.title,
                        subtitle = recipe.tags.joinToString(", ").ifBlank { null },
                        onClick = {
                            viewModel.selectRecipe(recipe.spaceSlug, recipe.id)
                            onOpenRecipe(recipe.spaceSlug, recipe.id)
                        },
                    )
                }
                if (state.spaceRecipesCursor != null || state.loading.moreSpaceRecipes) {
                    LoadMoreRow(
                        loading = state.loading.moreSpaceRecipes,
                        onLoadMore = { viewModel.loadMoreSpaceRecipes() },
                    )
                }
            }
        }
    }
}

@Composable
private fun SubListRow(
    title: String,
    subtitle: String?,
    onClick: () -> Unit,
) {
    ListItem(
        headlineContent = {
            Text(title, maxLines = 1, overflow = TextOverflow.Ellipsis)
        },
        supportingContent = {
            if (!subtitle.isNullOrBlank()) {
                Text(subtitle, maxLines = 1, overflow = TextOverflow.Ellipsis)
            }
        },
        modifier = Modifier.fillMaxWidth().clickable(onClick = onClick),
    )
}

// ---------------------------------------------------------------------------
// Search destination
// ---------------------------------------------------------------------------

@OptIn(ExperimentalMaterial3Api::class, FlowPreview::class)
@Composable
private fun SearchDestination(
    state: MobileAppState,
    viewModel: HorologiaViewModel,
    initialQuery: String?,
    onOpenTask: (String, String) -> Unit,
    onOpenRecipe: (String, String) -> Unit,
) {
    var query by rememberSaveable(initialQuery) { mutableStateOf(initialQuery ?: state.searchQuery) }
    var expanded by rememberSaveable { mutableStateOf(false) }

    LaunchedEffect(Unit) {
        snapshotFlow { query }
            .debounce(300)
            .collectLatest { viewModel.submitSearch(it) }
    }

    Column(modifier = Modifier.fillMaxSize().statusBarsPadding()) {
        Text(
            text = stringResource(R.string.nav_search),
            style = MaterialTheme.typography.headlineMedium,
            modifier =
                Modifier
                    .padding(horizontal = 24.dp, vertical = 16.dp)
                    .semantics { heading() },
        )
        SearchBar(
            inputField = {
                SearchBarDefaults.InputField(
                    query = query,
                    onQueryChange = { query = it },
                    onSearch = {
                        viewModel.submitSearch(query)
                        expanded = false
                    },
                    expanded = expanded,
                    onExpandedChange = { expanded = it },
                    placeholder = { Text(stringResource(R.string.search_hint)) },
                    leadingIcon = {
                        Icon(Icons.Filled.Search, contentDescription = null)
                    },
                    trailingIcon = {
                        if (query.isNotEmpty()) {
                            IconButton(onClick = { query = "" }) {
                                Icon(
                                    Icons.Filled.Clear,
                                    contentDescription = stringResource(R.string.action_clear_text),
                                )
                            }
                        }
                    },
                )
            },
            expanded = expanded,
            onExpandedChange = { expanded = it },
        ) {
            SearchResults(
                state = state,
                onOpenResult = { result ->
                    expanded = false
                    when (result.kind) {
                        "task" -> {
                            viewModel.selectTask(result.spaceSlug, result.id)
                            onOpenTask(result.spaceSlug, result.id)
                        }

                        else -> {
                            viewModel.selectRecipe(result.spaceSlug, result.id)
                            onOpenRecipe(result.spaceSlug, result.id)
                        }
                    }
                },
                onRetry = { viewModel.submitSearch(query) },
            )
        }
        if (!expanded) {
            SearchResults(
                state = state,
                onOpenResult = { result ->
                    when (result.kind) {
                        "task" -> {
                            viewModel.selectTask(result.spaceSlug, result.id)
                            onOpenTask(result.spaceSlug, result.id)
                        }

                        else -> {
                            viewModel.selectRecipe(result.spaceSlug, result.id)
                            onOpenRecipe(result.spaceSlug, result.id)
                        }
                    }
                },
                onRetry = { viewModel.submitSearch(query) },
            )
        }
    }
}

@Composable
private fun SearchResults(
    state: MobileAppState,
    onOpenResult: (MobileSearchResult) -> Unit,
    onRetry: () -> Unit,
) {
    when {
        state.searchQuery.isBlank() ->
            EmptyPane(
                title = stringResource(R.string.search_prompt_title),
                body = stringResource(R.string.search_prompt_body),
            )

        state.searchResults.isEmpty() && state.loading.search -> LoadingPane()

        state.searchResults.isEmpty() && state.error != null ->
            ErrorPane(
                title = stringResource(R.string.search_error_title),
                error = state.error,
                onRetry = onRetry,
            )

        state.searchResults.isEmpty() ->
            EmptyPane(
                title = stringResource(R.string.search_no_results, state.searchQuery),
                body = null,
            )

        else ->
            LazyColumn(
                modifier = Modifier.fillMaxSize(),
                contentPadding = PaddingValues(bottom = 24.dp),
            ) {
                item {
                    ListBanners(
                        fromCache = state.searchFromCache,
                        generatedAtEpochSeconds = state.searchGeneratedAtEpochSeconds,
                        error = state.error,
                        onRetry = onRetry,
                    )
                }
                val grouped = state.searchResults.groupBy { it.kind }
                grouped.forEach { (kind, results) ->
                    item(key = "header-$kind") {
                        SectionHeader(
                            text =
                                stringResource(
                                    when (kind) {
                                        "task" -> R.string.search_kind_task
                                        "recipe" -> R.string.search_kind_recipe
                                        else -> R.string.search_hint
                                    },
                                ),
                        )
                    }
                    items(results, key = { "$kind:${it.id}" }) { result ->
                        ListItem(
                            headlineContent = {
                                Text(result.title, maxLines = 1, overflow = TextOverflow.Ellipsis)
                            },
                            supportingContent = {
                                if (result.detail.isNotBlank()) {
                                    Text(result.detail, maxLines = 1, overflow = TextOverflow.Ellipsis)
                                }
                            },
                            overlineContent = {
                                Text(result.spaceSlug, maxLines = 1, overflow = TextOverflow.Ellipsis)
                            },
                            modifier = Modifier.fillMaxWidth().clickable { onOpenResult(result) },
                        )
                    }
                }
            }
    }
}

// ---------------------------------------------------------------------------
// Account destination
// ---------------------------------------------------------------------------

@Composable
private fun AccountDestination(
    state: MobileAppState,
    viewModel: HorologiaViewModel,
) {
    val scope = rememberCoroutineScope()
    val user = state.user
    var name by rememberSaveable(user?.id) { mutableStateOf(user?.name.orEmpty()) }
    var email by rememberSaveable(user?.id) { mutableStateOf(user?.email.orEmpty()) }
    var saveAttempted by rememberSaveable(user?.id) { mutableStateOf(false) }
    var confirmSignOut by rememberSaveable { mutableStateOf(false) }
    val saving = state.loading.accountUpdate

    Column(
        modifier =
            Modifier
                .fillMaxSize()
                .statusBarsPadding()
                .verticalScroll(rememberScrollState())
                .padding(horizontal = 24.dp, vertical = 16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Text(
            text = stringResource(R.string.nav_account),
            style = MaterialTheme.typography.headlineMedium,
            modifier = Modifier.semantics { heading() },
        )
        SectionHeader(text = stringResource(R.string.account_profile_section))
        OutlinedTextField(
            value = name,
            onValueChange = { name = it },
            modifier = Modifier.fillMaxWidth(),
            label = { Text(stringResource(R.string.account_name_label)) },
            singleLine = true,
            enabled = !saving,
        )
        OutlinedTextField(
            value = email,
            onValueChange = { email = it },
            modifier = Modifier.fillMaxWidth(),
            label = { Text(stringResource(R.string.account_email_label)) },
            singleLine = true,
            enabled = !saving,
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Email),
        )
        if (user?.isOwner == true) {
            Surface(
                color = MaterialTheme.colorScheme.secondaryContainer,
                shape = MaterialTheme.shapes.small,
            ) {
                Text(
                    text = stringResource(R.string.account_owner_badge),
                    style = MaterialTheme.typography.labelLarge,
                    color = MaterialTheme.colorScheme.onSecondaryContainer,
                    modifier = Modifier.padding(horizontal = 12.dp, vertical = 6.dp),
                )
            }
        }
        if (saveAttempted && state.error != null && !saving) {
            FormError(
                title = stringResource(R.string.account_save_error_title),
                error = state.error,
            )
        }
        Button(
            onClick = {
                saveAttempted = true
                scope.launch {
                    viewModel.updateProfile(name.trim(), email.trim())
                }
            },
            modifier = Modifier.fillMaxWidth(),
            enabled =
                !saving && name.isNotBlank() &&
                    (name.trim() != user?.name || email.trim() != user?.email),
        ) {
            Text(stringResource(R.string.action_save))
        }

        HorizontalDivider(modifier = Modifier.padding(vertical = 8.dp))
        SectionHeader(text = stringResource(R.string.account_server_section))
        MetaRow(label = stringResource(R.string.account_server_url_label), value = state.server.baseUrl)
        state.accountId?.let {
            MetaRow(label = stringResource(R.string.account_id_label), value = it)
        }
        val authConfig = state.authConfig
        if (authConfig != null && authConfig.oidcEnabled) {
            MetaRow(
                label = stringResource(R.string.account_sign_in_method_label),
                value = authConfig.oidcLabel,
            )
        }

        Spacer(modifier = Modifier.height(16.dp))
        OutlinedButton(
            onClick = { confirmSignOut = true },
            modifier = Modifier.fillMaxWidth(),
            colors =
                ButtonDefaults.outlinedButtonColors(
                    contentColor = MaterialTheme.colorScheme.error,
                ),
        ) {
            Icon(
                Icons.AutoMirrored.Filled.ExitToApp,
                contentDescription = null,
                modifier = Modifier.size(18.dp),
            )
            Spacer(modifier = Modifier.width(8.dp))
            Text(stringResource(R.string.action_sign_out))
        }
    }

    if (confirmSignOut) {
        AlertDialog(
            onDismissRequest = { confirmSignOut = false },
            title = { Text(stringResource(R.string.sign_out_confirm_title)) },
            text = { Text(stringResource(R.string.sign_out_confirm_body)) },
            confirmButton = {
                TextButton(
                    onClick = {
                        confirmSignOut = false
                        viewModel.signOut()
                    },
                    colors =
                        ButtonDefaults.textButtonColors(
                            contentColor = MaterialTheme.colorScheme.error,
                        ),
                ) {
                    Text(stringResource(R.string.action_confirm_sign_out))
                }
            },
            dismissButton = {
                TextButton(onClick = { confirmSignOut = false }) {
                    Text(stringResource(R.string.action_cancel))
                }
            },
        )
    }
}

// ---------------------------------------------------------------------------
// Shared building blocks
// ---------------------------------------------------------------------------

@Composable
private fun ScreenFrame(title: String, content: @Composable ColumnScope.() -> Unit) {
    Column(modifier = Modifier.fillMaxSize().statusBarsPadding()) {
        Text(
            text = title,
            style = MaterialTheme.typography.headlineMedium,
            modifier =
                Modifier
                    .padding(horizontal = 24.dp, vertical = 16.dp)
                    .semantics { heading() },
        )
        content()
    }
}

@Composable
private fun DetailHeader(title: String, onBack: () -> Unit) {
    Row(
        modifier = Modifier.fillMaxWidth().padding(horizontal = 8.dp, vertical = 4.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        IconButton(onClick = onBack) {
            Icon(
                Icons.AutoMirrored.Filled.ArrowBack,
                contentDescription = stringResource(R.string.action_back),
            )
        }
        Text(
            text = title,
            style = MaterialTheme.typography.titleLarge,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
            modifier = Modifier.semantics { heading() },
        )
    }
}

@Composable
private fun LoadingRow(text: String) {
    Row(
        horizontalArrangement = Arrangement.spacedBy(16.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        CircularProgressIndicator(
            modifier = Modifier.size(24.dp).semantics { contentDescription = text },
            strokeWidth = 2.dp,
        )
        Text(
            text = text,
            style = MaterialTheme.typography.bodyLarge,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}

@Composable
private fun LoadingPane() {
    Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
        CircularProgressIndicator(
            modifier =
                Modifier.semantics {
                    contentDescription = ""
                },
        )
    }
}

@Composable
private fun ErrorBlock(
    title: String,
    error: MobileAppError,
    onRetry: () -> Unit,
) {
    Column(
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(8.dp),
        modifier = Modifier.semantics { liveRegion = LiveRegionMode.Polite },
    ) {
        Text(
            text = title,
            style = MaterialTheme.typography.titleMedium,
            color = MaterialTheme.colorScheme.error,
        )
        Text(
            text = error.message,
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        TextButton(onClick = onRetry) {
            Text(stringResource(R.string.action_retry))
        }
    }
}

@Composable
private fun ErrorPane(
    title: String,
    error: MobileAppError?,
    onRetry: () -> Unit,
) {
    Box(modifier = Modifier.fillMaxSize().padding(24.dp), contentAlignment = Alignment.Center) {
        ErrorBlock(
            title = title,
            error = error ?: MobileAppError(message = ""),
            onRetry = onRetry,
        )
    }
}

@Composable
private fun EmptyPane(title: String, body: String?) {
    Box(modifier = Modifier.fillMaxSize().padding(24.dp), contentAlignment = Alignment.Center) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            Text(
                text = title,
                style = MaterialTheme.typography.titleMedium,
                color = MaterialTheme.colorScheme.onSurface,
            )
            if (body != null) {
                Text(
                    text = body,
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
    }
}

@Composable
private fun PanePlaceholder(text: String) {
    Box(modifier = Modifier.fillMaxSize().padding(24.dp), contentAlignment = Alignment.Center) {
        Text(
            text = text,
            style = MaterialTheme.typography.bodyLarge,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}

@Composable
private fun ListBanners(
    fromCache: Boolean,
    generatedAtEpochSeconds: Long?,
    error: MobileAppError?,
    onRetry: () -> Unit,
) {
    Column(modifier = Modifier.fillMaxWidth().padding(horizontal = 24.dp, vertical = 4.dp)) {
        if (fromCache) {
            Surface(
                color = MaterialTheme.colorScheme.surfaceVariant,
                shape = MaterialTheme.shapes.small,
                modifier = Modifier.fillMaxWidth(),
            ) {
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                    modifier = Modifier.padding(horizontal = 12.dp, vertical = 8.dp),
                ) {
                    Icon(
                        Icons.Filled.Info,
                        contentDescription = null,
                        modifier = Modifier.size(16.dp),
                        tint = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                    Text(
                        text =
                            stringResource(
                                R.string.cached_banner,
                                formatEpochSeconds(generatedAtEpochSeconds),
                            ),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
        }
        if (error != null) {
            Surface(
                color = MaterialTheme.colorScheme.errorContainer,
                shape = MaterialTheme.shapes.small,
                modifier = Modifier.fillMaxWidth().padding(top = 4.dp),
            ) {
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    modifier = Modifier.padding(horizontal = 12.dp, vertical = 4.dp),
                ) {
                    Text(
                        text = error.message,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onErrorContainer,
                        modifier = Modifier.weight(1f),
                    )
                    TextButton(onClick = onRetry) {
                        Text(stringResource(R.string.action_retry))
                    }
                }
            }
        }
    }
}

@Composable
private fun LoadMoreRow(loading: Boolean, onLoadMore: () -> Unit) {
    Box(
        modifier = Modifier.fillMaxWidth().padding(vertical = 8.dp),
        contentAlignment = Alignment.Center,
    ) {
        if (loading) {
            CircularProgressIndicator(modifier = Modifier.size(24.dp), strokeWidth = 2.dp)
        } else {
            TextButton(onClick = onLoadMore) {
                Text(stringResource(R.string.action_load_more))
            }
        }
    }
}

@Composable
private fun SectionHeader(text: String) {
    Text(
        text = text,
        style = MaterialTheme.typography.titleSmall,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
        modifier =
            Modifier
                .padding(top = 8.dp)
                .semantics { heading() },
    )
}

@Composable
private fun MetaRow(label: String, value: String) {
    Row(modifier = Modifier.fillMaxWidth()) {
        Text(
            text = label,
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.width(120.dp),
        )
        Text(
            text = value,
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurface,
        )
    }
}

@Composable
private fun NullableTextField(
    value: String,
    onValueChange: (String) -> Unit,
    label: String,
    clear: Boolean,
    onClearChange: (Boolean) -> Unit,
    enabled: Boolean,
) {
    OutlinedTextField(
        value = value,
        onValueChange = onValueChange,
        modifier = Modifier.fillMaxWidth(),
        label = { Text(label) },
        singleLine = true,
        enabled = enabled && !clear,
    )
    ClearCheckbox(
        label = stringResource(R.string.field_clear_suffix, label),
        checked = clear,
        onCheckedChange = onClearChange,
        enabled = enabled,
    )
}

@Composable
private fun MinutesField(
    value: String,
    onValueChange: (String) -> Unit,
    label: String,
    valid: Boolean,
    clear: Boolean,
    onClearChange: (Boolean) -> Unit,
    enabled: Boolean,
) {
    OutlinedTextField(
        value = value,
        onValueChange = onValueChange,
        modifier = Modifier.fillMaxWidth(),
        label = { Text(label) },
        singleLine = true,
        enabled = enabled && !clear,
        isError = !valid,
        supportingText = {
            if (!valid) {
                Text(stringResource(R.string.recipe_field_minutes_invalid))
            }
        },
        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
    )
    ClearCheckbox(
        label = stringResource(R.string.field_clear_suffix, label),
        checked = clear,
        onCheckedChange = onClearChange,
        enabled = enabled,
    )
}

@Composable
private fun ClearCheckbox(
    label: String,
    checked: Boolean,
    onCheckedChange: (Boolean) -> Unit,
    enabled: Boolean,
) {
    Row(
        verticalAlignment = Alignment.CenterVertically,
        modifier =
            Modifier
                .fillMaxWidth()
                .clickable(enabled = enabled) { onCheckedChange(!checked) },
    ) {
        Checkbox(
            checked = checked,
            onCheckedChange = null,
            enabled = enabled,
        )
        Text(
            text = label,
            style = MaterialTheme.typography.bodyMedium,
            color =
                if (enabled) {
                    MaterialTheme.colorScheme.onSurface
                } else {
                    MaterialTheme.colorScheme.onSurfaceVariant
                },
        )
    }
}

@Composable
private fun FormError(title: String, error: MobileAppError?) {
    Column(modifier = Modifier.semantics { liveRegion = LiveRegionMode.Polite }) {
        Text(
            text = title,
            style = MaterialTheme.typography.titleSmall,
            color = MaterialTheme.colorScheme.error,
        )
        if (error != null) {
            Text(
                text = error.message,
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
    }
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

private fun formatEpochSeconds(epochSeconds: Long?): String {
    if (epochSeconds == null) return ""
    val formatter = DateFormat.getDateTimeInstance(DateFormat.SHORT, DateFormat.SHORT)
    return formatter.format(Date(epochSeconds * 1_000L))
}

private val isoDatePattern = Regex("""\d{4}-\d{2}-\d{2}""")

private fun isValidIsoDate(value: String): Boolean {
    if (!isoDatePattern.matches(value)) return false
    val year = value.substring(0, 4).toInt()
    val month = value.substring(5, 7).toInt()
    val day = value.substring(8, 10).toInt()
    if (month !in 1..12) return false
    val maxDay =
        when (month) {
            1, 3, 5, 7, 8, 10, 12 -> 31
            4, 6, 9, 11 -> 30
            else -> if (year % 4 == 0 && (year % 100 != 0 || year % 400 == 0)) 29 else 28
        }
    return day in 1..maxDay
}

/** Maps a nullable text field + explicit clear control to a patch value. */
private fun nullablePatch(
    text: String,
    clear: Boolean,
    original: String?,
): PatchField<String> =
    when {
        clear -> PatchField.Null
        text.isBlank() && original == null -> PatchField.Absent
        text.isBlank() -> PatchField.Null
        text == original -> PatchField.Absent
        else -> PatchField.Value(text.trim())
    }

/** Maps an optional integer field + explicit clear control to a patch value. */
private fun optionalIntPatch(text: String, clear: Boolean): PatchField<Int> =
    when {
        clear -> PatchField.Null
        text.isBlank() -> PatchField.Absent
        else -> PatchField.Value(text.trim().toInt())
    }

// ---------------------------------------------------------------------------
// Previews
// ---------------------------------------------------------------------------

@Preview(showBackground = true)
@Composable
private fun SignInScreenPreview() {
    HorologiaTheme {
        SignInScreen(
            state = MobileAppState(phase = MobileSessionPhase.SIGNED_OUT),
            onConnect = {},
        )
    }
}
