package dev.horologia.mobile.ui

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
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Edit
import androidx.compose.material3.Button
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LinearWavyProgressIndicator
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
import dev.horologia.mobile.designsystem.InfoBadge
import dev.horologia.mobile.designsystem.ListErrorSnackbarEffect
import dev.horologia.mobile.designsystem.LoadMoreRow
import dev.horologia.mobile.designsystem.LoadingPane
import dev.horologia.mobile.designsystem.MetaRow
import dev.horologia.mobile.designsystem.NullableTextField
import dev.horologia.mobile.designsystem.TaskListItem
import dev.horologia.mobile.designsystem.UpdatedFooter
import dev.horologia.mobile.domain.MobileTask
import dev.horologia.mobile.domain.MobileTaskDue
import dev.horologia.mobile.domain.MobileTaskUpdate
import dev.horologia.mobile.domain.PatchField
import dev.horologia.mobile.runtime.MobileAppState
import java.util.TimeZone
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TasksDestination(
    state: MobileAppState,
    viewModel: HorologiaViewModel,
    selectedTaskId: String?,
    onOpenTask: (String, String) -> Unit,
    onEditTask: (String, String) -> Unit,
) {
    LaunchedEffect(Unit) { viewModel.refreshMyTasks() }
    val snackbarHostState = remember { SnackbarHostState() }
    ListErrorSnackbarEffect(
        snackbarHostState = snackbarHostState,
        error = state.error.takeIf { state.myTasks.isNotEmpty() },
        onRetry = { viewModel.refreshMyTasks() },
    )
    Scaffold(
        topBar = { TopAppBar(title = { Text(stringResource(R.string.nav_tasks)) }) },
        snackbarHost = { SnackbarHost(snackbarHostState) },
    ) { innerPadding ->
        Box(modifier = Modifier.padding(innerPadding).consumeWindowInsets(innerPadding)) {
            MyTaskList(
                state = state,
                viewModel = viewModel,
                selectedTaskId = selectedTaskId,
                showSelection = selectedTaskId != null,
                onTaskClick = { task ->
                    viewModel.selectTask(task.spaceSlug, task.id)
                    onOpenTask(task.spaceSlug, task.id)
                },
            )
        }
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
                    items(state.myTasks, key = { it.id }) { task ->
                        TaskListItem(
                            item = state.taskListItem(task),
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
                    item {
                        UpdatedFooter(state.myTasksGeneratedAtEpochSeconds)
                    }
                }
        }
    }
}

// ---------------------------------------------------------------------------
// Task detail + edit
// ---------------------------------------------------------------------------

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TaskDetailScreen(
    state: MobileAppState,
    viewModel: HorologiaViewModel,
    spaceSlug: String,
    taskId: String,
    showBackButton: Boolean,
    onBack: () -> Unit,
    onEdit: (String, String) -> Unit,
) {
    LaunchedEffect(spaceSlug, taskId) { viewModel.selectTask(spaceSlug, taskId) }
    val task = state.selectedTask?.takeIf { it.id == taskId }
    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Text(
                        text = task?.title ?: stringResource(R.string.nav_tasks),
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
                    if (task != null) {
                        IconButton(onClick = { onEdit(task.spaceSlug, task.id) }) {
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
            TaskDetailBody(
                state = state,
                viewModel = viewModel,
                spaceSlug = spaceSlug,
                taskId = taskId,
                onEdit = onEdit,
                showHeading = false,
            )
        }
    }
}

@Composable
fun TaskDetailBody(
    state: MobileAppState,
    viewModel: HorologiaViewModel,
    spaceSlug: String?,
    taskId: String,
    onEdit: (String, String) -> Unit,
    showHeading: Boolean = true,
) {
    val task = state.selectedTask?.takeIf { it.id == taskId }
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
                if (showHeading) {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Text(
                            text = task.title,
                            style = MaterialTheme.typography.headlineSmall,
                            modifier = Modifier.weight(1f).semantics { heading() },
                        )
                        IconButton(onClick = { onEdit(task.spaceSlug, task.id) }) {
                            Icon(
                                Icons.Filled.Edit,
                                contentDescription = stringResource(R.string.action_edit),
                            )
                        }
                    }
                }
                InfoBadge(
                    text = task.status,
                    containerColor = MaterialTheme.colorScheme.primaryContainer,
                    contentColor = MaterialTheme.colorScheme.onPrimaryContainer,
                )
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

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TaskEditScreen(
    state: MobileAppState,
    viewModel: HorologiaViewModel,
    spaceSlug: String,
    taskId: String,
    onDone: () -> Unit,
) {
    val task = state.selectedTask?.takeIf { it.id == taskId }
    if (task == null) {
        Scaffold(
            topBar = {
                TopAppBar(
                    title = { Text(stringResource(R.string.task_edit_title)) },
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

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(stringResource(R.string.task_edit_title)) },
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
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

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
