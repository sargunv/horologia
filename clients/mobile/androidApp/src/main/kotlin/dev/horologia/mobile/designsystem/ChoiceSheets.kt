package dev.horologia.mobile.designsystem

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.Checkbox
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.ListItem
import androidx.compose.material3.ListItemDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.RadioButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import dev.horologia.mobile.R

/** Rows sit transparently on the sheet's `surfaceContainerLow` container. */
private val SheetRowColors
    @Composable get() = ListItemDefaults.colors(containerColor = Color.Transparent)

/**
 * Modal bottom sheet hosting a single editing choice. Selection is the save:
 * sheets report their result through the caller's callback and dismiss.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ChoiceSheet(
    title: String,
    onDismiss: () -> Unit,
    modifier: Modifier = Modifier,
    content: @Composable ColumnScope.() -> Unit,
) {
    ModalBottomSheet(onDismissRequest = onDismiss, modifier = modifier) {
        Column {
            Text(
                text = title,
                style = MaterialTheme.typography.titleLarge,
                modifier =
                    Modifier
                        .padding(horizontal = 24.dp)
                        .padding(bottom = 8.dp)
                        .semantics { heading() },
            )
            content()
        }
    }
}

/**
 * Optional search field above a filtered option list. Searching kicks in once
 * the option count passes [searchable]; rows are rendered by [itemContent].
 */
@Composable
fun <T> SearchableChoiceList(
    options: List<T>,
    optionLabel: (T) -> String,
    modifier: Modifier = Modifier,
    searchable: Boolean = options.size > 8,
    itemContent: @Composable (T) -> Unit,
) {
    var query by rememberSaveable { mutableStateOf("") }
    Column(modifier = modifier) {
        if (searchable) {
            OutlinedTextField(
                value = query,
                onValueChange = { query = it },
                placeholder = { Text(stringResource(R.string.choice_search_hint)) },
                leadingIcon = {
                    Icon(Icons.Filled.Search, contentDescription = null)
                },
                singleLine = true,
                modifier =
                    Modifier
                        .fillMaxWidth()
                        .padding(horizontal = 24.dp)
                        .padding(bottom = 8.dp),
            )
        }
        val filtered =
            if (query.isBlank()) {
                options
            } else {
                options.filter { optionLabel(it).contains(query.trim(), ignoreCase = true) }
            }
        if (filtered.isEmpty()) {
            Text(
                text = stringResource(R.string.search_no_results, query.trim()),
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(horizontal = 24.dp, vertical = 16.dp),
            )
        } else {
            filtered.forEach { itemContent(it) }
        }
    }
}

/**
 * Radio-semantics single choice. Tapping a row reports it via [onSelect] and
 * dismisses the sheet. When [clearLabel] is set and something is selected, a
 * destructive clear row at the top reports null.
 */
@Composable
fun <T> SingleChoiceSheet(
    title: String,
    options: List<T>,
    optionLabel: (T) -> String,
    selected: T?,
    onSelect: (T?) -> Unit,
    onDismiss: () -> Unit,
    modifier: Modifier = Modifier,
    optionIcon: (T) -> ImageVector? = { null },
    clearLabel: String? = null,
) {
    ChoiceSheet(title = title, onDismiss = onDismiss, modifier = modifier) {
        if (clearLabel != null && selected != null) {
            ListItem(
                colors = SheetRowColors,
                modifier =
                    Modifier
                        .fillMaxWidth()
                        .clickable {
                            onSelect(null)
                            onDismiss()
                        },
            ) {
                Text(
                    text = clearLabel,
                    color = MaterialTheme.colorScheme.error,
                )
            }
        }
        SearchableChoiceList(options = options, optionLabel = optionLabel) { option ->
            ListItem(
                colors = SheetRowColors,
                leadingContent =
                    optionIcon(option)?.let {
                        @Composable {
                            Icon(imageVector = it, contentDescription = null)
                        }
                    },
                trailingContent = {
                    RadioButton(selected = option == selected, onClick = null)
                },
                modifier =
                    Modifier
                        .fillMaxWidth()
                        .clickable {
                            onSelect(option)
                            onDismiss()
                        },
            ) {
                Text(
                    text = optionLabel(option),
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
        }
    }
}

/**
 * Checkbox multi choice with a local draft set. The draft commits through
 * [onCommit] on dismiss, only when it differs from [selected].
 */
@Composable
fun <T> MultiChoiceSheet(
    title: String,
    options: List<T>,
    optionLabel: (T) -> String,
    selected: Set<T>,
    onCommit: (Set<T>) -> Unit,
    onDismiss: () -> Unit,
    modifier: Modifier = Modifier,
    optionIcon: (T) -> ImageVector? = { null },
) {
    var draft by remember { mutableStateOf(selected) }
    ChoiceSheet(
        title = title,
        onDismiss = {
            if (draft != selected) onCommit(draft)
            onDismiss()
        },
        modifier = modifier,
    ) {
        SearchableChoiceList(options = options, optionLabel = optionLabel) { option ->
            ListItem(
                colors = SheetRowColors,
                leadingContent =
                    optionIcon(option)?.let {
                        @Composable {
                            Icon(imageVector = it, contentDescription = null)
                        }
                    },
                trailingContent = {
                    Checkbox(
                        checked = option in draft,
                        onCheckedChange = null,
                    )
                },
                modifier =
                    Modifier
                        .fillMaxWidth()
                        .clickable {
                            draft =
                                if (option in draft) {
                                    draft - option
                                } else {
                                    draft + option
                                }
                        },
            ) {
                Text(
                    text = optionLabel(option),
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
        }
    }
}
