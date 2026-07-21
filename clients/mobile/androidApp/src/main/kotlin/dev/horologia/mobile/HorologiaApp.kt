package dev.horologia.mobile

import android.content.pm.PackageManager
import android.os.Build
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInHorizontally
import androidx.compose.animation.slideOutHorizontally
import androidx.compose.animation.togetherWith
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.MenuBook
import androidx.compose.material.icons.filled.AccountCircle
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Layers
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.ExperimentalMaterial3ExpressiveApi
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.adaptive.ExperimentalMaterial3AdaptiveApi
import androidx.compose.material3.adaptive.navigation.BackNavigationBehavior
import androidx.compose.material3.adaptive.navigation3.ListDetailSceneStrategy
import androidx.compose.material3.adaptive.navigation3.rememberListDetailSceneStrategy
import androidx.compose.material3.adaptive.navigationsuite.NavigationSuiteItem
import androidx.compose.material3.adaptive.navigationsuite.NavigationSuiteScaffold
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.IntOffset
import androidx.core.content.ContextCompat
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import androidx.navigation3.runtime.NavKey
import androidx.navigation3.runtime.entryProvider
import androidx.navigation3.runtime.rememberNavBackStack
import androidx.navigation3.ui.NavDisplay
import dev.horologia.mobile.designsystem.PanePlaceholder
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
        TopLevelDestination(HorologiaRoute.Recipes, R.string.nav_recipes, Icons.AutoMirrored.Filled.MenuBook),
        TopLevelDestination(HorologiaRoute.Spaces, R.string.nav_spaces, Icons.Filled.Layers),
        TopLevelDestination(HorologiaRoute.Search(), R.string.nav_search, Icons.Filled.Search),
        TopLevelDestination(HorologiaRoute.Account, R.string.nav_account, Icons.Filled.AccountCircle),
    )

/**
 * Fade-through for switching between top-level destinations; drill-in
 * entries carry no metadata and inherit the NavDisplay slide specs. Timing
 * comes from the Material motion scheme (see [SignedInShell]).
 */
@Composable
private fun topLevelFadeTransitions(): Map<String, Any> {
    val effectsSpec = MaterialTheme.motionScheme.defaultEffectsSpec<Float>()
    return NavDisplay.transitionSpec { fadeIn(effectsSpec) togetherWith fadeOut(effectsSpec) } +
        NavDisplay.popTransitionSpec { fadeIn(effectsSpec) togetherWith fadeOut(effectsSpec) } +
        NavDisplay.predictivePopTransitionSpec {
            fadeIn(effectsSpec) togetherWith fadeOut(effectsSpec)
        }
}

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

