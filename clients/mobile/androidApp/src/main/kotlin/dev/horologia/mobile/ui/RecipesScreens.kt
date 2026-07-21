package dev.horologia.mobile.ui

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.consumeWindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Edit
import androidx.compose.material3.Button
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LinearWavyProgressIndicator
import androidx.compose.material3.ListItem
import androidx.compose.material3.ListItemDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.derivedStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.selected
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import dev.horologia.mobile.HorologiaViewModel
import dev.horologia.mobile.R
import dev.horologia.mobile.designsystem.ClearCheckbox
import dev.horologia.mobile.designsystem.EmptyPane
import dev.horologia.mobile.designsystem.ErrorPane
import dev.horologia.mobile.designsystem.FormError
import dev.horologia.mobile.designsystem.ListErrorSnackbarEffect
import dev.horologia.mobile.designsystem.LoadMoreRow
import dev.horologia.mobile.designsystem.LoadingPane
import dev.horologia.mobile.designsystem.MetaRow
import dev.horologia.mobile.designsystem.MinutesField
import dev.horologia.mobile.domain.MobileRecipe
import dev.horologia.mobile.domain.MobileRecipeUpdate
import dev.horologia.mobile.domain.MobileRecipeYield
import dev.horologia.mobile.domain.MobileSpace
import dev.horologia.mobile.domain.PatchField
import dev.horologia.mobile.runtime.MobileAppState
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun RecipesDestination(
    state: MobileAppState,
    viewModel: HorologiaViewModel,
    selectedRecipeId: String?,
    onOpenRecipe: (String, String) -> Unit,
    onEditRecipe: (String, String) -> Unit,
) {
    LaunchedEffect(Unit) { viewModel.loadSpaces() }
    LaunchedEffect(state.spaces, state.selectedSpace) {
        if (state.selectedSpace == null && state.spaces.isNotEmpty()) {
            viewModel.selectSpace(state.spaces.first().slug)
        }
    }

    val snackbarHostState = remember { SnackbarHostState() }
    ListErrorSnackbarEffect(
        snackbarHostState = snackbarHostState,
        error = state.error.takeIf { state.spaceRecipes.isNotEmpty() },
        onRetry = { state.selectedSpace?.let { viewModel.selectSpace(it.slug) } },
    )
    Scaffold(
        topBar = { TopAppBar(title = { Text(stringResource(R.string.nav_recipes)) }) },
        snackbarHost = { SnackbarHost(snackbarHostState) },
        floatingActionButton = {
            FloatingActionButton(
                // TODO: wire to create flow
                onClick = {},
            ) {
                Icon(
                    Icons.Filled.Add,
                    contentDescription = stringResource(R.string.action_new_recipe),
                )
            }
        },
    ) { innerPadding ->
        Column(modifier = Modifier.padding(innerPadding).consumeWindowInsets(innerPadding)) {
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
                    SpaceRecipeList(
                        state = state,
                        viewModel = viewModel,
                        selectedRecipeId = selectedRecipeId,
                        showSelection = selectedRecipeId != null,
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
                    contentPadding = PaddingValues(bottom = 16.dp),
                ) {
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
    ) {
        Text(recipe.title, maxLines = 1, overflow = TextOverflow.Ellipsis)
    }
}

@Composable
private fun SpaceChipsRow(
    spaces: List<MobileSpace>,
    selectedSlug: String?,
    onSelect: (String) -> Unit,
) {
    LazyRow(
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        contentPadding = PaddingValues(horizontal = 16.dp, vertical = 4.dp),
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

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun RecipeDetailScreen(
    state: MobileAppState,
    viewModel: HorologiaViewModel,
    spaceSlug: String,
    recipeId: String,
    showBackButton: Boolean,
    onBack: () -> Unit,
    onEdit: (String, String) -> Unit,
) {
    LaunchedEffect(spaceSlug, recipeId) { viewModel.selectRecipe(spaceSlug, recipeId) }
    val recipe = state.selectedRecipe?.takeIf { it.id == recipeId }
    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Text(
                        text = recipe?.title ?: stringResource(R.string.nav_recipes),
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
                actions = {
                    if (recipe != null) {
                        IconButton(onClick = { onEdit(recipe.spaceSlug, recipe.id) }) {
                            Icon(
                                Icons.Filled.Edit,
                                contentDescription = stringResource(R.string.action_edit),
                            )
                        }
                    }
                },
            )
        },
    ) { innerPadding ->
        Box(modifier = Modifier.padding(innerPadding).consumeWindowInsets(innerPadding)) {
            RecipeDetailBody(
                state = state,
                viewModel = viewModel,
                recipeId = recipeId,
                onEdit = onEdit,
                showHeading = false,
                onRetry = { viewModel.selectRecipe(spaceSlug, recipeId) },
            )
        }
    }
}

@Composable
fun RecipeDetailBody(
    state: MobileAppState,
    viewModel: HorologiaViewModel,
    recipeId: String,
    onEdit: (String, String) -> Unit,
    showHeading: Boolean = true,
    onRetry: () -> Unit = {},
) {
    val recipe = state.selectedRecipe?.takeIf { it.id == recipeId }
    when {
        recipe != null ->
            Column(
                modifier =
                    Modifier
                        .fillMaxSize()
                        .verticalScroll(rememberScrollState())
                        .padding(horizontal = 16.dp, vertical = 16.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                if (showHeading) {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Text(
                            text = recipe.title,
                            style = MaterialTheme.typography.headlineSmall,
                            modifier = Modifier.weight(1f).semantics { heading() },
                        )
                        IconButton(onClick = { onEdit(recipe.spaceSlug, recipe.id) }) {
                            Icon(
                                Icons.Filled.Edit,
                                contentDescription = stringResource(R.string.action_edit),
                            )
                        }
                    }
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

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun RecipeEditScreen(
    state: MobileAppState,
    viewModel: HorologiaViewModel,
    spaceSlug: String,
    recipeId: String,
    onDone: () -> Unit,
) {
    val recipe = state.selectedRecipe?.takeIf { it.id == recipeId }
    if (recipe == null) {
        Scaffold(
            topBar = {
                TopAppBar(
                    title = { Text(stringResource(R.string.recipe_edit_title)) },
                    navigationIcon = {
                        IconButton(onClick = onDone) {
                            Icon(
                                Icons.AutoMirrored.Filled.ArrowBack,
                                contentDescription = stringResource(R.string.action_back),
                            )
                        }
                    },
                )
            },
        ) { innerPadding ->
            Box(modifier = Modifier.padding(innerPadding).consumeWindowInsets(innerPadding)) {
                LoadingPane()
            }
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

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(stringResource(R.string.recipe_edit_title)) },
                navigationIcon = {
                    IconButton(onClick = onDone) {
                        Icon(
                            Icons.AutoMirrored.Filled.ArrowBack,
                            contentDescription = stringResource(R.string.action_back),
                        )
                    }
                },
            )
        },
    ) { innerPadding ->
        Column(modifier = Modifier.padding(innerPadding).consumeWindowInsets(innerPadding)) {
            if (saving) {
                LinearWavyProgressIndicator(modifier = Modifier.fillMaxWidth())
            }
            Column(
                modifier =
                    Modifier
                        .fillMaxSize()
                        .verticalScroll(rememberScrollState())
                        .padding(horizontal = 16.dp, vertical = 16.dp),
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
}

/** Maps an optional integer field + explicit clear control to a patch value. */
private fun optionalIntPatch(text: String, clear: Boolean): PatchField<Int> =
    when {
        clear -> PatchField.Null
        text.isBlank() -> PatchField.Absent
        else -> PatchField.Value(text.trim().toInt())
    }
