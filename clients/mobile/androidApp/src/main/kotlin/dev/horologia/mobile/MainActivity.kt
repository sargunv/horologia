package dev.horologia.mobile

import android.content.Intent
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.activity.viewModels
import dev.horologia.mobile.auth.AndroidAuthorizationHandoff
import dev.horologia.mobile.designsystem.HorologiaTheme

/** Single-activity host for the Compose shell. */
class MainActivity : ComponentActivity() {
    private companion object {
        const val EXTRA_SEMANTIC_LINK_DISPATCHED = "dev.horologia.mobile.semanticLinkDispatched"
    }

    private var authorizationLaunchRegistration: AutoCloseable? = null
    private val viewModel: HorologiaViewModel by viewModels()

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        dispatchIntent(intent)
        setContent {
            HorologiaTheme {
                HorologiaApp(viewModel)
            }
        }
    }

    override fun onStart() {
        super.onStart()
        authorizationLaunchRegistration =
            AndroidAuthorizationHandoff.installLaunchHandler { authorizationIntent ->
                startActivity(authorizationIntent)
            }
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        setIntent(intent)
        dispatchIntent(intent)
    }

    private fun dispatchIntent(intent: Intent) {
        if (AndroidAuthorizationHandoff.handleCallback(intent)) return
        if (intent.getBooleanExtra(EXTRA_SEMANTIC_LINK_DISPATCHED, false)) return
        val data = intent.data ?: return
        val isOAuthCallback =
            data.scheme.equals("horologia", ignoreCase = true) &&
                data.host.equals("oauth", ignoreCase = true) &&
                data.path == "/callback"
        val isSemanticLink =
            data.scheme.equals("horologia", ignoreCase = true) ||
                data.scheme.equals("https", ignoreCase = true)
        if (intent.action == Intent.ACTION_VIEW && isSemanticLink && !isOAuthCallback) {
            intent.putExtra(EXTRA_SEMANTIC_LINK_DISPATCHED, true)
            viewModel.handleDeepLink(data.toString())
        }
    }

    override fun onStop() {
        authorizationLaunchRegistration?.close()
        authorizationLaunchRegistration = null
        super.onStop()
    }

}
