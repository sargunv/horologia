package dev.horologia.mobile.repositories

import dev.horologia.mobile.domain.MobileAuthConfig
import dev.horologia.mobile.domain.MobileProfileUpdate
import dev.horologia.mobile.domain.MobileRecipe
import dev.horologia.mobile.domain.MobileRecipeUpdate
import dev.horologia.mobile.domain.MobileSearchResult
import dev.horologia.mobile.domain.MobileSpace
import dev.horologia.mobile.domain.MobileTask
import dev.horologia.mobile.domain.MobileTaskEffortDefinition
import dev.horologia.mobile.domain.MobileTaskPriorityDefinition
import dev.horologia.mobile.domain.MobileTaskStatusDefinition
import dev.horologia.mobile.domain.MobileTaskUpdate
import dev.horologia.mobile.domain.MobileUser
import dev.horologia.mobile.domain.Page
import dev.horologia.mobile.domain.ServerScope
import dev.horologia.mobile.domain.SessionScope

interface MobileRepository {
    suspend fun authConfig(scope: ServerScope): MobileAuthConfig

    suspend fun currentUser(scope: SessionScope): MobileUser

    suspend fun myTasks(scope: SessionScope, cursor: String? = null, limit: Int? = null): Page<MobileTask>

    suspend fun task(scope: SessionScope, spaceSlug: String, taskId: String): MobileTask

    suspend fun updateTask(
        scope: SessionScope,
        spaceSlug: String,
        taskId: String,
        update: MobileTaskUpdate,
    ): MobileTask

    suspend fun spaces(scope: SessionScope): List<MobileSpace>

    suspend fun spaceTasks(
        scope: SessionScope,
        spaceSlug: String,
        cursor: String? = null,
        limit: Int? = null,
    ): Page<MobileTask>

    suspend fun taskStatuses(scope: SessionScope, spaceSlug: String): List<MobileTaskStatusDefinition>

    suspend fun taskEffortLevels(scope: SessionScope, spaceSlug: String): List<MobileTaskEffortDefinition>

    suspend fun taskPriorityLevels(scope: SessionScope, spaceSlug: String): List<MobileTaskPriorityDefinition>

    suspend fun spaceRecipes(
        scope: SessionScope,
        spaceSlug: String,
        cursor: String? = null,
        limit: Int? = null,
    ): Page<MobileRecipe>

    suspend fun recipe(scope: SessionScope, spaceSlug: String, recipeId: String): MobileRecipe

    suspend fun updateRecipe(
        scope: SessionScope,
        spaceSlug: String,
        recipeId: String,
        update: MobileRecipeUpdate,
    ): MobileRecipe

    suspend fun search(
        scope: SessionScope,
        query: String,
        spaceSlug: String? = null,
        limit: Int? = null,
    ): List<MobileSearchResult>

    suspend fun updateProfile(scope: SessionScope, update: MobileProfileUpdate): MobileUser
}
