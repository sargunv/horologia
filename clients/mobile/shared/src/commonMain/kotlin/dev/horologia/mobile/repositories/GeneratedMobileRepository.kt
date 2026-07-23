package dev.horologia.mobile.repositories

import dev.horologia.mobile.api.apis.AuthApi
import dev.horologia.mobile.api.apis.RecipesApi
import dev.horologia.mobile.api.apis.SpacesApi
import dev.horologia.mobile.api.apis.TaskEffortLevelsApi
import dev.horologia.mobile.api.apis.TaskPriorityLevelsApi
import dev.horologia.mobile.api.apis.TaskStatusesApi
import dev.horologia.mobile.api.apis.TasksApi
import dev.horologia.mobile.api.apis.UsersApi
import dev.horologia.mobile.api.infrastructure.HttpResponse
import dev.horologia.mobile.api.models.ApiError
import dev.horologia.mobile.api.models.PatchField as WirePatchField
import dev.horologia.mobile.api.models.Recipe as WireRecipe
import dev.horologia.mobile.api.models.RecipeIngredientInput
import dev.horologia.mobile.api.models.RecipeIngredientSectionInput
import dev.horologia.mobile.api.models.RecipeInstructionSectionInput
import dev.horologia.mobile.api.models.RecipeStepInput
import dev.horologia.mobile.api.models.RecipeSummary
import dev.horologia.mobile.api.models.RecipeUpdateWire
import dev.horologia.mobile.api.models.RecipeYield
import dev.horologia.mobile.api.models.Space as WireSpace
import dev.horologia.mobile.api.models.TaskEffortLevel
import dev.horologia.mobile.api.models.TaskPriorityLevel
import dev.horologia.mobile.api.models.TaskStatus
import dev.horologia.mobile.api.models.Task as WireTask
import dev.horologia.mobile.api.models.TaskDue
import dev.horologia.mobile.api.models.TaskOverdueAction
import dev.horologia.mobile.api.models.TaskOverdueActionRule
import dev.horologia.mobile.api.models.TaskRecurrenceType
import dev.horologia.mobile.api.models.TaskUpdateWire
import dev.horologia.mobile.api.models.User as WireUser
import dev.horologia.mobile.api.models.UserUpdate
import dev.horologia.mobile.domain.MobileAuthConfig
import dev.horologia.mobile.domain.MobileOverdueActionRule
import dev.horologia.mobile.domain.MobileProfileUpdate
import dev.horologia.mobile.domain.MobileRecipe
import dev.horologia.mobile.domain.MobileRecipeUpdate
import dev.horologia.mobile.domain.MobileSearchResult
import dev.horologia.mobile.domain.MobileSpace
import dev.horologia.mobile.domain.MobileTask
import dev.horologia.mobile.domain.MobileTaskEffortDefinition
import dev.horologia.mobile.domain.MobileTaskPriorityDefinition
import dev.horologia.mobile.domain.MobileTaskStatusDefinition
import dev.horologia.mobile.domain.MobileTaskDue
import dev.horologia.mobile.domain.MobileTaskUpdate
import dev.horologia.mobile.domain.MobileUser
import dev.horologia.mobile.domain.Page
import dev.horologia.mobile.domain.PatchField
import dev.horologia.mobile.domain.RepositoryException
import dev.horologia.mobile.domain.ServerScope
import dev.horologia.mobile.domain.SessionScope
import dev.horologia.mobile.domain.TaskStatusCategory
import io.ktor.client.HttpClient
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.client.statement.bodyAsText
import io.ktor.serialization.kotlinx.json.json
import kotlinx.datetime.LocalDate
import kotlinx.serialization.json.Json
import kotlin.coroutines.cancellation.CancellationException

class GeneratedMobileRepository : MobileRepository {
    override suspend fun authConfig(scope: ServerScope): MobileAuthConfig =
        withAuthApi(scope) { api ->
            val config = api.webAuthConfig().checked()
            MobileAuthConfig(
                oidcEnabled = config.oidc.enabled,
                oidcLabel = config.oidc.label,
                oidcAutoRedirect = config.oidc.autoRedirect,
                passwordEnabled = config.password.enabled,
            )
        }

