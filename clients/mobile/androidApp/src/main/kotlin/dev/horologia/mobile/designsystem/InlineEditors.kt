package dev.horologia.mobile.designsystem

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.platform.LocalSoftwareKeyboardController
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.LiveRegionMode
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.liveRegion
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.TextRange
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.TextFieldValue
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import dev.horologia.mobile.R
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

private const val DESCRIPTION_AUTOSAVE_DEBOUNCE_MILLIS = 1_500L

/**
 * Tap-to-edit headline. The heading text swaps to a borderless field with
 * focus and IME shown; IME Done or focus loss commits a non-blank, changed
 * title through [onCommit], while system back reverts. An empty title is
 * never committed.
 *
 * [style] defaults to `headlineSmall`; pass `LocalTextStyle.current` inside a
 * `TopAppBar` title slot to inherit the app bar's (animated collapse) style.
 */
@Composable
fun EditableHeadline(
    title: String,
    onCommit: (String) -> Unit,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
    style: TextStyle = MaterialTheme.typography.headlineSmall,
    maxLines: Int = Int.MAX_VALUE,
    overflow: TextOverflow = TextOverflow.Clip,
) {
    var editing by remember { mutableStateOf(false) }
    if (!editing) {
        Text(
            text = title,
            style = style,
            maxLines = maxLines,
            overflow = overflow,
            modifier =
                modifier
                    .fillMaxWidth()
                    .semantics { heading() }
                    .clickable(enabled = enabled) { editing = true },
        )
        return
    }

    var draft by remember {
        mutableStateOf(TextFieldValue(title, selection = TextRange(title.length)))
    }
    var hadFocus by remember { mutableStateOf(false) }
    var discardOnFocusLoss by remember { mutableStateOf(false) }
    val focusRequester = remember { FocusRequester() }
    val keyboard = LocalSoftwareKeyboardController.current

    fun revert() {
        // Disposing the focused field fires a focus-loss event; flag it so
        // that event doesn't commit the discarded draft.
        discardOnFocusLoss = true
        editing = false
    }

    fun commit() {
        val committed = draft.text.trim()
        editing = false
        if (committed.isNotEmpty() && committed != title) onCommit(committed)
    }

    BackHandler(onBack = ::revert)
    LaunchedEffect(Unit) {
        focusRequester.requestFocus()
        keyboard?.show()
    }
    BasicTextField(
        value = draft,
        onValueChange = { draft = it },
        textStyle = style.copy(color = MaterialTheme.colorScheme.onSurface),
        singleLine = true,
        keyboardOptions = KeyboardOptions(imeAction = ImeAction.Done),
        keyboardActions = KeyboardActions(onDone = { commit() }),
        modifier =
            modifier
                .fillMaxWidth()
                .focusRequester(focusRequester)
                .onFocusChanged { state ->
                    if (state.isFocused) {
                        hadFocus = true
                    } else if (hadFocus && !discardOnFocusLoss) {
                        commit()
                    }
                },
    )
}

/**
 * Borderless multiline description that saves itself: edits commit through
 * [onSave] after a short debounce, or immediately on focus loss. [onSave]
 * returns false when the write failed, which shows a brief inline error
 * caption until the next edit. External [description] changes resync the
 * field only while it is not focused, so in-flight typing is never clobbered.
 */
@Composable
fun AutosaveDescriptionField(
    description: String,
    onSave: suspend (String) -> Boolean,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
) {
    val scope = rememberCoroutineScope()
    var draft by remember { mutableStateOf(description) }
    var focused by remember { mutableStateOf(false) }
    var saveFailed by remember { mutableStateOf(false) }

    LaunchedEffect(description) {
        if (!focused && draft != description) draft = description
    }
    LaunchedEffect(draft, focused) {
        if (focused && draft != description) {
            delay(DESCRIPTION_AUTOSAVE_DEBOUNCE_MILLIS)
            saveFailed = !onSave(draft)
        }
    }

    Column(modifier = modifier) {
        BasicTextField(
            value = draft,
            onValueChange = {
                draft = it
                saveFailed = false
            },
            textStyle =
                MaterialTheme.typography.bodyLarge.copy(
                    color = MaterialTheme.colorScheme.onSurface,
                ),
            enabled = enabled,
            modifier =
                Modifier
                    .fillMaxWidth()
                    .onFocusChanged { state ->
                        val wasFocused = focused
                        focused = state.isFocused
                        if (wasFocused && !state.isFocused && draft != description) {
                            scope.launch { saveFailed = !onSave(draft) }
                        }
                    },
            decorationBox = { innerTextField ->
                if (draft.isEmpty()) {
                    Text(
                        text = stringResource(R.string.task_field_description),
                        style = MaterialTheme.typography.bodyLarge,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
                innerTextField()
            },
        )
        if (saveFailed) {
            Text(
                text = stringResource(R.string.description_save_error),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.error,
                modifier =
                    Modifier
                        .padding(top = 4.dp)
                        .semantics { liveRegion = LiveRegionMode.Polite },
            )
        }
    }
}
