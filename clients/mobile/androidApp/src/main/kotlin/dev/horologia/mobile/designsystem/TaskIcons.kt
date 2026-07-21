package dev.horologia.mobile.designsystem

import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Air
import androidx.compose.material.icons.outlined.ArrowUpward
import androidx.compose.material.icons.outlined.Autorenew
import androidx.compose.material.icons.outlined.Block
import androidx.compose.material.icons.outlined.Bolt
import androidx.compose.material.icons.outlined.Cancel
import androidx.compose.material.icons.outlined.Check
import androidx.compose.material.icons.outlined.CheckCircle
import androidx.compose.material.icons.outlined.Circle
import androidx.compose.material.icons.outlined.CrisisAlert
import androidx.compose.material.icons.outlined.DoneAll
import androidx.compose.material.icons.outlined.Eco
import androidx.compose.material.icons.outlined.ErrorOutline
import androidx.compose.material.icons.outlined.Flag
import androidx.compose.material.icons.automirrored.outlined.HelpOutline
import androidx.compose.material.icons.outlined.HourglassEmpty
import androidx.compose.material.icons.outlined.KeyboardArrowUp
import androidx.compose.material.icons.outlined.KeyboardDoubleArrowUp
import androidx.compose.material.icons.outlined.RadioButtonChecked
import androidx.compose.material.icons.outlined.RocketLaunch
import androidx.compose.material.icons.outlined.Schedule
import androidx.compose.material.icons.outlined.SignalCellularAlt
import androidx.compose.material.icons.outlined.SignalCellularAlt1Bar
import androidx.compose.material.icons.outlined.SignalCellularAlt2Bar
import androidx.compose.material.icons.outlined.Speed
import androidx.compose.material.icons.outlined.Terrain
import androidx.compose.material.icons.outlined.Timer
import androidx.compose.material.icons.outlined.WarningAmber
import androidx.compose.material.icons.outlined.Whatshot
import androidx.compose.ui.graphics.vector.ImageVector

/**
 * Resolves Lucide icon tokens (kebab-case, as stored in task status, effort,
 * and priority settings) to Material symbols.
 *
 * Covers the canonical suggested sets plus common Lucide aliases/renames.
 * Unknown tokens resolve to a neutral question mark, matching the web
 * fallback ([CircleHelp]).
 */
private val LucideToMaterial: Map<String, ImageVector> =
    mapOf(
        // Status (canonical set: circle, circle-dot, loader, circle-check, circle-x, ban)
        "circle" to Icons.Outlined.Circle,
        "circle-dot" to Icons.Outlined.RadioButtonChecked,
        "loader" to Icons.Outlined.Autorenew,
        "loader-circle" to Icons.Outlined.Autorenew,
        "circle-check" to Icons.Outlined.CheckCircle,
        "circle-check-big" to Icons.Outlined.CheckCircle,
        "check-circle" to Icons.Outlined.CheckCircle,
        "circle-x" to Icons.Outlined.Cancel,
        "x-circle" to Icons.Outlined.Cancel,
        "ban" to Icons.Outlined.Block,
        "circle-alert" to Icons.Outlined.ErrorOutline,
        "alert-circle" to Icons.Outlined.ErrorOutline,
        "circle-help" to Icons.AutoMirrored.Outlined.HelpOutline,
        "help-circle" to Icons.AutoMirrored.Outlined.HelpOutline,
        "check" to Icons.Outlined.Check,
        "check-check" to Icons.Outlined.DoneAll,
        "clock" to Icons.Outlined.Schedule,
        "hourglass" to Icons.Outlined.HourglassEmpty,
        // Effort (canonical set: feather, leaf, gauge, mountain, flame, rocket)
        "feather" to Icons.Outlined.Air,
        "leaf" to Icons.Outlined.Eco,
        "gauge" to Icons.Outlined.Speed,
        "mountain" to Icons.Outlined.Terrain,
        "flame" to Icons.Outlined.Whatshot,
        "fire" to Icons.Outlined.Whatshot,
        "rocket" to Icons.Outlined.RocketLaunch,
        "timer" to Icons.Outlined.Timer,
        "zap" to Icons.Outlined.Bolt,
        // Priority (canonical set: signal-low, signal-medium, signal-high, flag, alert-triangle, siren)
        "signal-low" to Icons.Outlined.SignalCellularAlt1Bar,
        "signal-medium" to Icons.Outlined.SignalCellularAlt2Bar,
        "signal-high" to Icons.Outlined.SignalCellularAlt,
        "flag" to Icons.Outlined.Flag,
        "alert-triangle" to Icons.Outlined.WarningAmber,
        "triangle-alert" to Icons.Outlined.WarningAmber,
        "siren" to Icons.Outlined.CrisisAlert,
        "arrow-up" to Icons.Outlined.ArrowUpward,
        "chevron-up" to Icons.Outlined.KeyboardArrowUp,
        "chevrons-up" to Icons.Outlined.KeyboardDoubleArrowUp,
    )

/** Neutral fallback for tokens without a Material equivalent. */
private val FallbackIcon: ImageVector = Icons.AutoMirrored.Outlined.HelpOutline

/** Map a Lucide kebab-case icon token to a Material [ImageVector]. */
fun taskIcon(token: String): ImageVector = LucideToMaterial[token.trim().lowercase()] ?: FallbackIcon