@OptIn(ExperimentalMaterial3AdaptiveApi::class, ExperimentalMaterial3ExpressiveApi::class)
@Composable
private fun SignedInShell(state: MobileAppState, viewModel: HorologiaViewModel) {
    val backStack = rememberNavBackStack(HorologiaRoute.Tasks)
    val deepLinkDestination by viewModel.deepLinkDestination.collectAsStateWithLifecycle()
    val motionScheme = MaterialTheme.motionScheme
    val spatialSpec = motionScheme.defaultSpatialSpec<IntOffset>()
    val effectsSpec = motionScheme.defaultEffectsSpec<Float>()
    val topLevelFadeTransitions = topLevelFadeTransitions()

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

    NotificationPermissionRequest()

    NavigationSuiteScaffold(
        navigationItems = {
            topLevelDestinations.forEach { destination ->
                NavigationSuiteItem(
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
            sceneStrategies =
                listOf(
                    // PopUntilCurrentDestinationChange: on wide screens the pane
                    // count doesn't change when popping a detail (the detail pane
                    // stays as a placeholder), so the default
                    // PopUntilScaffoldValueChange would leave system back
                    // unhandled and finish the activity.
                    rememberListDetailSceneStrategy(
                        backNavigationBehavior = BackNavigationBehavior.PopUntilCurrentDestinationChange,
                    ),
                ),
            // Drill-in: shared-axis-X style horizontal slide, timed with the
            // Material motion scheme (spatial spec for position, effects for
            // alpha) — the same specs Material components animate with.
            transitionSpec = {
                slideInHorizontally(spatialSpec, initialOffsetX = { it }) +
                    fadeIn(effectsSpec) togetherWith
                    slideOutHorizontally(spatialSpec, targetOffsetX = { -it / 4 }) +
                    fadeOut(effectsSpec)
            },
            popTransitionSpec = {
                slideInHorizontally(spatialSpec, initialOffsetX = { -it / 4 }) +
                    fadeIn(effectsSpec) togetherWith
                    slideOutHorizontally(spatialSpec, targetOffsetX = { it }) +
                    fadeOut(effectsSpec)
            },
            predictivePopTransitionSpec = {
                slideInHorizontally(spatialSpec, initialOffsetX = { -it / 4 }) +
                    fadeIn(effectsSpec) togetherWith
                    slideOutHorizontally(spatialSpec, targetOffsetX = { it }) +
                    fadeOut(effectsSpec)
            },
            entryProvider =
                entryProvider {
                    entry<HorologiaRoute.Tasks>(
                        metadata =
                            topLevelFadeTransitions +
                                ListDetailSceneStrategy.listPane(
                                    sceneKey = "tasks",
                                    detailPlaceholder = {
                                        PanePlaceholder(text = stringResource(R.string.task_select_prompt))
                                    },
                                ),
                    ) {
                        TasksDestination(
                            state = state,
                            viewModel = viewModel,
                            selectedTaskId =
                                backStack
                                    .filterIsInstance<HorologiaRoute.TaskDetail>()
                                    .lastOrNull()
                                    ?.taskId,
                            onOpenTask = { spaceSlug, taskId ->
                                // Switch the visible detail in place instead of
                                // stacking detail entries.
                                if (backStack.lastOrNull() is HorologiaRoute.TaskDetail) {
                                    backStack.removeLastOrNull()
                                }
                                backStack.add(HorologiaRoute.TaskDetail(spaceSlug, taskId))
                            },
                            onEditTask = { spaceSlug, taskId ->
                                backStack.add(HorologiaRoute.TaskEdit(spaceSlug, taskId))
                            },
                        )
                    }
                    entry<HorologiaRoute.TaskDetail>(
                        metadata = ListDetailSceneStrategy.detailPane(sceneKey = "tasks"),
                    ) { key ->
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
                    entry<HorologiaRoute.Recipes>(
                        metadata =
                            topLevelFadeTransitions +
                                ListDetailSceneStrategy.listPane(
                                    sceneKey = "recipes",
                                    detailPlaceholder = {
                                        PanePlaceholder(text = stringResource(R.string.recipe_select_prompt))
                                    },
                                ),
                    ) {
                        RecipesDestination(
                            state = state,
                            viewModel = viewModel,
                            selectedRecipeId =
                                backStack
                                    .filterIsInstance<HorologiaRoute.RecipeDetail>()
                                    .lastOrNull()
                                    ?.recipeId,
                            onOpenRecipe = { spaceSlug, recipeId ->
                                if (backStack.lastOrNull() is HorologiaRoute.RecipeDetail) {
                                    backStack.removeLastOrNull()
                                }
                                backStack.add(HorologiaRoute.RecipeDetail(spaceSlug, recipeId))
                            },
                            onEditRecipe = { spaceSlug, recipeId ->
                                backStack.add(HorologiaRoute.RecipeEdit(spaceSlug, recipeId))
                            },
                        )
                    }
                    entry<HorologiaRoute.RecipeDetail>(
                        metadata = ListDetailSceneStrategy.detailPane(sceneKey = "recipes"),
                    ) { key ->
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
                    entry<HorologiaRoute.Spaces>(
                        metadata =
                            topLevelFadeTransitions +
                                ListDetailSceneStrategy.listPane(
                                    sceneKey = "spaces",
                                    detailPlaceholder = {
                                        PanePlaceholder(text = stringResource(R.string.space_select_prompt))
                                    },
                                ),
                    ) {
                        SpacesDestination(
                            state = state,
                            viewModel = viewModel,
                            selectedSlug =
                                backStack
                                    .filterIsInstance<HorologiaRoute.SpaceDetail>()
                                    .lastOrNull()
                                    ?.spaceSlug,
                            onOpenSpace = { spaceSlug ->
                                if (backStack.lastOrNull() is HorologiaRoute.SpaceDetail) {
                                    backStack.removeLastOrNull()
                                }
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
                    entry<HorologiaRoute.SpaceDetail>(
                        metadata = ListDetailSceneStrategy.detailPane(sceneKey = "spaces"),
                    ) { key ->
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
                    entry<HorologiaRoute.Search>(metadata = topLevelFadeTransitions) { key ->
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
                    entry<HorologiaRoute.Account>(metadata = topLevelFadeTransitions) {
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
