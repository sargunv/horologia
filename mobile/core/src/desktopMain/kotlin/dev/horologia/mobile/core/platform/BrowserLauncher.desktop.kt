package dev.horologia.mobile.core.platform

import com.sun.net.httpserver.HttpServer
import java.awt.Desktop
import java.net.InetSocketAddress
import java.net.URI
import kotlin.coroutines.resume
import kotlin.coroutines.resumeWithException
import kotlinx.coroutines.suspendCancellableCoroutine

/**
 * Desktop browser launcher: binds a loopback listener on a free port at first `redirectUri()` call,
 * then opens the system browser on the authorize URL and suspends until the browser returns to
 * `http://127.0.0.1:<port>/`.
 *
 * Listener is shut down 5 seconds after a valid callback so the browser tab's "You may close this
 * tab" response has time to render.
 */
actual class BrowserLauncher {
  private var server: HttpServer? = null

  actual fun redirectUri(): String =
    "http://${ensureServer().address.hostString}:${ensureServer().address.port}/"

  @Synchronized
  private fun ensureServer(): HttpServer {
    val existing = server
    if (existing != null) return existing
    val fresh = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
    fresh.start()
    server = fresh
    return fresh
  }

  actual suspend fun launchAndAwait(authorizeUrl: String): String {
    val srv = ensureServer()
    return suspendCancellableCoroutine { continuation ->
        val context =
          srv.createContext("/") { exchange ->
            val requestUri = exchange.requestURI
            val callbackUrl =
              "${redirectUri().trimEnd('/')}${requestUri.rawPath}" +
                if (requestUri.rawQuery != null) "?${requestUri.rawQuery}" else ""
            val body = "You may close this window.".toByteArray(Charsets.UTF_8)
            exchange.responseHeaders.add("Content-Type", "text/plain; charset=utf-8")
            exchange.sendResponseHeaders(200, body.size.toLong())
            exchange.responseBody.use { it.write(body) }

            if (continuation.isActive) {
              continuation.resume(callbackUrl)
            }
          }

        continuation.invokeOnCancellation {
          runCatching { srv.removeContext(context) }
          scheduleShutdown(srv = srv)
        }

        try {
          Desktop.getDesktop().browse(URI(authorizeUrl))
        } catch (t: Throwable) {
          runCatching { srv.removeContext(context) }
          if (continuation.isActive) continuation.resumeWithException(t)
        }
      }
      .also {
        // Leave the server up briefly so the browser tab's response finishes streaming.
        scheduleShutdown(srv = srv)
      }
  }

  private fun scheduleShutdown(srv: HttpServer) {
    Thread {
        try {
          Thread.sleep(5000)
        } catch (_: InterruptedException) {
          Thread.currentThread().interrupt()
        }
        srv.stop(1)
        if (server === srv) server = null
      }
      .apply { isDaemon = true }
      .start()
  }
}
