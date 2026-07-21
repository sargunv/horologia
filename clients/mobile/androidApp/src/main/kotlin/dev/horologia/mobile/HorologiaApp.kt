package dev.horologia.mobile

import android.content.pm.PackageManager
import android.os.Build
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.AccountCircle
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Layers
import androidx.compose.material.icons.filled.MenuBook
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.material3.adaptive.ExperimentalMaterial3AdaptiveApi
import androidx.compose.material3.adaptive.currentWindowAdaptiveInfo
import androidx.compose.material3.adaptive.navigationsuite.NavigationSuiteScaffold
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.core.content.ContextCompat
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import androidx.navigation3.runtime.NavKey
import androidx.navigation3.runtime.entryProvider
import androidx.navigation3.runtime.rememberNavBackStack
import androidx.navigation3.ui.NavDisplay
import androidx.window.core.layout.WindowSizeClass
import dev.horologia.mobile.navigation.SemanticDestination
import dev.horologia.mobile.runtime.MobileAppState
import dev.horologia.mobile.runtime.MobileSessionPhase
import dev.horologia.mobile.ui.AccountDestination
import dev.horologia.mobile.ui.BootstrapScreen
import dev.horologia.mobile.ui.RecipeDetailScreen
import dev.horologia.mobile.ui.RecipeEditScreen
import dev.horologia.mobile.ui.RecipesDestination
import dev.horologia.mobile.ui.SearchDestination
import dev.horologia.mobile.ui.SignInScreen
import dev.horologia.mobile.ui.SpaceDetailScreen
import dev.horologia.mobile.ui.SpacesDestination
import dev.horologia.mobile.ui.TaskDetailScreen
import dev.horologia.mobile.ui.TaskEditScreen
import dev.horologia.mobile.ui.TasksDestination
import kotlinx.serialization.Serializable

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
