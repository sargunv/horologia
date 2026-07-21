package dev.horologia.mobile.widgets

import dev.horologia.mobile.domain.MobileTask
import dev.horologia.mobile.domain.SessionScope
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertFailsWith
import kotlin.test.assertNull
import kotlin.test.assertTrue

class TaskWidgetSnapshotTest {
    @Test
    fun projectionCarriesOnlyStableScopeAndCompactTaskFields() {
        val snapshot = projectTaskWidgetSnapshotV1(
            scope = scope(serverId = "server-a", accountId = "account-7"),
            tasks = listOf(task(id = "task-1", due = "2026-07-21")),
            generatedAt = "2026-07-20T12:34:56Z",
        )

        assertEquals(TASK_WIDGET_SNAPSHOT_VERSION, snapshot.version)
        assertEquals("server-a", snapshot.serverId)
        assertEquals("account-7", snapshot.accountId)
        assertEquals("2026-07-20T12:34:56Z", snapshot.generatedAt)
        assertEquals(1, snapshot.taskCount)
        assertFalse(snapshot.hasMore)
        assertEquals(
            TaskWidgetRowV1(
                id = "task-1",
                spaceSlug = "kitchen",
                title = "Task task-1",
                due = "2026-07-21",
                status = "open",
            ),
            snapshot.rows.single(),
        )
    }

    @Test
    fun projectionPreservesOrderCountsBeforeTruncationAndReportsMoreRows() {
        val tasks = (1..14).map { index -> task(id = "task-$index") }

        val snapshot = projectTaskWidgetSnapshotV1(
            scope = scope(),
            tasks = tasks,
            generatedAt = "instant",
        )

        assertEquals(14, snapshot.taskCount)
        assertEquals((1..12).map { "task-$it" }, snapshot.rows.map { it.id })
        assertTrue(snapshot.hasMore)
    }

    @Test
    fun upstreamHasMoreSurvivesWhenListDoesNotExceedLimit() {
        val snapshot = projectTaskWidgetSnapshotV1(
            scope = scope(),
            tasks = listOf(task("task-1")),
            generatedAt = "instant",
            hasMore = true,
        )

        assertTrue(snapshot.hasMore)
    }

    @Test
    fun emptyProjectionIsACompleteScopedSnapshot() {
        val snapshot = projectTaskWidgetSnapshotV1(
            scope = scope(serverId = "empty-server", accountId = "empty-account"),
            tasks = emptyList(),
            generatedAt = "2026-07-20T00:00:00Z",
        )

        assertEquals("empty-server", snapshot.serverId)
        assertEquals("empty-account", snapshot.accountId)
        assertEquals(0, snapshot.taskCount)
        assertFalse(snapshot.hasMore)
        assertTrue(snapshot.rows.isEmpty())
    }

    @Test
    fun jsonRoundTripIsDeterministicAndKeepsNullDue() {
        val snapshot = projectTaskWidgetSnapshotV1(
            scope = scope(),
            tasks = listOf(task(id = "task-1", due = null)),
            generatedAt = "2026-07-20T12:34:56Z",
        )

        val encoded = TaskWidgetSnapshotJson.encode(snapshot)
        val decoded = TaskWidgetSnapshotJson.decode(encoded)

        assertEquals(snapshot, decoded)
        assertNull(decoded.rows.single().due)
        assertEquals(encoded, TaskWidgetSnapshotJson.encode(decoded))
        assertEquals(
            "{\"version\":1,\"serverId\":\"server\",\"accountId\":\"account\",\"generatedAt\":\"2026-07-20T12:34:56Z\",\"taskCount\":1,\"hasMore\":false,\"rows\":[{\"id\":\"task-1\",\"spaceSlug\":\"kitchen\",\"title\":\"Task task-1\",\"due\":null,\"status\":\"open\"}]}",
            encoded,
        )
    }

    @Test
    fun decoderRejectsSnapshotsFromAnotherVersion() {
        val encoded = "{\"version\":2,\"serverId\":\"s\",\"accountId\":\"a\",\"generatedAt\":\"now\",\"taskCount\":0,\"hasMore\":false,\"rows\":[]}"

        assertFailsWith<IllegalArgumentException> {
            TaskWidgetSnapshotJson.decode(encoded)
        }
    }

    private fun scope(
        serverId: String = "server",
        accountId: String = "account",
    ) = SessionScope(
        serverId = serverId,
        baseUrl = "https://not-projected.example",
        accountId = accountId,
        accessToken = "secret-not-projected",
    )

    private fun task(
        id: String,
        due: String? = null,
    ) = MobileTask(
        id = id,
        spaceSlug = "kitchen",
        title = "Task $id",
        description = "description not projected",
        status = "open",
        effort = "small",
        priority = "high",
        dueText = due,
        tags = listOf("not-projected"),
    )
}
