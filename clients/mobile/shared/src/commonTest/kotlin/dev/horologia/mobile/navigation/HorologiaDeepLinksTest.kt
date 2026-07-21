package dev.horologia.mobile.navigation

import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFailsWith
import kotlin.test.assertNull

class HorologiaDeepLinksTest {
    private val server = "https://clock.example/horologia"
    private val serverId = "production-eu"

    @Test
    fun serverLinksRoundTripEveryNavigationDestination() {
        val destinations = listOf(
            SemanticDestination.Tasks,
            SemanticDestination.Task("family", "T-42"),
            SemanticDestination.Recipes,
            SemanticDestination.Recipe("kitchen", "R-9"),
            SemanticDestination.Spaces,
            SemanticDestination.Space("home"),
            SemanticDestination.Search(),
            SemanticDestination.Search("bread timer"),
            SemanticDestination.Account,
        )

        destinations.forEach { destination ->
            val link = HorologiaDeepLinks.formatServer(destination, server)
            assertEquals(destination, HorologiaDeepLinks.parse(link, expectedBaseUrl = server), link)
        }
    }

    @Test
    fun scopedAppLinksRoundTripEveryNavigationDestination() {
        val destinations = listOf(
            SemanticDestination.Tasks,
            SemanticDestination.Task("family", "T-42"),
            SemanticDestination.Recipes,
            SemanticDestination.Recipe("kitchen", "R-9"),
            SemanticDestination.Spaces,
            SemanticDestination.Space("home"),
            SemanticDestination.Search("bread timer"),
            SemanticDestination.Account,
        )

        destinations.forEach { destination ->
            val link = HorologiaDeepLinks.formatApp(destination, serverId)
            assertEquals(destination, HorologiaDeepLinks.parse(link, expectedServerId = serverId), link)
        }
    }

    @Test
    fun canonicalWebDetailRoutesMapToStableIdentifiers() {
        assertEquals(
            SemanticDestination.Task("family", "T-42"),
            HorologiaDeepLinks.parse(
                "https://clock.example/horologia/spaces/family/tasks/T-42",
                expectedBaseUrl = server,
            ),
        )
        assertEquals(
            SemanticDestination.Recipe("kitchen", "R-9"),
            HorologiaDeepLinks.parse(
                "https://clock.example/horologia/spaces/kitchen/recipes/R-9",
                expectedBaseUrl = server,
            ),
        )
        assertEquals(
            SemanticDestination.Account,
            HorologiaDeepLinks.parse("https://clock.example/horologia/settings", expectedBaseUrl = server),
        )
    }

    @Test
    fun encodedIdentifiersAndSearchQueryAreDecodedAndReencoded() {
        val task = SemanticDestination.Task("meal plans", "task/one ✓")
        val recipe = SemanticDestination.Recipe("café", "recipe?#2")
        val search = SemanticDestination.Search("tomato soup & bread/rolls")

        listOf(task, recipe, search).forEach { destination ->
            val serverLink = HorologiaDeepLinks.formatServer(destination, server)
            assertEquals(destination, HorologiaDeepLinks.parse(serverLink, expectedBaseUrl = server))
            val appLink = HorologiaDeepLinks.formatApp(destination, serverId)
            assertEquals(destination, HorologiaDeepLinks.parse(appLink, expectedServerId = serverId))
        }

        assertEquals(
            SemanticDestination.Search("two words + one"),
            HorologiaDeepLinks.parse(
                "https://clock.example/horologia/search?q=two+words+%2B+one",
                expectedBaseUrl = server,
            ),
        )
    }

    @Test
    fun malformedOrAmbiguousLinksAreRejected() {
        val links = listOf(
            "not a URL",
            "horologia://tasks/family",
            "horologia://tasks/family/T-1/extra",
            "horologia://spaces/",
            "horologia://spaces/%ZZ",
            "horologia://search?q=%E0%A4%A",
            "horologia://search?q=one&q=two",
            "horologia://tasks?server=one&server=two",
            "horologia://tasks?server=",
            "horologia://account?pane=security",
            "https://clock.example/horologia/unknown",
            "https://clock.example/horologia/tasks/family/T-1#pane",
            "http://clock.example/horologia/tasks",
        )

        links.forEach { link ->
            assertNull(
                HorologiaDeepLinks.parse(
                    link,
                    expectedServerId = if (link.startsWith("horologia:")) null else serverId,
                    expectedBaseUrl = server,
                ),
                link,
            )
        }
    }

    @Test
    fun serverOriginAndBasePathMustMatch() {
        assertNull(
            HorologiaDeepLinks.parse(
                "https://other.example/horologia/tasks/family/T-1",
                expectedBaseUrl = server,
            ),
        )
        assertNull(
            HorologiaDeepLinks.parse(
                "https://clock.example/not-horologia/tasks/family/T-1",
                expectedBaseUrl = server,
            ),
        )
        assertNull(
            HorologiaDeepLinks.parse(
                "https://clock.example.evil/horologia/tasks/family/T-1",
                expectedBaseUrl = server,
            ),
        )
        assertNull(HorologiaDeepLinks.parse("https://clock.example/horologia/tasks/family/T-1"))
    }

    @Test
    fun appLinkServerScopeMustMatchWhenExpected() {
        val link = HorologiaDeepLinks.formatApp(SemanticDestination.Space("home"), serverId)
        assertEquals(
            SemanticDestination.Space("home"),
            HorologiaDeepLinks.parse(link, expectedServerId = serverId),
        )
        assertNull(HorologiaDeepLinks.parse(link, expectedServerId = "staging"))
        assertNull(HorologiaDeepLinks.parse("horologia://spaces/home", expectedServerId = serverId))
    }

    @Test
    fun oauthCallbacksAreSeparatedFromAccountAndServerRoutes() {
        val success = SemanticDestination.OAuthCallback(state = "state/with + symbols", code = "code?42")
        val failure = SemanticDestination.OAuthCallback(
            state = "state-2",
            error = "access_denied",
            errorDescription = "The user declined",
        )

        listOf(success, failure).forEach { callback ->
            val link = HorologiaDeepLinks.formatApp(callback)
            assertEquals(callback, HorologiaDeepLinks.parse(link, expectedServerId = serverId))
        }

        assertNull(
            HorologiaDeepLinks.parse(
                "https://clock.example/horologia/oauth/callback?state=s&code=c",
                expectedBaseUrl = server,
            ),
        )
        assertNull(HorologiaDeepLinks.parse("horologia://oauth/callback?state=s"))
        assertNull(HorologiaDeepLinks.parse("horologia://oauth/callback?state=s&code=c&error=denied"))
        assertNull(HorologiaDeepLinks.parse("horologia://account?state=s&code=c"))
        assertFailsWith<IllegalArgumentException> { HorologiaDeepLinks.formatServer(success, server) }
    }
}
