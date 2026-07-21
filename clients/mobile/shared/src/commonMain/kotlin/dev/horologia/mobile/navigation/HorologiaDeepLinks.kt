package dev.horologia.mobile.navigation

import io.ktor.http.Url
import io.ktor.http.decodeURLQueryComponent
import io.ktor.http.encodeURLPathPart
import io.ktor.http.encodeURLParameter

/** A navigation request that is independent of either native shell's pane and back-stack state. */
sealed interface SemanticDestination {
    data object Tasks : SemanticDestination
    data class Task(val spaceSlug: String, val taskId: String) : SemanticDestination
    data object Recipes : SemanticDestination
    data class Recipe(val spaceSlug: String, val recipeId: String) : SemanticDestination
    data object Spaces : SemanticDestination
    data class Space(val spaceSlug: String) : SemanticDestination
    data class Search(val query: String? = null) : SemanticDestination
    data object Account : SemanticDestination

    /** OAuth callbacks are deliberately not interpreted as ordinary account navigation. */
    data class OAuthCallback(
        val state: String,
        val code: String? = null,
        val error: String? = null,
        val errorDescription: String? = null,
    ) : SemanticDestination
}

/**
 * Parses links into stable semantic identifiers for mapping into either native navigation stack.
 *
 * HTTPS links are accepted only when [expectedBaseUrl] is supplied, and must belong to that exact
 * origin and base path. App-scheme links may carry a `server` query parameter; when
 * [expectedServerId] is supplied it is required and must match. OAuth callbacks are exempt from
 * server scoping because their random `state` binds them to the pending authorization instead.
 */
object HorologiaDeepLinks {
    fun parse(
        link: String,
        expectedServerId: String? = null,
        expectedBaseUrl: String? = null,
    ): SemanticDestination? {
        if (link.isBlank() || expectedServerId?.isBlank() == true || expectedBaseUrl?.isBlank() == true) return null
        if (link.substringBefore('#').substringBefore('?').endsWith('/')) return null
        val url = try {
            Url(link)
        } catch (_: Throwable) {
            return null
        }
        if (url.fragment.isNotEmpty()) return null

        return when (url.protocol.name.lowercase()) {
            "https" -> parseServerLink(url, expectedBaseUrl ?: return null)
            APP_SCHEME -> parseAppLink(url, expectedServerId)
            else -> null
        }
    }

    fun formatServer(destination: SemanticDestination, baseUrl: String): String {
        require(destination !is SemanticDestination.OAuthCallback) { "OAuth callbacks use the app scheme" }
        val base = validatedBaseUrl(baseUrl) ?: throw IllegalArgumentException("baseUrl must be an HTTPS URL without query or fragment")
        val route = route(destination)
        val basePath = base.encodedPath.trimEnd('/')
        return buildString {
            append("https://")
            append(base.host)
            if (base.port != 443) append(':').append(base.port)
            append(basePath)
            append(route.path)
            appendQuery(route.query)
        }
    }

    fun formatApp(destination: SemanticDestination, serverId: String? = null): String {
        if (destination is SemanticDestination.OAuthCallback) {
            require(destination.state.isNotBlank()) { "OAuth state must not be blank" }
            require((destination.code != null) xor (destination.error != null)) { "OAuth callback must contain exactly one of code or error" }
            require(destination.code?.isNotBlank() != false && destination.error?.isNotBlank() != false) { "OAuth result must not be blank" }
            require(destination.error != null || destination.errorDescription == null) { "errorDescription requires error" }
            return buildString {
                append("horologia://oauth/callback")
                appendQuery(
                    listOfNotNull(
                        "state" to destination.state,
                        destination.code?.let { "code" to it },
                        destination.error?.let { "error" to it },
                        destination.errorDescription?.let { "error_description" to it },
                    ),
                )
            }
        }

        require(serverId == null || serverId.isNotBlank()) { "serverId must not be blank" }
        val route = route(destination)
        val segments = route.path.removePrefix("/").split('/')
        return buildString {
            append("horologia://")
            append(segments.first())
            segments.drop(1).forEach { append('/').append(it) }
            appendQuery(route.query + listOfNotNull(serverId?.let { "server" to it }))
        }
    }

