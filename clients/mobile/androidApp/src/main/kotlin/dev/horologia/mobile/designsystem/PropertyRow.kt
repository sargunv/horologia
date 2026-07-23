package dev.horologia.mobile.designsystem

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.material3.Icon
import androidx.compose.material3.ListItem
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.style.TextOverflow
import dev.horologia.mobile.R

/**
 * Material 3 [ListItem]-based tappable property row: an overline label above
 * the current value, with an optional leading icon. A null [value] renders
 * [placeholder] dimmed (e.g. "Not set"). A null [onClick] makes the row
 * read-only. Clickable rows merge into a single accessibility node.
 */
@Composable
fun PropertyRow(
    label: String,
    value: String?,
    onClick: (() -> Unit)?,
    modifier: Modifier = Modifier,
    icon: ImageVector? = null,
    placeholder: String = stringResource(R.string.property_not_set),
) {
    ListItem(
        overlineContent = { Text(label) },
        leadingContent =
            icon?.let {
                @Composable {
                    Icon(imageVector = it, contentDescription = null)
                }
            },
        modifier =
            modifier
                .fillMaxWidth()
                .semantics(mergeDescendants = true) {}
                .then(
                    if (onClick != null) {
                        Modifier.clickable(onClick = onClick)
                    } else {
                        Modifier
                    },
                ),
    ) {
        Text(
            text = value ?: placeholder,
            color =
                if (value != null) {
                    MaterialTheme.colorScheme.onSurface
                } else {
                    MaterialTheme.colorScheme.onSurfaceVariant
                },
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
    }
}
