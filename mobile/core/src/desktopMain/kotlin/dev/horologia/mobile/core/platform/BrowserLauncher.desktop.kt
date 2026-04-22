package dev.horologia.mobile.core.platform

import com.sun.net.httpserver.HttpServer
import java.awt.Desktop
import java.io.IOException
import java.net.BindException
import java.net.InetSocketAddress
import java.net.URI
import kotlin.coroutines.resume
import kotlin.coroutines.resumeWithException
import kotlin.time.Duration.Companion.minutes
import kotlinx.coroutines.TimeoutCancellationException
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlinx.coroutines.withTimeout

/**
 * Desktop browser launcher: binds a loopback listener on a free port at first `redirectUri()` call,
 * then opens the system browser on the authorize URL and suspends until the browser returns to
 * `http://127.0.0.1:<port>/` with an OAuth callback (must carry at least `state=` in the query so
 * speculative browser prefetches don't hijack the continuation).
 *
 * The wait is bounded by a 5-minute timeout so an abandoned browser session doesn't leave the VM
 * stuck. The listener is shut down 5 seconds after a valid callback so the browser tab's "You may
 * close this tab" response has time to render.
 */
actual class BrowserLauncher {
  @Volatile private var server: HttpServer? = null
  @Volatile private var shuttingDown: Boolean = false

  actual fun redirectUri(): String =
    "http://${ensureServer().address.hostString}:${ensureServer().address.port}/"

  @Synchronized
  private fun ensureServer(): HttpServer {
    val existing = server
    if (existing != null) return existing
    val fresh =
      try {
        HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
      } catch (e: BindException) {
        throw BrowserFailedException(
          "Couldn't start local sign-in listener. Another app may be using the port."
        )
      } catch (e: IOException) {
        throw BrowserFailedException(
          "Couldn't start local sign-in listener. Another app may be using the port."
        )
      }
    fresh.start()
    server = fresh
    shuttingDown = false
    return fresh
  }

  actual suspend fun launchAndAwait(authorizeUrl: String): String {
    val srv = ensureServer()
    return try {
        withTimeout(5.minutes) {
          suspendCancellableCoroutine<String> { continuation ->
            val context =
              srv.createContext("/") { exchange ->
                val remote = exchange.remoteAddress.address
                if (remote == null || !remote.isLoopbackAddress) {
                  val body = "Forbidden".toByteArray(Charsets.UTF_8)
                  exchange.sendResponseHeaders(403, body.size.toLong())
                  exchange.responseBody.use { it.write(body) }
                  return@createContext
                }

                val requestUri = exchange.requestURI
                val rawQuery = requestUri.rawQuery
                // Ignore speculative favicon / prefetch hits — only accept a request that
                // carries the OAuth `state=` parameter; missing `code=` is handled downstream.
                if (rawQuery == null || !rawQuery.contains("state=")) {
                  val body = "Not found".toByteArray(Charsets.UTF_8)
                  exchange.sendResponseHeaders(404, body.size.toLong())
                  exchange.responseBody.use { it.write(body) }
                  return@createContext
                }

                val callbackUrl = "${redirectUri().trimEnd('/')}${requestUri.rawPath}?${rawQuery}"
                val body = "You may close this window.".toByteArray(Charsets.UTF_8)
                exchange.responseHeaders.add("Content-Type", "text/plain; charset=utf-8")
                exchange.sendResponseHeaders(200, body.size.toLong())
                exchange.responseBody.use { it.write(body) }

                if (continuation.isActive) {
                  runCatching { srv.removeContext("/") }
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
        }
      } catch (_: TimeoutCancellationException) {
        runCatching { srv.removeContext("/") }
        scheduleShutdown(srv = srv)
        throw BrowserCancelledException("Sign-in timed out. Try again.")
      }
      .also {
        // Leave the server up briefly so the browser tab's response finishes streaming.
        scheduleShutdown(srv = srv)
      }
  }

  private fun scheduleShutdown(srv: HttpServer) {
    synchronized(this) {
      if (shuttingDown) return
      shuttingDown = true
    }
    Thread {
        try {
          Thread.sleep(5000)
        } catch (_: InterruptedException) {
          Thread.currentThread().interrupt()
        }
        srv.stop(1)
        synchronized(this) {
          if (server === srv) server = null
          shuttingDown = false
        }
      }
      .apply { isDaemon = true }
      .start()
  }
}
