package dev.horologia.mobile.designsystem

import android.os.Build
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.ExperimentalMaterial3ExpressiveApi
import androidx.compose.material3.MaterialExpressiveTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.dynamicDarkColorScheme
import androidx.compose.material3.dynamicLightColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.platform.LocalContext

/**
 * Material You theme: dynamic color on Android 12+, falling back to the
 * Material 3 baseline schemes on older devices. No brand palette. Uses the
 * Material 3 Expressive theme for expressive shapes, motion, and component
 * defaults.
 */
@OptIn(ExperimentalMaterial3ExpressiveApi::class)
@Composable
fun HorologiaTheme(content: @Composable () -> Unit) {
    val dark = isSystemInDarkTheme()
    val colorScheme =
        when {
            Build.VERSION.SDK_INT >= Build.VERSION_CODES.S -> {
                val context = LocalContext.current
                if (dark) dynamicDarkColorScheme(context) else dynamicLightColorScheme(context)
            }

            dark -> darkColorScheme()
            else -> lightColorScheme()
        }
    MaterialExpressiveTheme(colorScheme = colorScheme, content = content)
}
