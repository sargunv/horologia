package dev.horologia.mobile.ui

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
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.MoreVert
import androidx.compose.material.icons.outlined.Event
import androidx.compose.material.icons.outlined.Layers
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LinearWavyProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.derivedStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.key
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import dev.horologia.mobile.HorologiaViewModel
import dev.horologia.mobile.R
import dev.horologia.mobile.designsystem.AutosaveDescriptionField
import dev.horologia.mobile.designsystem.ChoiceSheet
import dev.horologia.mobile.designsystem.DatePickerSheet
import dev.horologia.mobile.designsystem.EditableHeadline
import dev.horologia.mobile.designsystem.EmptyPane
import dev.horologia.mobile.designsystem.ErrorPane
import dev.horologia.mobile.designsystem.ListErrorSnackbarEffect
import dev.horologia.mobile.designsystem.LoadMoreRow
import dev.horologia.mobile.designsystem.LoadingPane
import dev.horologia.mobile.designsystem.PropertyRow
import dev.horologia.mobile.designsystem.SectionHeader
import dev.horologia.mobile.designsystem.SingleChoiceSheet
import dev.horologia.mobile.designsystem.TagsField
import dev.horologia.mobile.designsystem.TaskListItem
import dev.horologia.mobile.designsystem.UpdatedFooter
import dev.horologia.mobile.designsystem.taskIcon
import dev.horologia.mobile.domain.MobileTask
import dev.horologia.mobile.domain.MobileTaskDue
import dev.horologia.mobile.domain.MobileTaskUpdate
import dev.horologia.mobile.domain.MobileTaskVisualMetadata
import dev.horologia.mobile.domain.PatchField
import dev.horologia.mobile.runtime.MobileAppState
import kotlinx.coroutines.launch
import kotlinx.datetime.TimeZone

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TasksDestination(
    state: MobileAppState,
    viewModel: HorologiaViewModel,
    selectedTaskId: String?,
    onOpenTask: (String, String) -> Unit,
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
        floatingActionButton = {
            FloatingActionButton(
                // TODO: wire to create flow
                onClick = {},
            ) {
                Icon(
                    Icons.Filled.Add,
                    contentDescription = stringResource(R.string.action_new_task),
                )
            }
        },
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
                    contentPadding = PaddingValues(bottom = 16.dp),
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
// Task detail (inline editing: selection is the save)
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
) {
    LaunchedEffect(spaceSlug, taskId) { viewModel.selectTask(spaceSlug, taskId) }
    val task = state.selectedTask?.takeIf { it.id == taskId }
    val scope = rememberCoroutineScope()
    val snackbarHostState = remember { SnackbarHostState() }
    var menuOpen by remember { mutableStateOf(false) }
    var confirmDelete by rememberSaveable { mutableStateOf(false) }
    ListErrorSnackbarEffect(
        snackbarHostState = snackbarHostState,
        error = state.error.takeIf { task != null },
        onRetry = { viewModel.selectTask(spaceSlug, taskId) },
    )
    Scaffold(
        topBar = {
            TopAppBar(
                // No title: the in-body EditableHeadline is the single,
                // editable title (the pane-embedded body has no app bar).
                title = {},
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
                        IconButton(onClick = { menuOpen = true }) {
                            Icon(
                                Icons.Filled.MoreVert,
                                contentDescription = stringResource(R.string.action_more),
                            )
                        }
                        DropdownMenu(
                            expanded = menuOpen,
                            onDismissRequest = { menuOpen = false },
                        ) {
                            DropdownMenuItem(
                                text = { Text(stringResource(R.string.action_delete)) },
                                onClick = {
                                    menuOpen = false
                                    confirmDelete = true
                                },
                            )
                        }
                    }
                },
            )
        },
        snackbarHost = { SnackbarHost(snackbarHostState) },
    ) { innerPadding ->
        Column(modifier = Modifier.padding(innerPadding).consumeWindowInsets(innerPadding)) {
            if (state.loading.taskUpdate) {
                LinearWavyProgressIndicator(modifier = Modifier.fillMaxWidth())
            }
            Box(modifier = Modifier.fillMaxSize()) {
                TaskDetailBody(
                    state = state,
                    viewModel = viewModel,
                    spaceSlug = spaceSlug,
                    taskId = taskId,
                    showHeading = true,
                )
            }
        }
    }

    if (confirmDelete) {
        AlertDialog(
            onDismissRequest = { confirmDelete = false },
            title = { Text(stringResource(R.string.task_delete_confirm_title)) },
            text = { Text(stringResource(R.string.task_delete_confirm_body)) },
            confirmButton = {
                TextButton(
                    onClick = {
                        confirmDelete = false
                        scope.launch {
                            if (viewModel.deleteTask(spaceSlug, taskId)) onBack()
                        }
                    },
                    colors =
                        ButtonDefaults.textButtonColors(
                            contentColor = MaterialTheme.colorScheme.error,
                        ),
                ) {
                    Text(stringResource(R.string.action_confirm_delete))
                }
            },
            dismissButton = {
                TextButton(onClick = { confirmDelete = false }) {
                    Text(stringResource(R.string.action_cancel))
                }
            },
        )
    }
}

