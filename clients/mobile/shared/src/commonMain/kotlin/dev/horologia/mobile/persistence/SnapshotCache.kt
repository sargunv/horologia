package dev.horologia.mobile.persistence

import app.cash.sqldelight.db.SqlDriver
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

data class CachedSnapshot<T>(
    val items: List<T>,
    val generatedAt: Long,
    val cursor: String?,
    val hasMore: Boolean,
)

data class CachedTask(
    val id: String,
    val spaceSlug: String,
    val title: String,
    val description: String?,
    val status: String,
    val effort: String?,
    val priority: String?,
    val dueText: String?,
    val tags: List<String>,
)

data class CachedSpace(
    val slug: String,
    val name: String,
)

data class CachedRecipe(
    val id: String,
    val spaceSlug: String,
    val title: String,
    val description: String?,
    val tags: List<String>,
)

data class CachedSearchResult(
    val id: String,
    val spaceSlug: String,
    val title: String,
    val kind: String,
    val detail: String?,
)

interface SnapshotStore {
    fun replaceTasks(
        serverId: String,
        accountId: String,
        items: List<CachedTask>,
        generatedAt: Long,
        cursor: String?,
        hasMore: Boolean,
    )

    fun readTasks(serverId: String, accountId: String): CachedSnapshot<CachedTask>?

    fun replaceSpaces(
        serverId: String,
        accountId: String,
        items: List<CachedSpace>,
        generatedAt: Long,
        cursor: String?,
        hasMore: Boolean,
    )

    fun readSpaces(serverId: String, accountId: String): CachedSnapshot<CachedSpace>?

    fun replaceRecipes(
        serverId: String,
        accountId: String,
        items: List<CachedRecipe>,
        generatedAt: Long,
        cursor: String?,
        hasMore: Boolean,
    )

    fun readRecipes(serverId: String, accountId: String): CachedSnapshot<CachedRecipe>?

    fun replaceSearch(
        serverId: String,
        accountId: String,
        query: String,
        items: List<CachedSearchResult>,
        generatedAt: Long,
        cursor: String?,
        hasMore: Boolean,
    )

    fun readSearch(serverId: String, accountId: String, query: String): CachedSnapshot<CachedSearchResult>?
    fun clearAccount(serverId: String, accountId: String)
    fun clearServer(serverId: String)
}

expect class DatabaseDriverFactory {
    fun createDriver(): SqlDriver
}

