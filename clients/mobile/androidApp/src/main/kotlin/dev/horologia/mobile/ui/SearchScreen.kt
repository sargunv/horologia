package dev.horologia.mobile.ui

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.consumeWindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.text.input.TextFieldState
import androidx.compose.foundation.text.input.clearText
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Clear
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.ListItem
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SearchBar
import androidx.compose.material3.SearchBarDefaults
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.rememberSearchBarState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.snapshotFlow
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import dev.horologia.mobile.HorologiaViewModel
import dev.horologia.mobile.R
import dev.horologia.mobile.designsystem.EmptyPane
import dev.horologia.mobile.designsystem.ErrorPane
import dev.horologia.mobile.designsystem.ListErrorSnackbarEffect
import dev.horologia.mobile.designsystem.LoadingPane
import dev.horologia.mobile.designsystem.SectionHeader
import dev.horologia.mobile.designsystem.UpdatedFooter
import dev.horologia.mobile.domain.MobileSearchResult
import dev.horologia.mobile.runtime.MobileAppState
import kotlinx.coroutines.FlowPreview
import kotlinx.coroutines.flow.collectLatest
import kotlinx.coroutines.flow.debounce

@OptIn(ExperimentalMaterial3Api::class, FlowPreview::class)
@Composable
fun SearchDestination(
    state: MobileAppState,
    viewModel: HorologiaViewModel,
    initialQuery: String?,
    onOpenTask: (String, String) -> Unit,
    onOpenRecipe: (String, String) -> Unit,
) {
    val textFieldState =
        rememberSaveable(initialQuery, saver = TextFieldState.Saver) {
            TextFieldState(initialQuery ?: state.searchQuery)
        }
    val searchBarState = rememberSearchBarState()

    LaunchedEffect(Unit) {
        snapshotFlow { textFieldState.text }
            .debounce(300)
            .collectLatest { viewModel.submitSearch(it.toString()) }
    }

    val snackbarHostState = remember { SnackbarHostState() }
    ListErrorSnackbarEffect(
        snackbarHostState = snackbarHostState,
        error = state.error.takeIf { state.searchResults.isNotEmpty() },
        onRetry = { viewModel.submitSearch(textFieldState.text.toString()) },
    )
    Scaffold(
        topBar = { TopAppBar(title = { Text(stringResource(R.string.nav_search)) }) },
        snackbarHost = { SnackbarHost(snackbarHostState) },
    ) { innerPadding ->
        Column(modifier = Modifier.padding(innerPadding).consumeWindowInsets(innerPadding)) {
            SearchBar(
                state = searchBarState,
                modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp),
                inputField = {
                    SearchBarDefaults.InputField(
                        textFieldState = textFieldState,
                        searchBarState = searchBarState,
                        onSearch = { viewModel.submitSearch(it) },
                        placeholder = { Text(stringResource(R.string.search_hint)) },
                        leadingIcon = {
                            Icon(Icons.Filled.Search, contentDescription = null)
                        },
                        trailingIcon = {
                            if (textFieldState.text.isNotEmpty()) {
                                IconButton(onClick = { textFieldState.clearText() }) {
                                    Icon(
                                        Icons.Filled.Clear,
                                        contentDescription = stringResource(R.string.action_clear_text),
                                    )
                                }
                            }
                        },
                    )
                },
            )
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
                onRetry = { viewModel.submitSearch(textFieldState.text.toString()) },
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
                contentPadding = PaddingValues(bottom = 16.dp),
            ) {
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
                            modifier = Modifier.padding(horizontal = 16.dp),
                        )
                    }
                    items(results, key = { "$kind:${it.id}" }) { result ->
                        ListItem(
                            overlineContent = {
                                Text(result.spaceSlug, maxLines = 1, overflow = TextOverflow.Ellipsis)
                            },
                            supportingContent = {
                                if (result.detail.isNotBlank()) {
                                    Text(result.detail, maxLines = 1, overflow = TextOverflow.Ellipsis)
                                }
                            },
                            modifier = Modifier.fillMaxWidth().clickable { onOpenResult(result) },
                        ) {
                            Text(result.title, maxLines = 1, overflow = TextOverflow.Ellipsis)
                        }
                    }
                }
                item {
                    UpdatedFooter(state.searchGeneratedAtEpochSeconds)
                }
            }
    }
}