@Composable
fun TaskDetailBody(
    state: MobileAppState,
    viewModel: HorologiaViewModel,
    spaceSlug: String?,
    taskId: String,
    showHeading: Boolean = true,
) {
    val task = state.selectedTask?.takeIf { it.id == taskId }
    when {
        task != null ->
            EditableTaskDetail(
                state = state,
                viewModel = viewModel,
                task = task,
                showHeading = showHeading,
            )

        state.error != null && !state.loading.task ->
            ErrorPane(
                title = stringResource(R.string.task_detail_error_title),
                error = state.error,
                onRetry = { viewModel.selectTask(spaceSlug ?: task?.spaceSlug ?: "", taskId) },
            )

        else -> LoadingPane()
    }
}

@Composable
private fun EditableTaskDetail(
    state: MobileAppState,
    viewModel: HorologiaViewModel,
    task: MobileTask,
    showHeading: Boolean,
) {
    val scope = rememberCoroutineScope()
    var sheet by rememberSaveable { mutableStateOf<TaskDetailSheet?>(null) }
    val metadata = state.taskVisualMetadataBySpace[task.spaceSlug] ?: MobileTaskVisualMetadata()
    val status = metadata.statuses.firstOrNull { it.label == task.status }
    val priority = metadata.priorityLevels.firstOrNull { it.label == task.priority }
    val effort = metadata.effortLevels.firstOrNull { it.label == task.effort }

    fun patch(update: MobileTaskUpdate) {
        scope.launch { viewModel.updateTask(task.spaceSlug, task.id, update) }
    }

    Column(
        modifier =
            Modifier
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .padding(vertical = 16.dp),
        verticalArrangement = Arrangement.spacedBy(4.dp),
    ) {
        if (showHeading) {
            key(task.id) {
                EditableHeadline(
                    title = task.title,
                    onCommit = { patch(MobileTaskUpdate(title = it)) },
                    modifier = Modifier.padding(horizontal = 16.dp),
                )
            }
        }
        key(task.id) {
            AutosaveDescriptionField(
                description = task.description,
                onSave = { value ->
                    viewModel.updateTask(task.spaceSlug, task.id, MobileTaskUpdate(description = value))
                },
                modifier = Modifier.padding(horizontal = 16.dp),
            )
        }
        PropertyRow(
            label = stringResource(R.string.detail_status_label),
            value = task.status,
            icon = status?.let { taskIcon(it.iconToken) },
            onClick = { sheet = TaskDetailSheet.STATUS },
        )
        PropertyRow(
            label = stringResource(R.string.detail_priority_label),
            value = task.priority,
            icon = priority?.let { taskIcon(it.iconToken) },
            onClick = { sheet = TaskDetailSheet.PRIORITY },
        )
        PropertyRow(
            label = stringResource(R.string.detail_effort_label),
            value = task.effort,
            icon = effort?.let { taskIcon(it.iconToken) },
            onClick = { sheet = TaskDetailSheet.EFFORT },
        )
        PropertyRow(
            label = stringResource(R.string.detail_due_label),
            value = task.dueText?.take(10),
            icon = Icons.Outlined.Event,
            onClick = { sheet = TaskDetailSheet.DUE },
        )
        SectionHeader(
            text = stringResource(R.string.detail_tags_label),
            modifier = Modifier.padding(horizontal = 16.dp),
        )
        TagsField(
            tags = task.tags,
            onTagsChange = { patch(MobileTaskUpdate(tags = it)) },
            modifier =
                Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 16.dp),
        )
        PropertyRow(
            label = stringResource(R.string.detail_space_label),
            value = task.spaceSlug,
            icon = Icons.Outlined.Layers,
            onClick = null,
        )
        PropertyRow(
            label = stringResource(R.string.detail_recurrence_label),
            value = null,
            onClick = { sheet = TaskDetailSheet.RECURRENCE },
        )
        PropertyRow(
            label = stringResource(R.string.detail_overdue_action_label),
            value = null,
            onClick = { sheet = TaskDetailSheet.OVERDUE_ACTION },
        )
        PropertyRow(
            label = stringResource(R.string.detail_assignees_label),
            value = null,
            onClick = { sheet = TaskDetailSheet.ASSIGNEES },
        )
        PropertyRow(
            label = stringResource(R.string.detail_rotation_pool_label),
            value = null,
            onClick = { sheet = TaskDetailSheet.ROTATION_POOL },
        )
        PropertyRow(
            label = stringResource(R.string.detail_relations_label),
            value = null,
            onClick = { sheet = TaskDetailSheet.RELATIONS },
        )
    }

    when (sheet) {
        TaskDetailSheet.STATUS ->
            SingleChoiceSheet(
                title = stringResource(R.string.detail_status_label),
                options = metadata.statuses,
                optionLabel = { it.label },
                optionIcon = { taskIcon(it.iconToken) },
                selected = status,
                onSelect = { option ->
                    option?.let { patch(MobileTaskUpdate(status = it.label)) }
                },
                onDismiss = { sheet = null },
            )

        TaskDetailSheet.PRIORITY ->
            SingleChoiceSheet(
                title = stringResource(R.string.detail_priority_label),
                options = metadata.priorityLevels,
                optionLabel = { it.label },
                optionIcon = { taskIcon(it.iconToken) },
                selected = priority,
                clearLabel = stringResource(R.string.choice_none),
                onSelect = { option ->
                    patch(
                        MobileTaskUpdate(
                            priority =
                                if (option == null) PatchField.Null else PatchField.Value(option.label),
                        ),
                    )
                },
                onDismiss = { sheet = null },
            )

        TaskDetailSheet.EFFORT ->
            SingleChoiceSheet(
                title = stringResource(R.string.detail_effort_label),
                options = metadata.effortLevels,
                optionLabel = { it.label },
                optionIcon = { taskIcon(it.iconToken) },
                selected = effort,
                clearLabel = stringResource(R.string.choice_none),
                onSelect = { option ->
                    patch(
                        MobileTaskUpdate(
                            effort =
                                if (option == null) PatchField.Null else PatchField.Value(option.label),
                        ),
                    )
                },
                onDismiss = { sheet = null },
            )

        TaskDetailSheet.DUE ->
            DatePickerSheet(
                title = stringResource(R.string.task_field_due_date),
                current = task.dueText?.take(10),
                onSelect = { date ->
                    patch(
                        MobileTaskUpdate(
                            due =
                                if (date == null) {
                                    PatchField.Null
                                } else {
                                    PatchField.Value(
                                        MobileTaskDue(date, TimeZone.currentSystemDefault().id),
                                    )
                                },
                        ),
                    )
                },
                onDismiss = { sheet = null },
            )

        TaskDetailSheet.RECURRENCE ->
            UnsupportedSheet(
                label = stringResource(R.string.detail_recurrence_label),
                onDismiss = { sheet = null },
            )

        TaskDetailSheet.OVERDUE_ACTION ->
            UnsupportedSheet(
                label = stringResource(R.string.detail_overdue_action_label),
                onDismiss = { sheet = null },
            )

        TaskDetailSheet.ASSIGNEES ->
            UnsupportedSheet(
                label = stringResource(R.string.detail_assignees_label),
                onDismiss = { sheet = null },
            )

        TaskDetailSheet.ROTATION_POOL ->
            UnsupportedSheet(
                label = stringResource(R.string.detail_rotation_pool_label),
                onDismiss = { sheet = null },
            )

        TaskDetailSheet.RELATIONS ->
            UnsupportedSheet(
                label = stringResource(R.string.detail_relations_label),
                onDismiss = { sheet = null },
            )

        null -> Unit
    }
}

private enum class TaskDetailSheet {
    STATUS,
    PRIORITY,
    EFFORT,
    DUE,
    RECURRENCE,
    OVERDUE_ACTION,
    ASSIGNEES,
    ROTATION_POOL,
    RELATIONS,
}

/** Placeholder sheet for task fields not yet editable on mobile. */
@Composable
private fun UnsupportedSheet(label: String, onDismiss: () -> Unit) {
    ChoiceSheet(title = label, onDismiss = onDismiss) {
        Text(
            text = stringResource(R.string.detail_unsupported_mobile),
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier =
                Modifier
                    .padding(horizontal = 24.dp)
                    .padding(bottom = 24.dp),
        )
    }
}
