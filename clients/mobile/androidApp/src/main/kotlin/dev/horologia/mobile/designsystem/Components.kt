package dev.horologia.mobile.designsystem

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Checkbox
import androidx.compose.material3.ExperimentalMaterial3ExpressiveApi
import androidx.compose.material3.LoadingIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.SnackbarDuration
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.SnackbarResult
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.LiveRegionMode
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.liveRegion
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import dev.horologia.mobile.R
import dev.horologia.mobile.runtime.MobileAppError
import java.text.DateFormat
import java.util.Date

@OptIn(ExperimentalMaterial3ExpressiveApi::class)
@Composable
fun LoadingRow(text: String) {
    Row(
        horizontalArrangement = Arrangement.spacedBy(16.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        LoadingIndicator(
            modifier = Modifier.size(24.dp).semantics { contentDescription = text },
        )
        Text(
            text = text,
            style = MaterialTheme.typography.bodyLarge,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}

@OptIn(ExperimentalMaterial3ExpressiveApi::class)
@Composable
fun LoadingPane() {
    Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
        LoadingIndicator()
    }
}

@Composable
fun ErrorBlock(
    title: String,
    error: MobileAppError,
    onRetry: () -> Unit,
) {
    Column(
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(8.dp),
        modifier = Modifier.semantics { liveRegion = LiveRegionMode.Polite },
    ) {
        Text(
            text = title,
            style = MaterialTheme.typography.titleMedium,
            color = MaterialTheme.colorScheme.error,
        )
        Text(
            text = error.message,
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        TextButton(onClick = onRetry) {
            Text(stringResource(R.string.action_retry))
        }
    }
}

@Composable
fun ErrorPane(
    title: String,
    error: MobileAppError?,
    onRetry: () -> Unit,
) {
    Box(modifier = Modifier.fillMaxSize().padding(24.dp), contentAlignment = Alignment.Center) {
        ErrorBlock(
            title = title,
            error = error ?: MobileAppError(message = ""),
            onRetry = onRetry,
        )
    }
}

@Composable
fun EmptyPane(title: String, body: String?) {
    Box(modifier = Modifier.fillMaxSize().padding(24.dp), contentAlignment = Alignment.Center) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            Text(
                text = title,
                style = MaterialTheme.typography.titleMedium,
                color = MaterialTheme.colorScheme.onSurface,
            )
            if (body != null) {
                Text(
                    text = body,
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
    }
}

@Composable
fun PanePlaceholder(text: String) {
    Box(modifier = Modifier.fillMaxSize().padding(24.dp), contentAlignment = Alignment.Center) {
        Text(
            text = text,
            style = MaterialTheme.typography.bodyLarge,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}

/**
 * Shows [error] as an indefinite snackbar with a Retry action while it is set.
 * Cancellation (when the error clears) dismisses the snackbar.
 */
@Composable
fun ListErrorSnackbarEffect(
    snackbarHostState: SnackbarHostState,
    error: MobileAppError?,
    onRetry: () -> Unit,
) {
    val retryLabel = stringResource(R.string.action_retry)
    LaunchedEffect(error) {
        if (error != null) {
            val result =
                snackbarHostState.showSnackbar(
                    message = error.message,
                    actionLabel = retryLabel,
                    duration = SnackbarDuration.Indefinite,
                )
            if (result == SnackbarResult.ActionPerformed) onRetry()
        }
    }
}

/** Dim caption reporting when list content was generated; renders nothing when null. */
@Composable
fun UpdatedFooter(epochSeconds: Long?) {
    if (epochSeconds == null) return
    Text(
        text = stringResource(R.string.list_updated_footer, formatEpochSeconds(epochSeconds)),
        style = MaterialTheme.typography.bodySmall,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
        textAlign = TextAlign.Center,
        modifier = Modifier.fillMaxWidth().padding(16.dp),
    )
}

@OptIn(ExperimentalMaterial3ExpressiveApi::class)
@Composable
fun LoadMoreRow(loading: Boolean, onLoadMore: () -> Unit) {
    Box(
        modifier = Modifier.fillMaxWidth().padding(vertical = 8.dp),
        contentAlignment = Alignment.Center,
    ) {
        if (loading) {
            LoadingIndicator(modifier = Modifier.size(24.dp))
        } else {
            TextButton(onClick = onLoadMore) {
                Text(stringResource(R.string.action_load_more))
            }
        }
    }
}

@Composable
fun SectionHeader(text: String, modifier: Modifier = Modifier) {
    Text(
        text = text,
        style = MaterialTheme.typography.titleSmall,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
        modifier =
            modifier
                .padding(top = 8.dp)
                .semantics { heading() },
    )
}

@Composable
fun MetaRow(label: String, value: String) {
    Row(modifier = Modifier.fillMaxWidth()) {
        Text(
            text = label,
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.width(120.dp),
        )
        Text(
            text = value,
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurface,
        )
    }
}

/** Small status/role chip, e.g. a task status or the account owner badge. */
@Composable
fun InfoBadge(
    text: String,
    containerColor: Color,
    contentColor: Color,
) {
    Surface(
        color = containerColor,
        shape = MaterialTheme.shapes.small,
    ) {
        Text(
            text = text,
            style = MaterialTheme.typography.labelLarge,
            color = contentColor,
            modifier = Modifier.padding(horizontal = 12.dp, vertical = 6.dp),
        )
    }
}

@Composable
fun NullableTextField(
    value: String,
    onValueChange: (String) -> Unit,
    label: String,
    clear: Boolean,
    onClearChange: (Boolean) -> Unit,
    enabled: Boolean,
) {
    OutlinedTextField(
        value = value,
        onValueChange = onValueChange,
        modifier = Modifier.fillMaxWidth(),
        label = { Text(label) },
        singleLine = true,
        enabled = enabled && !clear,
    )
    ClearCheckbox(
        label = stringResource(R.string.field_clear_suffix, label),
        checked = clear,
        onCheckedChange = onClearChange,
        enabled = enabled,
    )
}

@Composable
fun MinutesField(
    value: String,
    onValueChange: (String) -> Unit,
    label: String,
    valid: Boolean,
    clear: Boolean,
    onClearChange: (Boolean) -> Unit,
    enabled: Boolean,
) {
    OutlinedTextField(
        value = value,
        onValueChange = onValueChange,
        modifier = Modifier.fillMaxWidth(),
        label = { Text(label) },
        singleLine = true,
        enabled = enabled && !clear,
        isError = !valid,
        supportingText = {
            if (!valid) {
                Text(stringResource(R.string.recipe_field_minutes_invalid))
            }
        },
        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
    )
    ClearCheckbox(
        label = stringResource(R.string.field_clear_suffix, label),
        checked = clear,
        onCheckedChange = onClearChange,
        enabled = enabled,
    )
}

@Composable
fun ClearCheckbox(
    label: String,
    checked: Boolean,
    onCheckedChange: (Boolean) -> Unit,
    enabled: Boolean,
) {
    Row(
        verticalAlignment = Alignment.CenterVertically,
        modifier =
            Modifier
                .fillMaxWidth()
                .clickable(enabled = enabled) { onCheckedChange(!checked) },
    ) {
        Checkbox(
            checked = checked,
            onCheckedChange = null,
            enabled = enabled,
        )
        Text(
            text = label,
            style = MaterialTheme.typography.bodyMedium,
            color =
                if (enabled) {
                    MaterialTheme.colorScheme.onSurface
                } else {
                    MaterialTheme.colorScheme.onSurfaceVariant
                },
        )
    }
}

@Composable
fun FormError(title: String, error: MobileAppError?) {
    Column(modifier = Modifier.semantics { liveRegion = LiveRegionMode.Polite }) {
        Text(
            text = title,
            style = MaterialTheme.typography.titleSmall,
            color = MaterialTheme.colorScheme.error,
        )
        if (error != null) {
            Text(
                text = error.message,
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
    }
}

private fun formatEpochSeconds(epochSeconds: Long?): String {
    if (epochSeconds == null) return ""
    val formatter = DateFormat.getDateTimeInstance(DateFormat.SHORT, DateFormat.SHORT)
    return formatter.format(Date(epochSeconds * 1_000L))
}
