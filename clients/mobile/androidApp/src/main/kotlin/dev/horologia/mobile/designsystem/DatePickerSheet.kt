package dev.horologia.mobile.designsystem

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.DatePicker
import androidx.compose.material3.DatePickerDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ListItem
import androidx.compose.material3.ListItemDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberDatePickerState
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import dev.horologia.mobile.R
import kotlin.time.Clock
import kotlin.time.Instant
import kotlinx.datetime.DatePeriod
import kotlinx.datetime.LocalDate
import kotlinx.datetime.TimeZone
import kotlinx.datetime.atStartOfDayIn
import kotlinx.datetime.plus
import kotlinx.datetime.todayIn
import kotlinx.datetime.toLocalDateTime

/**
 * Due-date choice sheet: relative shortcuts and an optional destructive clear
 * row above a Material [DatePicker]. Dates are ISO `yyyy-MM-dd` strings;
 * [onSelect] receives the picked date, or null when cleared. Shortcuts and
 * clear dismiss immediately; picker selection commits via the Done action.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DatePickerSheet(
    title: String,
    current: String?,
    onSelect: (String?) -> Unit,
    onDismiss: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val zone = TimeZone.currentSystemDefault()
    val today = Clock.System.todayIn(zone)
    val pickerState =
        rememberDatePickerState(
            initialSelectedDateMillis =
                current?.let {
                    LocalDate.parse(it).atStartOfDayIn(TimeZone.UTC).toEpochMilliseconds()
                },
        )
    ChoiceSheet(title = title, onDismiss = onDismiss, modifier = modifier) {
        Column(
            modifier = Modifier.verticalScroll(rememberScrollState()),
        ) {
            ShortcutRow(label = stringResource(R.string.due_shortcut_today)) {
                onSelect(today.toString())
                onDismiss()
            }
            ShortcutRow(label = stringResource(R.string.due_shortcut_tomorrow)) {
                onSelect(today.plus(DatePeriod(days = 1)).toString())
                onDismiss()
            }
            ShortcutRow(label = stringResource(R.string.due_shortcut_in_a_week)) {
                onSelect(today.plus(DatePeriod(days = 7)).toString())
                onDismiss()
            }
            if (current != null) {
                ListItem(
                    colors = ListItemDefaults.colors(containerColor = Color.Transparent),
                    modifier =
                        Modifier
                            .fillMaxWidth()
                            .clickable {
                                onSelect(null)
                                onDismiss()
                            },
                ) {
                    Text(
                        text = stringResource(R.string.due_clear),
                        color = MaterialTheme.colorScheme.error,
                    )
                }
            }
            DatePicker(
                state = pickerState,
                showModeToggle = false,
                colors = DatePickerDefaults.colors(containerColor = Color.Transparent),
            )
            Row(
                horizontalArrangement = Arrangement.End,
                verticalAlignment = Alignment.CenterVertically,
                modifier =
                    Modifier
                        .fillMaxWidth()
                        .padding(horizontal = 24.dp)
                        .padding(bottom = 16.dp),
            ) {
                TextButton(
                    onClick = {
                        val millis = pickerState.selectedDateMillis ?: return@TextButton
                        onSelect(
                            Instant
                                .fromEpochMilliseconds(millis)
                                .toLocalDateTime(TimeZone.UTC)
                                .date
                                .toString(),
                        )
                        onDismiss()
                    },
                    enabled = pickerState.selectedDateMillis != null,
                ) {
                    Text(stringResource(R.string.action_done))
                }
            }
        }
    }
}

@Composable
private fun ShortcutRow(label: String, onClick: () -> Unit) {
    ListItem(
        colors = ListItemDefaults.colors(containerColor = Color.Transparent),
        modifier =
            Modifier
                .fillMaxWidth()
                .clickable(onClick = onClick),
    ) {
        Text(label)
    }
}