    private fun parseServerLink(url: Url, expectedBaseUrl: String): SemanticDestination? {
        val base = validatedBaseUrl(expectedBaseUrl) ?: return null
        if (url.host.lowercase() != base.host.lowercase() || url.port != base.port) return null
        if (!validEncoded(url.encodedPath) || !validEncoded(url.encodedQuery)) return null
        val relativePath = removeBasePath(url.encodedPath, base.encodedPath) ?: return null
        val query = parseQuery(url.encodedQuery) ?: return null
        return parseRoute(pathSegments(relativePath) ?: return null, query)
    }

    private fun parseAppLink(url: Url, expectedServerId: String?): SemanticDestination? {
        if (
            !validEncoded(url.encodedPath) ||
            !validEncoded(url.encodedQuery) ||
            (url.encodedPath.length > 1 && url.encodedPath.endsWith('/'))
        ) return null
        val query = parseQuery(url.encodedQuery) ?: return null
        if (url.host.equals("oauth", ignoreCase = true)) {
            if (url.encodedPath != "/callback") return null
            return parseOAuthCallback(query)
        }

        val serverValues = query["server"]
        if (serverValues != null && (serverValues.size != 1 || serverValues.single().isBlank())) return null
        val server = serverValues?.single() ?: if (expectedServerId == null) null else return null
        if (expectedServerId != null && server != expectedServerId) return null
        val routeQuery = query - "server"
        val encodedSegments = buildList {
            add(url.host)
            addAll(url.encodedPath.split('/').filter { it.isNotEmpty() })
        }
        val segments = decodeSegments(encodedSegments) ?: return null
        return parseRoute(segments, routeQuery)
    }

    private fun parseRoute(segments: List<String>, query: Map<String, List<String>>): SemanticDestination? {
        if (segments.any { it.isBlank() }) return null
        val allowedQuery = when (segments.firstOrNull()) {
            "search" -> setOf("q")
            else -> emptySet()
        }
        if (query.keys.any { it !in allowedQuery }) return null

        return when {
            segments == listOf("tasks") -> SemanticDestination.Tasks
            segments.size == 3 && segments[0] == "tasks" -> SemanticDestination.Task(segments[1], segments[2])
            segments.size == 4 && segments[0] == "spaces" && segments[2] == "tasks" -> SemanticDestination.Task(segments[1], segments[3])
            segments == listOf("recipes") -> SemanticDestination.Recipes
            segments.size == 3 && segments[0] == "recipes" -> SemanticDestination.Recipe(segments[1], segments[2])
            segments.size == 4 && segments[0] == "spaces" && segments[2] == "recipes" -> SemanticDestination.Recipe(segments[1], segments[3])
            segments == listOf("spaces") -> SemanticDestination.Spaces
            segments.size == 2 && segments[0] == "spaces" -> SemanticDestination.Space(segments[1])
            segments == listOf("search") -> {
                val values = query["q"]
                if (values != null && values.size != 1) null else SemanticDestination.Search(values?.single())
            }
            segments == listOf("account") || segments == listOf("settings") -> SemanticDestination.Account
            else -> null
        }
    }

    private fun parseOAuthCallback(query: Map<String, List<String>>): SemanticDestination.OAuthCallback? {
        if (query.keys.any { it !in OAUTH_QUERY_KEYS }) return null
        val state = query.single("state")?.takeIf { it.isNotBlank() } ?: return null
        val code = query.single("code")?.takeIf { it.isNotBlank() }
        val error = query.single("error")?.takeIf { it.isNotBlank() }
        val description = query.single("error_description")
        if ((code != null) == (error != null) || (description != null && error == null)) return null
        return SemanticDestination.OAuthCallback(state, code, error, description)
    }

