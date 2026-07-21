package dev.horologia.mobile.domain

data class SessionScope(
    val serverId: String,
    val baseUrl: String,
    val accountId: String,
    val accessToken: String,
)

data class ServerScope(
    val serverId: String,
    val baseUrl: String,
)

data class MobileUser(
    val id: String,
    val email: String,
    val name: String,
    val isOwner: Boolean,
)

data class MobileTask(
    val id: String,
    val spaceSlug: String,
    val title: String,
    val description: String,
    val status: String,
    val effort: String?,
    val priority: String?,
    val dueText: String?,
    val tags: List<String>,
)

enum class TaskStatusCategory {
    INITIAL,
    INTERMEDIATE,
    COMPLETION,
    NEUTRAL,
}

data class MobileTaskStatusDefinition(
    val label: String,
    val category: TaskStatusCategory,
    val iconToken: String,
)

data class MobileTaskEffortDefinition(
    val label: String,
    val iconToken: String,
)

data class MobileTaskPriorityDefinition(
    val label: String,
    val iconToken: String,
)

data class MobileTaskVisualMetadata(
    val statuses: List<MobileTaskStatusDefinition> = emptyList(),
    val effortLevels: List<MobileTaskEffortDefinition> = emptyList(),
    val priorityLevels: List<MobileTaskPriorityDefinition> = emptyList(),
)

enum class TaskListIndicatorKind {
    PRIORITY,
    EFFORT,
}

data class TaskListIndicator(
    val kind: TaskListIndicatorKind,
    val label: String,
    val iconToken: String,
)

data class TaskListItemModel(
    val title: String,
    val dueText: String?,
    val statusLabel: String,
    val statusCategory: TaskStatusCategory,
    val statusIconToken: String,
    val trailingIndicators: List<TaskListIndicator>,
    val accessibilityLabel: String,
)

data class MobileRecipe(
    val id: String,
    val spaceSlug: String,
    val title: String,
    val description: String,
    val tags: List<String>,
)

data class MobileSpace(
    val slug: String,
    val name: String,
)

data class MobileSearchResult(
    val id: String,
    val spaceSlug: String,
    val title: String,
    val kind: String,
    val detail: String,
)

data class Page<T>(
    val items: List<T>,
    val nextCursor: String?,
)

data class MobileAuthConfig(
    val oidcEnabled: Boolean,
    val oidcLabel: String,
    val oidcAutoRedirect: Boolean,
    val passwordEnabled: Boolean,
)

sealed interface PatchField<out T> {
    data object Absent : PatchField<Nothing>
    data object Null : PatchField<Nothing>
    data class Value<T>(val value: T) : PatchField<T>
}

data class MobileTaskDue(
    val date: String,
    val timezone: String,
)

data class MobileOverdueActionRule(
    val afterDays: Int?,
    val action: String,
    val status: String? = null,
)

data class MobileTaskUpdate(
    val title: String? = null,
    val description: String? = null,
    val status: String? = null,
    val effort: PatchField<String> = PatchField.Absent,
    val priority: PatchField<String> = PatchField.Absent,
    val recurrenceType: String? = null,
    val recurrenceRule: PatchField<String> = PatchField.Absent,
    val assigneeIds: List<String>? = null,
    val rotationPool: List<String>? = null,
    val tags: List<String>? = null,
    val due: PatchField<MobileTaskDue> = PatchField.Absent,
    val overdueActionRule: PatchField<MobileOverdueActionRule> = PatchField.Absent,
)

data class MobileRecipeYield(
    val amount: Double,
    val unit: String,
)

data class MobileRecipeIngredient(
    val item: String,
    val quantity: Double? = null,
    val quantityMax: Double? = null,
    val unit: String? = null,
)

data class MobileRecipeIngredientSection(
    val title: String? = null,
    val ingredients: List<MobileRecipeIngredient>,
)

data class MobileRecipeInstructionSection(
    val title: String? = null,
    val steps: List<String>,
)

data class MobileRecipeUpdate(
    val title: String? = null,
    val description: String? = null,
    val yield: PatchField<MobileRecipeYield> = PatchField.Absent,
    val prepMinutes: PatchField<Int> = PatchField.Absent,
    val cookMinutes: PatchField<Int> = PatchField.Absent,
    val tags: List<String>? = null,
    val ingredientSections: List<MobileRecipeIngredientSection>? = null,
    val instructionSections: List<MobileRecipeInstructionSection>? = null,
)

data class MobileProfileUpdate(
    val name: String? = null,
    val email: String? = null,
)

class RepositoryException(
    message: String,
    val statusCode: Int? = null,
    cause: Throwable? = null,
) : Exception(message, cause)
