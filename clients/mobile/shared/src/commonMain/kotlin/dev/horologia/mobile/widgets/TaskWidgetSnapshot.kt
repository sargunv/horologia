package dev.horologia.mobile.widgets

import dev.horologia.mobile.domain.MobileTask
import dev.horologia.mobile.domain.SessionScope
import kotlinx.serialization.Serializable
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

const val TASK_WIDGET_SNAPSHOT_VERSION: Int = 1
const val DEFAULT_TASK_WIDGET_ROW_LIMIT: Int = 12

@Serializable
data class TaskWidgetRowV1(
    val id: String,
    val spaceSlug: String,
    val title: String,
    val due: String?,
    val status: String,
)

@Serializable
data class TaskWidgetSnapshotV1(
    val version: Int = TASK_WIDGET_SNAPSHOT_VERSION,
    val serverId: String,
    val accountId: String,
    val generatedAt: String,
    val taskCount: Int,
    val hasMore: Boolean,
    val rows: List<TaskWidgetRowV1>,
)

fun projectTaskWidgetSnapshotV1(
    scope: SessionScope,
    tasks: List<MobileTask>,
    generatedAt: String,
    hasMore: Boolean = false,
    limit: Int = DEFAULT_TASK_WIDGET_ROW_LIMIT,
): TaskWidgetSnapshotV1 {
    require(limit >= 0) { "Widget row limit must not be negative" }

    return TaskWidgetSnapshotV1(
        serverId = scope.serverId,
        accountId = scope.accountId,
        generatedAt = generatedAt,
        taskCount = tasks.size,
        hasMore = hasMore || tasks.size > limit,
        rows = tasks.take(limit).map { task ->
            TaskWidgetRowV1(
                id = task.id,
                spaceSlug = task.spaceSlug,
                title = task.title,
                due = task.dueText,
                status = task.status,
            )
        },
    )
}

object TaskWidgetSnapshotJson {
    private val json = Json {
        encodeDefaults = true
        explicitNulls = true
        ignoreUnknownKeys = true
    }

    fun encode(snapshot: TaskWidgetSnapshotV1): String = json.encodeToString(snapshot)

    fun decode(encoded: String): TaskWidgetSnapshotV1 =
        json.decodeFromString<TaskWidgetSnapshotV1>(encoded).also { snapshot ->
            require(snapshot.version == TASK_WIDGET_SNAPSHOT_VERSION) {
                "Unsupported task widget snapshot version: ${snapshot.version}"
            }
        }
}
