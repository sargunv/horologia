package dev.horologia.mobile.designsystem

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Close
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.InputChip
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.unit.dp
import dev.horologia.mobile.R

/**
 * Always-live tag editor: each tag is an [InputChip] whose tap removes it,
 * and the trailing field adds a tag on IME Done. Every add/remove reports the
 * full new list through [onTagsChange] immediately — there is no draft.
 */
@OptIn(ExperimentalLayoutApi::class, ExperimentalMaterial3Api::class)
@Composable
fun TagsField(
    tags: List<String>,
    onTagsChange: (List<String>) -> Unit,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
) {
    var draft by remember { mutableStateOf("") }

    fun addTag() {
        val tag = draft.trim()
        if (tag.isNotEmpty() && tags.none { it.equals(tag, ignoreCase = true) }) {
            onTagsChange(tags + tag)
        }
        draft = ""
    }

    FlowRow(
        modifier = modifier,
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalArrangement = Arrangement.spacedBy(0.dp, Alignment.CenterVertically),
    ) {
        tags.forEach { tag ->
            InputChip(
                selected = false,
                onClick = { onTagsChange(tags - tag) },
                label = { Text(tag) },
                enabled = enabled,
                trailingIcon = {
                    Icon(
                        Icons.Filled.Close,
                        contentDescription = stringResource(R.string.action_remove_tag, tag),
                        modifier = Modifier.size(18.dp),
                    )
                },
            )
        }
        if (enabled) {
            BasicTextField(
                value = draft,
                onValueChange = { draft = it },
                textStyle =
                    MaterialTheme.typography.bodyLarge.copy(
                        color = MaterialTheme.colorScheme.onSurface,
                    ),
                singleLine = true,
                keyboardOptions = KeyboardOptions(imeAction = ImeAction.Done),
                keyboardActions = KeyboardActions(onDone = { addTag() }),
                modifier = Modifier.widthIn(min = 120.dp),
                decorationBox = { innerTextField ->
                    if (draft.isEmpty()) {
                        Text(
                            text = stringResource(R.string.tags_add_hint),
                            style = MaterialTheme.typography.bodyLarge,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                    innerTextField()
                },
            )
        }
    }
}