    private fun route(destination: SemanticDestination): Route = when (destination) {
        SemanticDestination.Tasks -> Route("/tasks")
        is SemanticDestination.Task -> Route("/tasks/${destination.spaceSlug.pathPart()}/${destination.taskId.pathPart()}")
        SemanticDestination.Recipes -> Route("/recipes")
        is SemanticDestination.Recipe -> Route("/recipes/${destination.spaceSlug.pathPart()}/${destination.recipeId.pathPart()}")
        SemanticDestination.Spaces -> Route("/spaces")
        is SemanticDestination.Space -> Route("/spaces/${destination.spaceSlug.pathPart()}")
        is SemanticDestination.Search -> Route("/search", listOfNotNull(destination.query?.let { "q" to it }))
        SemanticDestination.Account -> Route("/account")
        is SemanticDestination.OAuthCallback -> error("OAuth callback has a dedicated formatter")
    }

    private fun String.pathPart(): String {
        require(isNotBlank()) { "Destination identifiers must not be blank" }
        return encodeURLPathPart()
    }

    private fun validatedBaseUrl(value: String): Url? {
        val url = try {
            Url(value)
        } catch (_: Throwable) {
            return null
        }
        if (url.protocol.name.lowercase() != "https" || url.host.isBlank() || url.encodedQuery.isNotEmpty() || url.fragment.isNotEmpty()) return null
        if (!validEncoded(url.encodedPath)) return null
        return url
    }

    private fun removeBasePath(path: String, basePath: String): String? {
        val normalizedBase = basePath.trimEnd('/').ifEmpty { "" }
        if (normalizedBase.isEmpty()) return path
        if (path == normalizedBase) return "/"
        if (!path.startsWith("$normalizedBase/")) return null
        return path.removePrefix(normalizedBase)
    }

    private fun pathSegments(path: String): List<String>? {
        if (!path.startsWith('/')) return null
        if (path.length > 1 && path.endsWith('/')) return null
        return decodeSegments(path.split('/').drop(1))
    }

    private fun decodeSegments(segments: List<String>): List<String>? = try {
        segments.map { encoded ->
            if (!validEncoded(encoded)) return null
            encoded.decodeURLQueryComponent(plusIsSpace = false)
        }
    } catch (_: Throwable) {
        null
    }

    private fun parseQuery(encodedQuery: String): Map<String, List<String>>? {
        if (encodedQuery.isEmpty()) return emptyMap()
        val result = linkedMapOf<String, MutableList<String>>()
        return try {
            encodedQuery.split('&').forEach { field ->
                if (field.isEmpty()) return null
                val separator = field.indexOf('=')
                val encodedName = if (separator < 0) field else field.substring(0, separator)
                val encodedValue = if (separator < 0) "" else field.substring(separator + 1)
                if (!validEncoded(encodedName) || !validEncoded(encodedValue)) return null
                val name = encodedName.decodeURLQueryComponent(plusIsSpace = true)
                val value = encodedValue.decodeURLQueryComponent(plusIsSpace = true)
                if (name.isEmpty()) return null
                result.getOrPut(name) { mutableListOf() }.add(value)
            }
            result
        } catch (_: Throwable) {
            null
        }
    }

    private fun Map<String, List<String>>.single(name: String): String? = get(name)?.singleOrNull()

    private fun validEncoded(value: String): Boolean {
        var index = 0
        while (index < value.length) {
            if (value[index] == '%') {
                if (index + 2 >= value.length || !value[index + 1].isHex() || !value[index + 2].isHex()) return false
                index += 3
            } else {
                index++
            }
        }
        return true
    }

    private fun Char.isHex(): Boolean = this in '0'..'9' || this in 'a'..'f' || this in 'A'..'F'

    private fun StringBuilder.appendQuery(parameters: List<Pair<String, String>>) {
        if (parameters.isEmpty()) return
        append('?')
        parameters.forEachIndexed { index, (name, value) ->
            if (index > 0) append('&')
            append(name.encodeURLParameter())
            append('=')
            append(value.encodeURLParameter(spaceToPlus = false))
        }
    }

    private data class Route(val path: String, val query: List<Pair<String, String>> = emptyList())

    private const val APP_SCHEME = "horologia"
    private val OAUTH_QUERY_KEYS = setOf("state", "code", "error", "error_description")
}
