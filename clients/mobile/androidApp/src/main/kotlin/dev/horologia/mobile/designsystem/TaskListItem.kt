package dev.horologia.mobile.designsystem

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.size
import androidx.compose.material3.Icon
import androidx.compose.material3.ListItem
import androidx.compose.material3.ListItemDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.selected
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import dev.horologia.mobile.domain.TaskListIndicator
import dev.horologia.mobile.domain.TaskListItemModel
import dev.horologia.mobile.domain.TaskStatusCategory

/**
 * Status icon tint by category, mapped onto Material semantic roles:
 * in-progress tasks use the primary brand accent, completed the secondary
 * (green) brand accent, and not-yet-started/unknown stay subdued.
 */
@Composable
private fun statusTint(category: TaskStatusCategory): Color =
    when (category) {
        TaskStatusCategory.INTERMEDIATE -> MaterialTheme.colorScheme.primary
        TaskStatusCategory.COMPLETION -> MaterialTheme.colorScheme.secondary
        TaskStatusCategory.INITIAL,
        TaskStatusCategory.NEUTRAL -> MaterialTheme.colorScheme.onSurfaceVariant
    }

/**
 * Ordered row of informational trailing indicators (priority, then effort).
 * Icons are purely informational: they are not individually focusable or
 * actionable — the parent row carries one merged accessibility label.
 */
@Composable
fun TaskIndicatorRow(
    indicators: List<TaskListIndicator>,
    modifier: Modifier = Modifier,
) {
    if (indicators.isEmpty()) return
    Row(
        modifier = modifier,
        horizontalArrangement = Arrangement.spacedBy(4.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        indicators.forEach { indicator ->
            Icon(
                imageVector = taskIcon(indicator.iconToken),
                contentDescription = null,
                tint = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.size(18.dp),
            )
        }
    }
}

/**
 * Material 3 one/two-line task row.
 *
 * - Leading: status icon, tinted by status category.
 * - Headline: task title (single line, ellipsized).
 * - Supporting: optional due text (present → two-line row, absent → one-line).
 * - Trailing: ordered priority/effort indicator icons.
 *
 * The whole row is a single click target exposing exactly one merged
 * TalkBack description ([TaskListItemModel.accessibilityLabel]) plus the
 * list-detail selection state.
 */
@Composable
fun TaskListItem(
    item: TaskListItemModel,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    selected: Boolean = false,
) {
    ListItem(
        supportingContent =
            item.dueText?.let { dueText ->
                @Composable {
                    Text(dueText, maxLines = 1, overflow = TextOverflow.Ellipsis)
                }
            },
        leadingContent = {
            Icon(
                imageVector = taskIcon(item.statusIconToken),
                contentDescription = null,
                tint = statusTint(item.statusCategory),
            )
        },
        trailingContent =
            if (item.trailingIndicators.isEmpty()) {
                null
            } else {
                @Composable {
                    TaskIndicatorRow(item.trailingIndicators)
                }
            },
        colors =
            if (selected) {
                ListItemDefaults.colors(containerColor = MaterialTheme.colorScheme.secondaryContainer)
            } else {
                ListItemDefaults.colors()
            },
        modifier =
            modifier
                .fillMaxWidth()
                .semantics(mergeDescendants = true) {
                    contentDescription = item.accessibilityLabel
                    this.selected = selected
                }
                .clickable(onClick = onClick),
    ) {
        Text(item.title, maxLines = 1, overflow = TextOverflow.Ellipsis)
    }
}
