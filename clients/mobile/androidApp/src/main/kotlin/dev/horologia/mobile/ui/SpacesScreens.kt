package dev.horologia.mobile.ui

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.consumeWindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.ListItem
import androidx.compose.material3.ListItemDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.material3.adaptive.ExperimentalMaterial3AdaptiveApi
import androidx.compose.material3.adaptive.layout.AnimatedPane
import androidx.compose.material3.adaptive.layout.ListDetailPaneScaffoldRole
import androidx.compose.material3.adaptive.layout.PaneAdaptedValue
import androidx.compose.material3.adaptive.navigation.NavigableListDetailPaneScaffold
import androidx.compose.material3.adaptive.navigation.rememberListDetailPaneScaffoldNavigator
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.selected
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import dev.horologia.mobile.HorologiaViewModel
import dev.horologia.mobile.R
import dev.horologia.mobile.designsystem.EmptyPane
import dev.horologia.mobile.designsystem.ErrorBlock
import dev.horologia.mobile.designsystem.ErrorPane
import dev.horologia.mobile.designsystem.ListErrorSnackbarEffect
import dev.horologia.mobile.designsystem.LoadMoreRow
import dev.horologia.mobile.designsystem.LoadingPane
import dev.horologia.mobile.designsystem.LoadingRow
import dev.horologia.mobile.designsystem.PanePlaceholder
import dev.horologia.mobile.designsystem.SectionHeader
import dev.horologia.mobile.designsystem.TaskListItem
import dev.horologia.mobile.designsystem.UpdatedFooter
import dev.horologia.mobile.domain.MobileSpace
import dev.horologia.mobile.runtime.MobileAppState
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SpacesDestination(
    state: MobileAppState,
    viewModel: HorologiaViewModel,
    isExpandedWidth: Boolean,
    onOpenSpace: (String) -> Unit,
    onOpenTask: (String, String) -> Unit,
    onOpenRecipe: (String, String) -> Unit,
) {
    LaunchedEffect(Unit) { viewModel.loadSpaces() }
    val snackbarHostState = remember { SnackbarHostState() }
    ListErrorSnackbarEffect(
        snackbarHostState = snackbarHostState,
        error = state.error.takeIf { state.spaces.isNotEmpty() },
        onRetry = { viewModel.loadSpaces() },
    )
    Scaffold(
        topBar = { TopAppBar(title = { Text(stringResource(R.string.nav_spaces)) }) },
        snackbarHost = { SnackbarHost(snackbarHostState) },
    ) { innerPadding ->
        Box(modifier = Modifier.padding(innerPadding).consumeWindowInsets(innerPadding)) {
            if (isExpandedWidth) {
                SpacesListDetail(
                    state = state,
                    viewModel = viewModel,
                    onOpenTask = onOpenTask,
                    onOpenRecipe = onOpenRecipe,
                )
            } else {
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
                    items(state.spaces, key = { it.slug }) { space ->
                        SpaceRow(
                            space = space,
                            selected = showSelection && space.slug == selectedSlug,
                            onClick = { onSpaceClick(space) },
                        )
                    }
                    item {
                        UpdatedFooter(state.spacesGeneratedAtEpochSeconds)
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

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SpaceDetailScreen(
    state: MobileAppState,
    viewModel: HorologiaViewModel,
    spaceSlug: String,
    showBackButton: Boolean,
    onBack: () -> Unit,
    onOpenTask: (String, String) -> Unit,
    onOpenRecipe: (String, String) -> Unit,
) {
    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Text(
                        text = state.selectedSpace?.takeIf { it.slug == spaceSlug }?.name ?: spaceSlug,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                },
                navigationIcon = {
                    if (showBackButton) {
                        IconButton(onClick = onBack) {
                            Icon(
                                Icons.AutoMirrored.Filled.ArrowBack,
                                contentDescription = stringResource(R.string.action_back),
                            )
                        }
                    }
                },
            )
        },
    ) { innerPadding ->
        Box(modifier = Modifier.padding(innerPadding).consumeWindowInsets(innerPadding)) {
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
}

@Composable
fun SpaceDetailBody(
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
                    TaskListItem(
                        item = state.taskListItem(task),
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