    override suspend fun currentUser(scope: SessionScope): MobileUser =
        withApis(scope) { it.users.usersMe().checked().toDomain() }

    override suspend fun myTasks(scope: SessionScope, cursor: String?, limit: Int?): Page<MobileTask> =
        withApis(scope) {
            val page = it.users.userTasksList(scope.accountId, cursor, limit).checked()
            Page(page.items.map(WireTask::toDomain), page.nextCursor)
        }

    override suspend fun task(scope: SessionScope, spaceSlug: String, taskId: String): MobileTask =
        withApis(scope) { it.tasks.spaceTasksRead(spaceSlug, taskId).checked().toDomain() }

    override suspend fun updateTask(
        scope: SessionScope,
        spaceSlug: String,
        taskId: String,
        update: MobileTaskUpdate,
    ): MobileTask = withApis(scope) {
        it.tasks.spaceTasksUpdate(spaceSlug, taskId, update.toWire()).checked().toDomain()
    }

    override suspend fun deleteTask(scope: SessionScope, spaceSlug: String, taskId: String) =
        withApis(scope) {
            it.tasks.spaceTasksDelete(spaceSlug, taskId).checked()
        }

    override suspend fun spaces(scope: SessionScope): List<MobileSpace> =
        withApis(scope) { it.spaces.spacesList().checked().items.map(WireSpace::toDomain) }

    override suspend fun spaceTasks(
        scope: SessionScope,
        spaceSlug: String,
        cursor: String?,
        limit: Int?,
    ): Page<MobileTask> = withApis(scope) {
        val page = it.tasks.spaceTasksList(spaceSlug, cursor, limit).checked()
        Page(page.items.map(WireTask::toDomain), page.nextCursor)
    }

    override suspend fun taskStatuses(
        scope: SessionScope,
        spaceSlug: String,
    ): List<MobileTaskStatusDefinition> = withApis(scope) {
        it.taskStatuses.spaceTaskStatusesList(spaceSlug).checked().items.map(TaskStatus::toDomain)
    }

    override suspend fun taskEffortLevels(
        scope: SessionScope,
        spaceSlug: String,
    ): List<MobileTaskEffortDefinition> = withApis(scope) {
        it.taskEffortLevels.spaceTaskEffortLevelsList(spaceSlug).checked().items.map(TaskEffortLevel::toDomain)
    }

    override suspend fun taskPriorityLevels(
        scope: SessionScope,
        spaceSlug: String,
    ): List<MobileTaskPriorityDefinition> = withApis(scope) {
        it.taskPriorityLevels.spaceTaskPriorityLevelsList(spaceSlug).checked().items.map(TaskPriorityLevel::toDomain)
    }

    override suspend fun spaceRecipes(
        scope: SessionScope,
        spaceSlug: String,
        cursor: String?,
        limit: Int?,
    ): Page<MobileRecipe> = withApis(scope) {
        val page = it.recipes.spaceRecipesList(spaceSlug, cursor, limit).checked()
        Page(page.items.map(RecipeSummary::toDomain), page.nextCursor)
    }

    override suspend fun recipe(scope: SessionScope, spaceSlug: String, recipeId: String): MobileRecipe =
        withApis(scope) { it.recipes.spaceRecipesRead(spaceSlug, recipeId).checked().toDomain() }

    override suspend fun updateRecipe(
        scope: SessionScope,
        spaceSlug: String,
        recipeId: String,
        update: MobileRecipeUpdate,
    ): MobileRecipe = withApis(scope) {
        it.recipes.spaceRecipesUpdate(spaceSlug, recipeId, update.toWire()).checked().toDomain()
    }