class SnapshotCache(
    driver: SqlDriver,
    private val json: Json = Json,
) : SnapshotStore {
    private val database = HorologiaDatabase(driver)
    private val queries = database.horologiaCacheQueries

    override fun replaceTasks(
        serverId: String,
        accountId: String,
        items: List<CachedTask>,
        generatedAt: Long,
        cursor: String?,
        hasMore: Boolean,
    ) {
        database.transaction {
            queries.deleteTasks(serverId, accountId)
            items.forEachIndexed { position, task ->
                queries.insertTask(
                    server_id = serverId,
                    account_id = accountId,
                    position = position.toLong(),
                    id = task.id,
                    space_slug = task.spaceSlug,
                    title = task.title,
                    description = task.description,
                    status = task.status,
                    effort = task.effort,
                    priority = task.priority,
                    due_text = task.dueText,
                    tags_json = json.encodeToString(task.tags),
                )
            }
            queries.upsertMetadata(serverId, accountId, TASKS, generatedAt, cursor, hasMore.toLong())
        }
    }

    override fun readTasks(serverId: String, accountId: String): CachedSnapshot<CachedTask>? =
        database.transactionWithResult {
            metadata(serverId, accountId, TASKS)?.let { metadata ->
                CachedSnapshot(
                    items = queries.selectTasks(serverId, accountId) { id, spaceSlug, title, description,
                        status, effort, priority, dueText, tagsJson ->
                        CachedTask(
                            id = id,
                            spaceSlug = spaceSlug,
                            title = title,
                            description = description,
                            status = status,
                            effort = effort,
                            priority = priority,
                            dueText = dueText,
                            tags = json.decodeFromString(tagsJson),
                        )
                    }.executeAsList(),
                    generatedAt = metadata.generatedAt,
                    cursor = metadata.cursor,
                    hasMore = metadata.hasMore,
                )
            }
        }

    override fun replaceSpaces(
        serverId: String,
        accountId: String,
        items: List<CachedSpace>,
        generatedAt: Long,
        cursor: String?,
        hasMore: Boolean,
    ) {
        database.transaction {
            queries.deleteSpaces(serverId, accountId)
            items.forEachIndexed { position, space ->
                queries.insertSpace(serverId, accountId, position.toLong(), space.slug, space.name)
            }
            queries.upsertMetadata(serverId, accountId, SPACES, generatedAt, cursor, hasMore.toLong())
        }
    }

    override fun readSpaces(serverId: String, accountId: String): CachedSnapshot<CachedSpace>? =
        database.transactionWithResult {
            metadata(serverId, accountId, SPACES)?.let { metadata ->
                CachedSnapshot(
                    items = queries.selectSpaces(serverId, accountId, ::CachedSpace).executeAsList(),
                    generatedAt = metadata.generatedAt,
                    cursor = metadata.cursor,
                    hasMore = metadata.hasMore,
                )
            }
        }

    override fun replaceRecipes(
        serverId: String,
        accountId: String,
        items: List<CachedRecipe>,
        generatedAt: Long,
        cursor: String?,
        hasMore: Boolean,
    ) {
        database.transaction {
            queries.deleteRecipes(serverId, accountId)
            items.forEachIndexed { position, recipe ->
                queries.insertRecipe(
                    server_id = serverId,
                    account_id = accountId,
                    position = position.toLong(),
                    id = recipe.id,
                    space_slug = recipe.spaceSlug,
                    title = recipe.title,
                    description = recipe.description,
                    tags_json = json.encodeToString(recipe.tags),
                )
            }
            queries.upsertMetadata(serverId, accountId, RECIPES, generatedAt, cursor, hasMore.toLong())
        }
    }

    override fun readRecipes(serverId: String, accountId: String): CachedSnapshot<CachedRecipe>? =
        database.transactionWithResult {
            metadata(serverId, accountId, RECIPES)?.let { metadata ->
                CachedSnapshot(
                    items = queries.selectRecipes(serverId, accountId) { id, spaceSlug, title, description, tagsJson ->
                        CachedRecipe(id, spaceSlug, title, description, json.decodeFromString(tagsJson))
                    }.executeAsList(),
                    generatedAt = metadata.generatedAt,
                    cursor = metadata.cursor,
                    hasMore = metadata.hasMore,
                )
            }
        }

    override fun replaceSearch(
        serverId: String,
        accountId: String,
        query: String,
        items: List<CachedSearchResult>,
        generatedAt: Long,
        cursor: String?,
        hasMore: Boolean,
    ) {
        database.transaction {
            queries.deleteSearchResults(serverId, accountId, query)
            items.forEachIndexed { position, result ->
                queries.insertSearchResult(
                    server_id = serverId,
                    account_id = accountId,
                    query = query,
                    position = position.toLong(),
                    id = result.id,
                    space_slug = result.spaceSlug,
                    title = result.title,
                    kind = result.kind,
                    detail = result.detail,
                )
            }
            queries.upsertSearchMetadata(serverId, accountId, query, generatedAt, cursor, hasMore.toLong())
        }
    }

    override fun readSearch(serverId: String, accountId: String, query: String): CachedSnapshot<CachedSearchResult>? =
        database.transactionWithResult {
            val metadata = queries.selectSearchMetadata(serverId, accountId, query) { generatedAt, cursor, hasMore ->
                Metadata(generatedAt, cursor, hasMore != 0L)
            }.executeAsOneOrNull() ?: return@transactionWithResult null
            CachedSnapshot(
                items = queries.selectSearchResults(serverId, accountId, query, ::CachedSearchResult).executeAsList(),
                generatedAt = metadata.generatedAt,
                cursor = metadata.cursor,
                hasMore = metadata.hasMore,
            )
        }

    override fun clearAccount(serverId: String, accountId: String) {
        database.transaction {
            queries.clearAccountTasks(serverId, accountId)
            queries.clearAccountSpaces(serverId, accountId)
            queries.clearAccountRecipes(serverId, accountId)
            queries.clearAccountSearchResults(serverId, accountId)
            queries.clearAccountMetadata(serverId, accountId)
            queries.clearAccountSearchMetadata(serverId, accountId)
        }
    }

    override fun clearServer(serverId: String) {
        database.transaction {
            queries.clearServerTasks(serverId)
            queries.clearServerSpaces(serverId)
            queries.clearServerRecipes(serverId)
            queries.clearServerSearchResults(serverId)
            queries.clearServerMetadata(serverId)
            queries.clearServerSearchMetadata(serverId)
        }
    }

    private fun metadata(serverId: String, accountId: String, kind: String): Metadata? =
        queries.selectMetadata(serverId, accountId, kind) { generatedAt, cursor, hasMore ->
            Metadata(generatedAt, cursor, hasMore != 0L)
        }.executeAsOneOrNull()

    private data class Metadata(
        val generatedAt: Long,
        val cursor: String?,
        val hasMore: Boolean,
    )

    private fun Boolean.toLong(): Long = if (this) 1L else 0L

    private companion object {
        const val TASKS = "my_tasks"
        const val SPACES = "spaces"
        const val RECIPES = "recipes"
    }
}
