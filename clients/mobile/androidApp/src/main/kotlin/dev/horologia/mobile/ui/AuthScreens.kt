package dev.horologia.mobile.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.consumeWindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.LiveRegionMode
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.liveRegion
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.tooling.preview.Preview
import androidx.compose.ui.unit.dp
import dev.horologia.mobile.R
import dev.horologia.mobile.designsystem.ErrorBlock
import dev.horologia.mobile.designsystem.HorologiaTheme
import dev.horologia.mobile.designsystem.LoadingRow
import dev.horologia.mobile.runtime.MobileAppState
import dev.horologia.mobile.runtime.MobileSessionPhase

@Composable
fun BootstrapScreen(state: MobileAppState, onRetry: () -> Unit) {
    Scaffold { innerPadding ->
        Box(
            modifier =
                Modifier
                    .fillMaxSize()
                    .padding(innerPadding)
                    .consumeWindowInsets(innerPadding)
                    .padding(24.dp),
            contentAlignment = Alignment.Center,
        ) {
            Column(
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.spacedBy(16.dp),
            ) {
                Text(
                    text = stringResource(R.string.app_name),
                    style = MaterialTheme.typography.displaySmall,
                    color = MaterialTheme.colorScheme.primary,
                    modifier = Modifier.semantics { heading() },
                )
                val error = state.error
                if (error != null && !state.loading.bootstrap) {
                    ErrorBlock(
                        title = stringResource(R.string.sign_in_error_title),
                        error = error,
                        onRetry = onRetry,
                    )
                } else {
                    LoadingRow(text = stringResource(R.string.status_preparing))
                }
            }
        }
    }
}

@Composable
fun SignInScreen(state: MobileAppState, onConnect: (String) -> Unit) {
    var serverUrl by rememberSaveable { mutableStateOf(state.server.baseUrl) }
    val busy = state.phase == MobileSessionPhase.AUTHORIZING || state.loading.bootstrap
    Scaffold { innerPadding ->
        Column(
            modifier =
                Modifier
                    .fillMaxSize()
                    .padding(innerPadding)
                    .consumeWindowInsets(innerPadding)
                    .verticalScroll(rememberScrollState())
                    .padding(24.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            Spacer(modifier = Modifier.height(48.dp))
            Text(
                text = stringResource(R.string.app_name),
                style = MaterialTheme.typography.displaySmall,
                color = MaterialTheme.colorScheme.primary,
                modifier = Modifier.semantics { heading() },
            )
            Text(
                text = stringResource(R.string.sign_in_subtitle),
                style = MaterialTheme.typography.bodyLarge,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            OutlinedTextField(
                value = serverUrl,
                onValueChange = { serverUrl = it },
                modifier = Modifier.fillMaxWidth(),
                label = { Text(stringResource(R.string.server_url_label)) },
                singleLine = true,
                enabled = !busy,
                keyboardOptions =
                    KeyboardOptions(
                        keyboardType = KeyboardType.Uri,
                        imeAction = ImeAction.Go,
                    ),
                keyboardActions = KeyboardActions(onGo = { onConnect(serverUrl) }),
            )
            val authConfig = state.authConfig
            if (authConfig != null && authConfig.oidcEnabled) {
                Text(
                    text = stringResource(R.string.sign_in_method, authConfig.oidcLabel),
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            val error = state.error
            if (error != null && !busy) {
                Column(modifier = Modifier.semantics { liveRegion = LiveRegionMode.Polite }) {
                    Text(
                        text = stringResource(R.string.sign_in_error_title),
                        style = MaterialTheme.typography.titleSmall,
                        color = MaterialTheme.colorScheme.error,
                    )
                    Text(
                        text = error.message,
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
            Button(
                onClick = { onConnect(serverUrl) },
                modifier = Modifier.fillMaxWidth(),
                enabled = serverUrl.isNotBlank() && !busy,
            ) {
                Text(stringResource(R.string.action_connect))
            }
            if (busy) {
                LoadingRow(
                    text =
                        stringResource(
                            if (state.phase == MobileSessionPhase.AUTHORIZING) {
                                R.string.sign_in_authorizing
                            } else {
                                R.string.sign_in_connecting
                            },
                        ),
                )
            }
        }
    }
}

@Preview(showBackground = true)
@Composable
private fun SignInScreenPreview() {
    HorologiaTheme {
        SignInScreen(
            state = MobileAppState(phase = MobileSessionPhase.SIGNED_OUT),
            onConnect = {},
        )
    }
}