    override suspend fun search(
        scope: SessionScope,
        query: String,
        spaceSlug: String?,
        limit: Int?,
    ): List<MobileSearchResult> = withApis(scope) { apis ->
        val tasks = apis.tasks.tasksSearch(query, spaceSlug, limit = limit).checked().items.map {
            MobileSearchResult(it.id, it.spaceSlug, it.title, "task", it.status)
        }
        val recipes = apis.recipes.recipesSearch(query, spaceSlug, limit).checked().items.map {
            MobileSearchResult(it.id, it.spaceSlug, it.name, "recipe", it.tags.joinToString(", "))
        }
        tasks + recipes
    }

    override suspend fun updateProfile(scope: SessionScope, update: MobileProfileUpdate): MobileUser =
        withApis(scope) {
            it.users.usersUpdate(
                scope.accountId,
                UserUpdate(name = update.name, email = update.email),
            ).checked().toDomain()
        }

    private suspend fun <T> withApis(scope: SessionScope, block: suspend (ScopedApis) -> T): T {
        scope.requireValid()
        val client = newHttpClient()
        return try {
            val baseUrl = scope.apiBaseUrl()
            val apis = ScopedApis(
                users = UsersApi(baseUrl, client),
                tasks = TasksApi(baseUrl, client),
                spaces = SpacesApi(baseUrl, client),
                recipes = RecipesApi(baseUrl, client),
                taskStatuses = TaskStatusesApi(baseUrl, client),
                taskEffortLevels = TaskEffortLevelsApi(baseUrl, client),
                taskPriorityLevels = TaskPriorityLevelsApi(baseUrl, client),
            )
            apis.setBearerToken(scope.accessToken)
            block(apis)
        } catch (error: CancellationException) {
            throw error
        } catch (error: RepositoryException) {
            throw error
        } catch (error: Throwable) {
            throw RepositoryException(error.message ?: "Repository request failed", cause = error)
        } finally {
            client.close()
        }
    }

    private suspend fun <T> withAuthApi(scope: ServerScope, block: suspend (AuthApi) -> T): T {
        scope.requireValid()
        val client = newHttpClient()
        return try {
            block(AuthApi(scope.apiBaseUrl(), client))
        } catch (error: CancellationException) {
            throw error
        } catch (error: RepositoryException) {
            throw error
        } catch (error: Throwable) {
            throw RepositoryException(error.message ?: "Repository request failed", cause = error)
        } finally {
            client.close()
        }
    }

    private suspend fun <T : Any> HttpResponse<T>.checked(): T {
        if (success) return body()
        val message = try {
            val text = response.bodyAsText()
            wireJson.decodeFromString(ApiError.serializer(), text).message
        } catch (_: Throwable) {
            response.status.description.ifBlank { "Request failed with HTTP $status" }
        }
        throw RepositoryException(message, status)
    }
}

private data class ScopedApis(
    val users: UsersApi,
    val tasks: TasksApi,
    val spaces: SpacesApi,
    val recipes: RecipesApi,
    val taskStatuses: TaskStatusesApi,
    val taskEffortLevels: TaskEffortLevelsApi,
    val taskPriorityLevels: TaskPriorityLevelsApi,
) {
    fun setBearerToken(token: String) {
        users.setBearerToken(token)
        tasks.setBearerToken(token)
        spaces.setBearerToken(token)
        recipes.setBearerToken(token)
        taskStatuses.setBearerToken(token)
        taskEffortLevels.setBearerToken(token)
        taskPriorityLevels.setBearerToken(token)
    }
}

private val wireJson = Json {
    ignoreUnknownKeys = true
    isLenient = true
}

private fun newHttpClient(): HttpClient = HttpClient {
    install(ContentNegotiation) { json(wireJson) }
}

private fun SessionScope.requireValid() {
    if (serverId.isBlank()) throw RepositoryException("serverId must not be blank")
    if (baseUrl.isBlank()) throw RepositoryException("baseUrl must not be blank")
    if (accountId.isBlank()) throw RepositoryException("accountId must not be blank")
    if (accessToken.isBlank()) throw RepositoryException("accessToken must not be blank")
}

private fun ServerScope.requireValid() {
    if (serverId.isBlank()) throw RepositoryException("serverId must not be blank")
    if (baseUrl.isBlank()) throw RepositoryException("baseUrl must not be blank")
}

private fun SessionScope.apiBaseUrl(): String = baseUrl.asApiBaseUrl()
private fun ServerScope.apiBaseUrl(): String = baseUrl.asApiBaseUrl()

private fun String.asApiBaseUrl(): String {
    val normalized = trimEnd('/')
    return if (normalized.endsWith("/api")) normalized else "$normalized/api"
}

private fun WireUser.toDomain() = MobileUser(id, email, name, isOwner)

private fun WireTask.toDomain() = MobileTask(
    id = id,
    spaceSlug = spaceSlug,
    title = title,
    description = description,
    status = status,
    effort = effort,
    priority = priority,
    dueText = due?.at?.toString(),
    tags = tags,
)

private fun WireSpace.toDomain() = MobileSpace(slug, name)

private fun TaskStatus.toDomain() = MobileTaskStatusDefinition(
    label = name,
    category = when (category.value) {
        "initial" -> TaskStatusCategory.INITIAL
        "intermediate" -> TaskStatusCategory.INTERMEDIATE
        "completion" -> TaskStatusCategory.COMPLETION
        else -> TaskStatusCategory.NEUTRAL
    },
    iconToken = icon,
)

private fun TaskEffortLevel.toDomain() = MobileTaskEffortDefinition(name, icon)

private fun TaskPriorityLevel.toDomain() = MobileTaskPriorityDefinition(name, icon)

private fun RecipeSummary.toDomain() = MobileRecipe(
    id = id,
    spaceSlug = spaceSlug,
    title = name,
    description = "",
    tags = tags,
)

private fun WireRecipe.toDomain() = MobileRecipe(
    id = id,
    spaceSlug = spaceSlug,
    title = name,
    description = description,
    tags = tags,
)

private fun MobileTaskUpdate.toWire() = TaskUpdateWire(
    title = title,
    description = description,
    status = status,
    effort = effort.toWire { it },
    priority = priority.toWire { it },
    recurrenceType = recurrenceType?.let { value ->
        TaskRecurrenceType.decode(value)
            ?: throw RepositoryException("Unknown task recurrence type: $value")
    },
    recurrenceRule = recurrenceRule.toWire { it },
    assigneeIds = assigneeIds,
    rotationPool = rotationPool,
    tags = tags,
    due = due.toWire(MobileTaskDue::toWire),
    overdueActionRule = overdueActionRule.toWire(MobileOverdueActionRule::toWire),
)

private fun MobileRecipeUpdate.toWire() = RecipeUpdateWire(
    name = title,
    description = description,
    yield = yield.toWire { RecipeYield(it.amount, it.unit) },
    prepMinutes = prepMinutes.toWire { it },
    cookMinutes = cookMinutes.toWire { it },
    tags = tags,
    ingredientSections = ingredientSections?.map { section ->
        RecipeIngredientSectionInput(
            ingredients = section.ingredients.map {
                RecipeIngredientInput(it.item, it.quantity, it.quantityMax, it.unit)
            },
            title = section.title,
        )
    },
    instructionSections = instructionSections?.map { section ->
        RecipeInstructionSectionInput(
            steps = section.steps.map(::RecipeStepInput),
            title = section.title,
        )
    },
)

private fun MobileTaskDue.toWire() = TaskDue(LocalDate.parse(date), timezone)

private fun MobileOverdueActionRule.toWire() = TaskOverdueActionRule(
    after = afterDays,
    action = TaskOverdueAction.decode(action)
        ?: throw RepositoryException("Unknown overdue action: $action"),
    status = status,
)

private inline fun <T, R> PatchField<T>.toWire(transform: (T) -> R): WirePatchField<R> =
    when (this) {
        PatchField.Absent -> WirePatchField.Absent
        PatchField.Null -> WirePatchField.Null
        is PatchField.Value -> WirePatchField.Value(transform(value))
    }
